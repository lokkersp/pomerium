package fileutil

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"

	"github.com/pomerium/pomerium/internal/log"
	"github.com/pomerium/pomerium/internal/signal"
)

type watcher = fsnotify.Watcher

func newWatcher() (*watcher, error) {
	return fsnotify.NewWatcher()
}

func newWatcherObject() *watcher {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil
	}
	return w
}

// A Watcher watches files for changes.
type Watcher struct {
	*signal.Signal
	mu        sync.Mutex
	filePaths map[string]bool
	W         *fsnotify.Watcher
}

// NewWatcher creates a new Watcher.
func NewWatcher() (*Watcher, error) {
	w, err := newWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		Signal:    signal.New(),
		filePaths: map[string]bool{},
		W:         w,
	}, nil
}

// AddPath: new implementation based on fsnotify library
func (watcher *Watcher) AddPath(path string) {
	initWG := sync.WaitGroup{}
	initWG.Add(1)
	// already watching
	if _, ok := watcher.filePaths[path]; ok {
		initWG.Done()
		return
	}
	ctx := log.WithContext(context.TODO(), func(c zerolog.Context) zerolog.Context {
		return c.Str("watch_file", path)
	})
	go func() {
		configFile := filepath.Clean(path)
		configDir, _ := filepath.Split(configFile)
		realConfigFile, _ := filepath.EvalSymlinks(path)

		eventsWG := sync.WaitGroup{}
		eventsWG.Add(1)
		go func() {
			for {
				select {
				case event, ok := <-watcher.W.Events:
					if !ok { // 'Events' channel is closed
						eventsWG.Done()
						return
					}
					currentConfigFile, _ := filepath.EvalSymlinks(path)
					// we only care about the config file with the following cases:
					// 1 - if the config file was modified or created
					// 2 - if the real path to the config file changed (eg: k8s ConfigMap replacement)
					const writeOrCreateMask = fsnotify.Write | fsnotify.Create
					if (filepath.Clean(event.Name) == configFile &&
						event.Op&writeOrCreateMask != 0) ||
						(currentConfigFile != "" && currentConfigFile != realConfigFile) {
						realConfigFile = currentConfigFile
						log.Info(ctx).Str("event", event.String()).Str("config", realConfigFile).Msg("filemgr: detected file change")
						watcher.Signal.Broadcast(ctx)
					} else if filepath.Clean(event.Name) == configFile &&
						event.Op&fsnotify.Remove != 0 {
						eventsWG.Done()
						return
					}

				case err, ok := <-watcher.W.Errors:
					if ok { // 'Errors' channel is not closed
						log.Printf("watcher error: %v\n", err)
					}
					eventsWG.Done()
					return
				}
			}
		}()
		err := watcher.W.Add(configDir)
		if err != nil {
			log.Error(ctx).Err(err).Msg("filemgr: error watching file path")
		}
		watcher.filePaths[path] = true
		initWG.Done()   // done initializing the watch in this go routine, so the parent routine can move on...
		eventsWG.Wait() // now, wait for event loop to end in this go-routine...
	}()
	initWG.Wait()
}

// Clear removes all watches.
func (watcher *Watcher) Clear() {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()

	for filePath, _ := range watcher.filePaths {
		//notify.Stop(ch)
		watcher.W.Remove(filePath)
		//close(ch)
		delete(watcher.filePaths, filePath)
	}
}

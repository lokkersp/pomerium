package keycloak

import (
	"context"
	"github.com/Nerzal/gocloak/v11"
	"sync"
)

type ProviderConfig struct {
	url    string
	id     string
	realm  string
	secret string
}

type Provider struct {
	cfg    ProviderConfig
	Client gocloak.GoCloak

	m     *sync.RWMutex
	token *gocloak.JWT
}

type ServiceAccount struct {
	ClientID     string
	ClientSecret string
	Realm        string
	EndpointUrl  string
}

func New(baseUrl, clientId, clientSecret, realm string) *Provider {
	return &Provider{
		cfg: ProviderConfig{
			url:    baseUrl,
			id:     clientId,
			secret: clientSecret,
			realm:  realm,
		},
		m: &sync.RWMutex{},
	}
}

func (p *Provider) getClient() gocloak.GoCloak {
	p.m.Lock()
	defer p.m.Unlock()
	if p.Client == nil {
		p.Client = gocloak.NewClient(p.cfg.url)
	}
	return p.Client
}

func (p *Provider) getToken() (*gocloak.JWT, error) {
	client := p.getClient()
	ctx := context.Background()
	token, err := client.LoginClient(ctx, p.cfg.id, p.cfg.secret, p.cfg.realm)
	if err != nil {
		return nil, err
	}
	p.token = token
	return p.token, nil
}

func (p *Provider) getUserInfo() {

}

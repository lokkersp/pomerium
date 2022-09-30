package keycloak

import (
	"context"
	"fmt"
	"github.com/Nerzal/gocloak/v11"
	"github.com/pomerium/pomerium/internal/directory"
	"sort"
	"strings"
)

// Name is the provider name.
const Name = "keycloak"

func getParts(url string) (string, string, error) {
	SEP := "/auth/realms/"
	if !strings.Contains(url, SEP) {
		return "", "", fmt.Errorf("'%s' doesn't contain required entity separator '%s'", url, SEP)
	}
	parts := strings.Split(url, SEP)
	return parts[0], parts[1], nil
}

func ParseServiceAccount(opts directory.Options) (*ServiceAccount, error) {
	endpoint, realm, err := getParts(opts.ProviderURL)
	if err != nil {
		return nil, err
	}
	return &ServiceAccount{ClientID: opts.ClientID,
		ClientSecret: opts.ClientSecret,
		Realm:        realm,
		EndpointUrl:  endpoint}, nil

}

// User returns a user's directory information.
func (p *Provider) User(ctx context.Context, userID, accessToken string) (*directory.User, error) {
	client := p.getClient()
	token, err := p.getToken()
	if err != nil {
		return nil, err
	}

	u, err := client.GetUserByID(ctx, token.AccessToken, p.cfg.realm, userID)
	if err != nil {
		return nil, err
	}

	return &directory.User{
		Id:          *u.ID,
		DisplayName: fmt.Sprintf("%s %s", *u.FirstName, *u.LastName),
		Email:       *u.Email,
		GroupIds:    *u.Groups,
	}, nil
}

func getGroupUsers(ctx context.Context, client gocloak.GoCloak, token, realm, groupId string) ([]*gocloak.User, error) {
	members, err := client.GetGroupMembers(ctx, token, realm, groupId, gocloak.GetGroupsParams{})
	if err != nil {
		return nil, err
	}
	return members, nil
}

// UserGroups returns all the users and groups in the directory.
func (p *Provider) UserGroups(ctx context.Context) ([]*directory.Group, []*directory.User, error) {
	client := p.getClient()
	token, err := p.getToken()
	if err != nil {
		return nil, nil, err
	}

	apiGroups, err := client.GetGroups(ctx, token.AccessToken, p.cfg.realm, gocloak.GetGroupsParams{})
	if err != nil {
		return nil, nil, err
	}

	directoryUserLookup := map[string]*directory.User{}
	directoryGroups := make([]*directory.Group, len(apiGroups))
	for i, ag := range apiGroups {
		dg := &directory.Group{
			Id:   *ag.ID,
			Name: *ag.Name,
		}

		apiUsers, err := getGroupUsers(ctx, client, token.AccessToken, p.cfg.realm, *ag.ID)
		if err != nil {
			return nil, nil, err
		}
		for _, u := range apiUsers {
			du, ok := directoryUserLookup[*u.ID]
			if !ok {
				du = &directory.User{
					Id:          *u.ID,
					DisplayName: fmt.Sprintf("%s %s", *u.FirstName, *u.LastName),
					Email:       *u.Email,
				}
				directoryUserLookup[*u.ID] = du
			}
			du.GroupIds = append(du.GroupIds, *ag.ID)
		}

		directoryGroups[i] = dg
	}
	sort.Slice(directoryGroups, func(i, j int) bool {
		return directoryGroups[i].Id < directoryGroups[j].Id
	})

	directoryUsers := make([]*directory.User, 0, len(directoryUserLookup))
	for _, du := range directoryUserLookup {
		directoryUsers = append(directoryUsers, du)
	}
	sort.Slice(directoryUsers, func(i, j int) bool {
		return directoryUsers[i].Id < directoryUsers[j].Id
	})

	return directoryGroups, directoryUsers, nil
}

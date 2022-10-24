package auth

import (
	"context"

	"golang.org/x/oauth2"
	goauth2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

func (s *Service) FetchUser(token *oauth2.Token) (*goauth2.Userinfo, error) {
	// Create oauth2 service
	ctx := context.Background()
	service, err := goauth2.NewService(ctx, option.WithTokenSource(s.config.TokenSource(ctx, token)))
	if err != nil {
		return nil, err
	}

	// Fetch user data with oauth2 service
	user, err := service.Userinfo.Get().Do()
	if err != nil {
		return nil, err
	}

	return user, nil
}

package auth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	goauth2 "google.golang.org/api/oauth2/v2"
)

// Service is an implementation of the AuthService interface defined in this package.
type Service struct {
	config *oauth2.Config
	key    []byte
	parser *jwt.Parser
}

func NewService(jwtKey, googleClient, googleSecret string) (*Service, error) {
	if jwtKey == "" || googleClient == "" || googleSecret == "" {
		return nil, fmt.Errorf(`%w: config cannot be ""`, ErrNotValid)
	}

	return &Service{
		config: &oauth2.Config{
			ClientID:     googleClient,
			ClientSecret: googleSecret,
			Scopes:       []string{goauth2.UserinfoEmailScope},
			Endpoint:     google.Endpoint,
		},
		key:    []byte(jwtKey),
		parser: &jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Alg()}},
	}, nil
}

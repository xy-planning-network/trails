package auth

import (
	"net/url"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/oauth2"
	goauth2 "google.golang.org/api/oauth2/v2"
)

type AuthService interface {
	AuthenticateJWT(v url.Values) (jwt.Claims, error)
	FetchUser(token *oauth2.Token) (*goauth2.Userinfo, error)
}

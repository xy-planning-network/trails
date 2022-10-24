package auth

import (
	"fmt"
	"net/url"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/oauth2"
	goauth2 "google.golang.org/api/oauth2/v2"
)

type AuthService interface {
	AuthenticateJWT(v url.Values) (AppToken, bool, error)
	FetchUser(token *oauth2.Token) (*goauth2.Userinfo, error)
}

// An AppToken is the structure to make JWT authentication requests.
type AppToken struct {
	sc              jwt.RegisteredClaims
	AccountAddress  string `json:"firmAddress"`
	AccountName     string `json:"firmName"`
	AccountPortalID string `json:"firmId"`
	IsXypnMember    bool   `json:"isXypnMember"`
	Email           string `json:"email"`
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	Phone           string `json:"phone"`
	PortalID        string `json:"portalId"`
}

// Valid asserts whether the AppToken is a valid JWT token.
func (at *AppToken) Valid() error {
	if err := at.sc.Valid(); err != nil {
		return err
	}

	for _, field := range []string{
		at.Email,
		at.PortalID,
	} {
		if field == "" {
			return fmt.Errorf("%w: missing required field", ErrNotValid)
		}
	}

	return nil
}

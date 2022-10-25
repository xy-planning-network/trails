package auth

import (
	"fmt"
	"net/url"

	"github.com/golang-jwt/jwt/v4"
)

// AuthenticateJWT decodes jwt claims from the provided query params.
// If no token is set in the params, AuthenticateJWT returns ErrUnexpected.
// Please note that the consuming party needs to pass appToken as a pointer
// so that it can be hyrdrayed by ParseWithClaims.
func (s *Service) AuthenticateJWT(v url.Values, appToken jwt.Claims) (jwt.Claims, error) {
	reqToken := v.Get("jwt")
	if reqToken == "" {
		return nil, fmt.Errorf("no jwt param set: %w", ErrNotValid)
	}

	token, err := s.parser.ParseWithClaims(reqToken, appToken, func(token *jwt.Token) (interface{}, error) {
		return s.key, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnexpected, err)
	}

	return token.Claims, nil
}

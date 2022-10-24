package auth

import (
	"fmt"
	"net/url"

	"github.com/golang-jwt/jwt/v4"
)

// AuthenticateJWT decodes an AppToken from the provided query params.
// If no token is set in the params, AuthenticateJWT returns ErrUnexpected.
// If the header can be decoded in an AppToken, but AppToken.Valid returns an error,
// AuthenticateJWT returns false but no error.
func (s *Service) AuthenticateJWT(v url.Values) (*AppToken, bool, error) {
	reqToken := v.Get("jwt")
	if reqToken == "" {
		return nil, false, fmt.Errorf("no jwt param set: %w", ErrNotValid)
	}

	at := &AppToken{}
	token, err := s.parser.ParseWithClaims(reqToken, at, func(token *jwt.Token) (interface{}, error) {
		return s.key, nil
	})

	if err != nil {
		return nil, false, fmt.Errorf("%w: %s", ErrUnexpected, err)
	}

	if claims, ok := token.Claims.(*AppToken); ok {
		return claims, true, nil
	}

	return nil, false, nil
}

package user

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
)

// jwtSign signs a token with HS256 using the configured secret and the
// provided claims. Standard claims (iss/iat/exp/nbf) are set by the
// caller via the claims map.
func jwtSign(claims map[string]any, secret string) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	return t.SignedString([]byte(secret))
}

// verifyJWT parses and validates a bearer access token. It enforces the
// issuer and a small leeway, and rejects tokens not signed with HMAC.
func verifyJWT(tokenStr, secret, issuer string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithIssuer(issuer), jwt.WithLeeway(2*time.Second))
	if err != nil {
		return nil, domainErrors.ErrInvalidToken
	}
	if !token.Valid {
		return nil, domainErrors.ErrInvalidToken
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domainErrors.ErrInvalidToken
	}
	return claims, nil
}

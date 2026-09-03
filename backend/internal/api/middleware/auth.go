package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/api/helper"
	"github.com/you/airline-ticketing/internal/pkg/contextkeys"
	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
)

// JWTAuth validates the bearer access token and populates the context
// with the authenticated user id and role. Tokens are verified against
// the configured issuer and HMAC secret.
func JWTAuth(secret, issuer string, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
				helper.JSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or malformed authorization header"})
				return
			}
			tokenStr := strings.TrimPrefix(h, "Bearer ")

			claims, err := verifyAccessToken(tokenStr, secret, issuer)
			if err != nil {
				log.Debug("auth failed", zap.Error(err))
				helper.JSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				return
			}

			ctx := context.WithValue(r.Context(), contextkeys.UserIDKey, claims.userID)
			ctx = context.WithValue(ctx, contextkeys.UserRoleKey, claims.role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type accessClaims struct {
	userID int64
	role   string
}

func verifyAccessToken(tokenStr, secret, issuer string) (accessClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithIssuer(issuer))
	if err != nil {
		return accessClaims{}, err
	}
	if !token.Valid {
		return accessClaims{}, domainErrors.ErrInvalidToken
	}
	c, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return accessClaims{}, domainErrors.ErrInvalidToken
	}
	sub, ok := c["sub"].(string)
	if !ok {
		return accessClaims{}, domainErrors.ErrInvalidToken
	}
	id, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return accessClaims{}, domainErrors.ErrInvalidToken
	}
	role, _ := c["role"].(string)
	return accessClaims{userID: id, role: role}, nil
}

// RequireRole gates a route group to one or more roles. ADMIN satisfies
// any check (role hierarchy), so an ADMIN can reach agent/customer routes.
func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := contextkeys.RoleFromContext(r.Context())
			if _, ok := allowedSet[role]; !ok && role != "ADMIN" {
				helper.JSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

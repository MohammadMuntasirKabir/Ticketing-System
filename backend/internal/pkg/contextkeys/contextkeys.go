package contextkeys

import "context"

// contextKey is an unexported-type-named string used to scope context
// values so they cannot collide with keys defined elsewhere. Both the
// auth middleware (which writes) and domain services (which read) import
// this package, keeping the key definition in one low-level place.
type contextKey string

const (
	// UserIDKey carries the authenticated user's id (int64).
	UserIDKey contextKey = "user_id"
	// UserRoleKey carries the authenticated user's role (string).
	UserRoleKey contextKey = "user_role"
)

// UserIDFromContext extracts a user id set by the auth middleware.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	return extractInt64(ctx, UserIDKey)
}

// RoleFromContext extracts the user role set by the auth middleware.
func RoleFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(UserRoleKey)
	if r, ok := v.(string); ok {
		return r, true
	}
	return "", false
}

func extractInt64(ctx context.Context, k contextKey) (int64, bool) {
	v := ctx.Value(k)
	if id, ok := v.(int64); ok {
		return id, true
	}
	return 0, false
}

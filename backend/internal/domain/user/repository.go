package user

import (
	"context"
	"time"
)

// Repository abstracts user persistence. Implementations: postgres (prod),
// in-memory (tests). The password hash is stored and returned so the
// service owns hashing concerns.
type Repository interface {
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, u *User) (int64, error)
	UpdateProfile(ctx context.Context, id int64, upd ProfileUpdate) error
	UpdateRole(ctx context.Context, id int64, role Role) error
	List(ctx context.Context, query string, limit, offset int) ([]User, error)
	Count(ctx context.Context, query string) (int, error)
}

// PasswordHasher hashes and compares passwords. bcrypt is the default.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// RefreshTokenStore persists refresh tokens. The KV store satisfies it.
type RefreshTokenStore interface {
	Issue(ctx context.Context, userID int64, token string, ttl time.Duration) error
	Revoke(ctx context.Context, token string) error
	FindUserID(ctx context.Context, token string) (int64, bool, error)
}

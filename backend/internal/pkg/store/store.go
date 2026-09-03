package store

import (
	"context"
	"time"
)

// KV is a minimal string-based key/value contract used for transient
// state (quotes, holds, refresh tokens). Concrete implementations back
// this with Redis or an in-process map, which keeps domain services
// deciderable and testable without an external broker.
type KV interface {
	GetString(ctx context.Context, key string) (string, error)
	GetInt64(ctx context.Context, key string) (int64, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Del(ctx context.Context, keys ...string) error
	Keys(ctx context.Context, pattern string) ([]string, error)
	Close() error
}

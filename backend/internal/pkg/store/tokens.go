package store

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/you/airline-ticketing/internal/pkg/crypto"
)

// KVTokenStore adapts a KV implementation to serve as an opaque,
// hashed refresh-token store. The raw token is never persisted: only its
// SHA-256 digest is stored, keyed by prefix and digest.
type KVTokenStore struct {
	kv KV
}

func NewKVTokenStore(kv KV) *KVTokenStore { return &KVTokenStore{kv: kv} }

const refreshTokenPrefix = "refresh:"

// Issue stores a refresh token for the given user with the provided TTL.
func (s *KVTokenStore) Issue(ctx context.Context, userID int64, token string, ttl time.Duration) error {
	key := refreshTokenPrefix + crypto.Sha256Hex(token)
	return s.kv.Set(ctx, key, itoa(userID), ttl)
}

// Revoke deletes a refresh token.
func (s *KVTokenStore) Revoke(ctx context.Context, token string) error {
	key := refreshTokenPrefix + crypto.Sha256Hex(token)
	return s.kv.Del(ctx, key)
}

// FindUserID returns the user id bound to a refresh token. A missing or
// expired token yields ok=false with a nil error so callers can
// distinguish "not found" from a genuine storage failure.
func (s *KVTokenStore) FindUserID(ctx context.Context, token string) (int64, bool, error) {
	val, err := s.kv.GetString(ctx, refreshTokenPrefix+crypto.Sha256Hex(token))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	id, perr := strconv.ParseInt(val, 10, 64)
	return id, perr == nil, perr
}

func itoa(n int64) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

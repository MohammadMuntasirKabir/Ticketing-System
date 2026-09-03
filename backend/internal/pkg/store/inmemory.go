package store

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrNotFound = errors.New("store: key not found")

type memEntry struct {
	value   string
	expires int64 // unix seconds; 0 means no expiry
}

type InMemory struct {
	mu   sync.RWMutex
	data map[string]memEntry
}

func NewInMemory() *InMemory {
	return &InMemory{data: make(map[string]memEntry)}
}

func (m *InMemory) GetString(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.data[key]
	if !ok {
		return "", ErrNotFound
	}
	if e.expires != 0 && time.Now().Unix() >= e.expires {
		return "", ErrNotFound
	}
	return e.value, nil
}

func (m *InMemory) GetInt64(ctx context.Context, key string) (int64, bool, error) {
	v, err := m.GetString(ctx, key)
	if err != nil {
		return 0, false, err
	}
	n, parseErr := strconv.ParseInt(v, 10, 64)
	return n, parseErr == nil, parseErr
}

func (m *InMemory) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).Unix()
	}
	m.data[key] = memEntry{value: value, expires: exp}
	return nil
}

func (m *InMemory) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.data[key]; ok {
		if e.expires == 0 || time.Now().Unix() < e.expires {
			return false, nil
		}
	}
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).Unix()
	}
	m.data[key] = memEntry{value: value, expires: exp}
	return true, nil
}

func (m *InMemory) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func (m *InMemory) Keys(ctx context.Context, pattern string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var keys []string
	for k := range m.data {
		if matchPattern(pattern, k) {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (m *InMemory) Close() error { return nil }

// matchPattern implements glob matching with an arbitrary number of '*'
// wildcards, sufficient for the hold:prefix patterns used here.
func matchPattern(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	rest := s[len(parts[0]):]
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		idx := strings.Index(rest, part)
		if idx < 0 {
			return false
		}
		rest = rest[idx+len(part):]
	}
	return true
}

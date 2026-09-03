package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInMemorySetGetDel(t *testing.T) {
	ctx := context.Background()
	m := NewInMemory()
	if err := m.Set(ctx, "a", "1", 0); err != nil {
		t.Fatal(err)
	}
	v, err := m.GetString(ctx, "a")
	if err != nil || v != "1" {
		t.Fatalf("GetString = %q, %v", v, err)
	}
	if err := m.Del(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetString(ctx, "a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestInMemoryTTL(t *testing.T) {
	ctx := context.Background()
	m := NewInMemory()
	if err := m.Set(ctx, "k", "v", 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	if _, err := m.GetString(ctx, "k"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected expiry, got %v", err)
	}
}

func TestInMemorySetNX(t *testing.T) {
	ctx := context.Background()
	m := NewInMemory()
	ok, err := m.SetNX(ctx, "lock", "1", time.Minute)
	if err != nil || !ok {
		t.Fatalf("first SetNX = %v, %v", ok, err)
	}
	ok, err = m.SetNX(ctx, "lock", "1", time.Minute)
	if err != nil || ok {
		t.Fatalf("second SetNX = %v, %v, want false", ok, err)
	}
	// Expired keys can be re-taken.
	if err := m.Set(ctx, "exp", "x", 20*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	ok, err = m.SetNX(ctx, "exp", "y", time.Minute)
	if err != nil || !ok {
		t.Fatalf("SetNX after expiry = %v, %v", ok, err)
	}
}

func TestInMemoryGetInt64(t *testing.T) {
	ctx := context.Background()
	m := NewInMemory()
	_ = m.Set(ctx, "n", "42", 0)
	n, ok, err := m.GetInt64(ctx, "n")
	if err != nil || !ok || n != 42 {
		t.Fatalf("GetInt64 = %d, %v, %v", n, ok, err)
	}
	_ = m.Set(ctx, "bad", "abc", 0)
	if _, _, err := m.GetInt64(ctx, "bad"); err == nil {
		t.Fatal("expected parse error for non-numeric value")
	}
}

func TestInMemoryKeysPattern(t *testing.T) {
	ctx := context.Background()
	m := NewInMemory()
	_ = m.Set(ctx, "inv:hold:1:2:ABC", "2", 0)
	_ = m.Set(ctx, "inv:hold:1:3:ABC", "1", 0)
	_ = m.Set(ctx, "inv:hold:9:9:ZZZ", "5", 0)
	_ = m.Set(ctx, "other", "x", 0)

	keys, err := m.Keys(ctx, "inv:hold:1:*:ABC")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("Keys(inv:hold:1:*:ABC) = %v, want 2", keys)
	}

	all, err := m.Keys(ctx, "*")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatalf("Keys(*) = %v, want 4", all)
	}
}

func TestKVTokenStore(t *testing.T) {
	ctx := context.Background()
	m := NewInMemory()
	s := NewKVTokenStore(m)

	if err := s.Issue(ctx, 7, "rt_token", time.Minute); err != nil {
		t.Fatal(err)
	}
	id, ok, err := s.FindUserID(ctx, "rt_token")
	if err != nil || !ok || id != 7 {
		t.Fatalf("FindUserID = %d, %v, %v", id, ok, err)
	}
	// The raw token is never stored.
	if _, err := m.GetString(ctx, "refresh:rt_token"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("raw token should not be stored, got %v", err)
	}
	if err := s.Revoke(ctx, "rt_token"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.FindUserID(ctx, "rt_token"); ok {
		t.Fatal("token should be revoked")
	}
}

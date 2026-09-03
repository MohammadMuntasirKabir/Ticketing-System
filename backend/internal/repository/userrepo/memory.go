package userrepo

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/you/airline-ticketing/internal/domain/user"
)

var errNotFound = errors.New("userrepo: not found")

// Memory is an in-memory user.Repository used for local development and
// tests. Data does not survive a restart.
type Memory struct {
	mu      sync.RWMutex
	users   map[int64]user.User
	byEmail map[string]int64
	nextID  int64
}

func New() *Memory {
	return &Memory{
		users:   map[int64]user.User{},
		byEmail: map[string]int64{},
		nextID:  1,
	}
}

func (m *Memory) GetByID(ctx context.Context, id int64) (*user.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	cp := u
	return &cp, nil
}

func (m *Memory) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.byEmail[strings.ToLower(email)]
	if !ok {
		return nil, nil
	}
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	cp := u
	return &cp, nil
}

func (m *Memory) Create(ctx context.Context, u *user.User) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.nextID
	m.nextID++
	u.ID = id
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = time.Now().UTC()
	m.users[id] = *u
	m.byEmail[strings.ToLower(u.Email)] = id
	return id, nil
}

func (m *Memory) UpdateProfile(ctx context.Context, id int64, upd user.ProfileUpdate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return errNotFound
	}
	if upd.FirstName != nil {
		u.FirstName = *upd.FirstName
	}
	if upd.LastName != nil {
		u.LastName = *upd.LastName
	}
	if upd.Phone != nil {
		u.Phone = upd.Phone
	}
	u.UpdatedAt = time.Now().UTC()
	m.users[id] = u
	return nil
}

func (m *Memory) UpdateRole(ctx context.Context, id int64, role user.Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return errNotFound
	}
	u.Role = role
	u.UpdatedAt = time.Now().UTC()
	m.users[id] = u
	return nil
}

func (m *Memory) List(ctx context.Context, query string, limit, offset int) ([]user.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	q := strings.ToLower(strings.TrimSpace(query))
	var out []user.User
	for _, u := range m.users {
		if q == "" || strings.Contains(strings.ToLower(u.Email+" "+u.FirstName+" "+u.LastName), q) {
			out = append(out, u)
		}
	}
	if offset >= len(out) {
		return nil, nil
	}
	out = out[offset:]
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) Count(ctx context.Context, query string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	q := strings.ToLower(strings.TrimSpace(query))
	n := 0
	for _, u := range m.users {
		if q == "" || strings.Contains(strings.ToLower(u.Email+" "+u.FirstName+" "+u.LastName), q) {
			n++
		}
	}
	return n, nil
}

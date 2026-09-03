package userrepo

import (
	"context"
	"testing"

	"github.com/you/airline-ticketing/internal/domain/user"
)

func newUser(email string) *user.User {
	return &user.User{
		Email:        email,
		PasswordHash: "hash",
		FirstName:    "F",
		LastName:     "L",
		Role:         user.RoleCustomer,
		Status:       "ACTIVE",
	}
}

func TestCRUD(t *testing.T) {
	ctx := context.Background()
	m := New()

	id, err := m.Create(ctx, newUser("a@test.com"))
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("id should be non-zero")
	}

	u, _ := m.GetByID(ctx, id)
	if u == nil || u.Email != "a@test.com" || u.Role != user.RoleCustomer {
		t.Fatalf("got: %+v", u)
	}

	// Email lookup is case-insensitive.
	byEmail, _ := m.GetByEmail(ctx, "A@TEST.COM")
	if byEmail == nil || byEmail.ID != id {
		t.Fatalf("email lookup: %+v", byEmail)
	}
}

func TestUpdateProfileAndRole(t *testing.T) {
	ctx := context.Background()
	m := New()
	id, _ := m.Create(ctx, newUser("b@test.com"))

	fn := "Updated"
	phone := "+1000"
	if err := m.UpdateProfile(ctx, id, user.ProfileUpdate{FirstName: &fn, Phone: &phone}); err != nil {
		t.Fatal(err)
	}
	u, _ := m.GetByID(ctx, id)
	if u.FirstName != "Updated" || u.Phone == nil || *u.Phone != "+1000" {
		t.Fatalf("updated: %+v", u)
	}

	if err := m.UpdateRole(ctx, id, user.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	u, _ = m.GetByID(ctx, id)
	if u.Role != user.RoleAdmin {
		t.Fatalf("role = %q", u.Role)
	}
}

func TestUpdateMissing(t *testing.T) {
	ctx := context.Background()
	m := New()
	if err := m.UpdateProfile(ctx, 999, user.ProfileUpdate{}); err == nil {
		t.Fatal("expected error for missing user")
	}
	if err := m.UpdateRole(ctx, 999, user.RoleAdmin); err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestListAndCount(t *testing.T) {
	ctx := context.Background()
	m := New()
	_, _ = m.Create(ctx, newUser("l1@test.com"))
	_, _ = m.Create(ctx, newUser("l2@test.com"))

	all, err := m.List(ctx, "", 10, 0)
	if err != nil || len(all) != 2 {
		t.Fatalf("list: %d, %v", len(all), err)
	}
	if n, _ := m.Count(ctx, ""); n != 2 {
		t.Fatalf("count = %d", n)
	}

	// Query filtering + pagination.
	one, _ := m.List(ctx, "l1", 10, 0)
	if len(one) != 1 {
		t.Fatalf("query results = %d", len(one))
	}
	paged, _ := m.List(ctx, "", 1, 1)
	if len(paged) != 1 {
		t.Fatalf("paged results = %d", len(paged))
	}
	// Offset past the end returns empty.
	empty, _ := m.List(ctx, "", 10, 50)
	if len(empty) != 0 {
		t.Fatalf("offset past end = %d", len(empty))
	}
}

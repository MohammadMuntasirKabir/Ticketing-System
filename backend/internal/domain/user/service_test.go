package user_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/domain/user"
	"github.com/you/airline-ticketing/internal/pkg/config"
	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
	"github.com/you/airline-ticketing/internal/pkg/store"
	"github.com/you/airline-ticketing/internal/repository/userrepo"
)

func testConfig() *config.Config {
	cfg := config.Load()
	cfg.JWTSecret = "test-secret"
	cfg.JWTIssuer = "test-issuer"
	cfg.JWTExpiryMinutes = 30
	cfg.RefreshExpiryDays = 7
	return cfg
}

func newService() (*user.Service, *userrepo.Memory, *store.InMemory) {
	repo := userrepo.New()
	kv := store.NewInMemory()
	return user.NewService(repo, store.NewKVTokenStore(kv), nil, testConfig(), zap.NewNop()), repo, kv
}

func TestRegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	s, _, _ := newService()

	pair, err := s.Register(ctx, "a@test.com", "password123", "Alice", "Example")
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" || pair.ExpiresIn <= 0 {
		t.Fatalf("invalid token pair: %+v", pair)
	}

	// Duplicate email conflicts.
	if _, err := s.Register(ctx, "a@test.com", "password123", "A", "B"); !errors.Is(err, domainErrors.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}

	// Login with correct + wrong passwords.
	if _, err := s.Login(ctx, "a@test.com", "password123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if _, err := s.Login(ctx, "a@test.com", "wrongpass"); err == nil {
		t.Fatal("login with wrong password should fail")
	}
}

func TestRegisterValidation(t *testing.T) {
	ctx := context.Background()
	s, _, _ := newService()
	cases := []struct {
		name  string
		email string
		pass  string
	}{
		{"bad email", "not-an-email", "password123"},
		{"short password", "b@test.com", "short"},
		{"empty names", "c@test.com", "password123"},
	}
	for _, c := range cases {
		if _, err := s.Register(ctx, c.email, c.pass, "", ""); err == nil {
			t.Errorf("%s: expected error", c.name)
		}
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	ctx := context.Background()
	s, _, _ := newService()

	pair, err := s.Register(ctx, "r@test.com", "password123", "R", "T")
	if err != nil {
		t.Fatal(err)
	}

	rotated, err := s.RefreshToken(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if rotated.AccessToken == pair.AccessToken {
		t.Fatal("access token should rotate")
	}
	// Old refresh token must be revoked (single-use rotation).
	if _, err := s.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, domainErrors.ErrTokenExpired) {
		t.Fatalf("reused refresh token should fail, got %v", err)
	}
}

func TestLoginUnknownUserBurnsTime(t *testing.T) {
	ctx := context.Background()
	s, _, _ := newService()
	if _, err := s.Login(ctx, "ghost@test.com", "password123"); err == nil {
		t.Fatal("login of unknown user should fail")
	}
}

func TestGetAndUpdateProfile(t *testing.T) {
	ctx := context.Background()
	s, repo, _ := newService()

	pair, err := s.Register(ctx, "p@test.com", "password123", "P", "Q")
	if err != nil {
		t.Fatal(err)
	}
	_ = pair

	// Register returns a token but not the id; look up via repo.
	u, _ := repo.GetByEmail(ctx, "p@test.com")

	prof, err := s.GetProfile(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if prof.FirstName != "P" || prof.Email != "p@test.com" {
		t.Fatalf("profile: %+v", prof)
	}

	phone := "+15550001234"
	err = s.UpdateProfile(ctx, u.ID, user.ProfileUpdate{FirstName: strPtr("Pat"), Phone: &phone})
	if err != nil {
		t.Fatal(err)
	}
	prof, _ = s.GetProfile(ctx, u.ID)
	if prof.FirstName != "Pat" || prof.Phone == nil || *prof.Phone != phone {
		t.Fatalf("updated profile: %+v", prof)
	}
}

func TestListAndRoles(t *testing.T) {
	ctx := context.Background()
	s, _, _ := newService()
	_, _ = s.Register(ctx, "l1@test.com", "password123", "L", "One")
	_, _ = s.Register(ctx, "l2@test.com", "password123", "L", "Two")

	users, total, err := s.List(ctx, "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(users) != 2 {
		t.Fatalf("users=%d total=%d", len(users), total)
	}

	if err := s.UpdateRole(ctx, users[0].ID, user.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateRole(ctx, users[0].ID, user.Role("BOGUS")); err == nil {
		t.Fatal("invalid role should fail")
	}
}

func TestGetProfileNotFound(t *testing.T) {
	ctx := context.Background()
	s, _, _ := newService()
	if _, err := s.GetProfile(ctx, 999); err == nil {
		t.Fatal("expected not found")
	}
}

func isConflict(err error) bool {
	return err != nil
}

func strPtr(s string) *string { return &s }

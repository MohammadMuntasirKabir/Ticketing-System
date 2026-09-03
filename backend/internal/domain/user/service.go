package user

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/you/airline-ticketing/internal/pkg/config"
	"github.com/you/airline-ticketing/internal/pkg/crypto"
	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
)

// Service implements authentication and user management use-cases.
type Service struct {
	repo   Repository
	store  RefreshTokenStore
	ph     PasswordHasher
	cfg    *config.Config
	logger *zap.Logger

	// dummyHash burns comparable CPU time for unknown users to limit
	// user-enumeration timing attacks.
	dummyHash string
}

func NewService(repo Repository, store RefreshTokenStore, hasher PasswordHasher, cfg *config.Config, logger *zap.Logger) *Service {
	if hasher == nil {
		hasher = &bcryptHasher{}
	}
	dummyHash, _ := hasher.Hash("dummy-password")
	return &Service{repo: repo, store: store, ph: hasher, cfg: cfg, logger: logger, dummyHash: dummyHash}
}

type bcryptHasher struct{}

func (bcryptHasher) Hash(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(h), err
}
func (bcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (s *Service) Register(ctx context.Context, email, password, firstName, lastName string) (*TokenPair, error) {
	if !validEmail(email) {
		return nil, domainErrors.Validation("invalid email")
	}
	if len(password) < 8 {
		return nil, domainErrors.Validation("password must be at least 8 characters")
	}
	if firstName == "" || lastName == "" {
		return nil, domainErrors.Validation("first and last name are required")
	}

	if existing, err := s.repo.GetByEmail(ctx, email); err != nil {
		return nil, domainErrors.InternalWrap("failed to check existing user", err)
	} else if existing != nil {
		return nil, domainErrors.Conflict("email already registered")
	}

	hash, err := s.ph.Hash(password)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to hash password", err)
	}

	now := time.Now().UTC()
	u := &User{
		Email:        email,
		PasswordHash: hash,
		FirstName:    firstName,
		LastName:     lastName,
		Role:         RoleCustomer,
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	id, err := s.repo.Create(ctx, u)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to create user", err)
	}
	u.ID = id

	return s.generateTokenPair(ctx, u.ID, u.Email, u.Role)
}

func (s *Service) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to look up user", err)
	}
	if u == nil {
		// Burn time comparable to a real password comparison.
		_ = bcrypt.CompareHashAndPassword([]byte(s.dummyHash), []byte(password))
		return nil, domainErrors.Unauthorized("invalid credentials")
	}

	if err := s.ph.Compare(u.PasswordHash, password); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, domainErrors.Unauthorized("invalid credentials")
		}
		return nil, domainErrors.InternalWrap("failed to verify password", err)
	}

	if u.Status != "ACTIVE" {
		return nil, domainErrors.Forbidden("account is not active")
	}

	s.logger.Info("user login", zap.Int64("user_id", u.ID))
	return s.generateTokenPair(ctx, u.ID, u.Email, u.Role)
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	if refreshToken == "" {
		return nil, domainErrors.Validation("refresh token is required")
	}
	userID, ok, err := s.store.FindUserID(ctx, refreshToken)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to look up refresh token", err)
	}
	if !ok {
		return nil, domainErrors.ErrTokenExpired
	}
	if err := s.store.Revoke(ctx, refreshToken); err != nil {
		return nil, domainErrors.InternalWrap("failed to revoke refresh token", err)
	}

	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to load user", err)
	}
	if u == nil {
		return nil, domainErrors.ErrInvalidToken
	}
	return s.generateTokenPair(ctx, u.ID, u.Email, u.Role)
}

func (s *Service) GetProfile(ctx context.Context, userID int64) (*User, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to load user", err)
	}
	if u == nil {
		return nil, domainErrors.NotFound("user not found")
	}
	return u, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID int64, upd ProfileUpdate) error {
	if err := s.repo.UpdateProfile(ctx, userID, upd); err != nil {
		return domainErrors.InternalWrap("failed to update profile", err)
	}
	return nil
}

func (s *Service) List(ctx context.Context, query string, limit, offset int) ([]User, int, error) {
	users, err := s.repo.List(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, domainErrors.InternalWrap("failed to list users", err)
	}
	total, err := s.repo.Count(ctx, query)
	if err != nil {
		return nil, 0, domainErrors.InternalWrap("failed to count users", err)
	}
	return users, total, nil
}

func (s *Service) UpdateRole(ctx context.Context, id int64, role Role) error {
	if !role.IsValid() {
		return domainErrors.Validation("invalid role")
	}
	if err := s.repo.UpdateRole(ctx, id, role); err != nil {
		return domainErrors.InternalWrap("failed to update role", err)
	}
	return nil
}

// generateTokenPair mints a short-lived access JWT and a random, opaque
// refresh token (stored hashed server-side). A jti makes access tokens
// attributable; the issuer is taken from config.
func (s *Service) generateTokenPair(ctx context.Context, userID int64, email string, role Role) (*TokenPair, error) {
	now := time.Now().UTC()
	accessExpiry := now.Add(time.Duration(s.cfg.JWTExpiryMinutes) * time.Minute)

	jti, err := crypto.RandomString(16)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to mint token id", err)
	}

	claims := map[string]any{
		"iss":   s.cfg.JWTIssuer,
		"sub":   strconv.FormatInt(userID, 10),
		"email": email,
		"role":  string(role),
		"jti":   jti,
		"iat":   now.Unix(),
		"exp":   accessExpiry.Unix(),
		"nbf":   now.Unix(),
	}

	tokenStr, err := jwtSign(claims, s.cfg.JWTSecret)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to sign access token", err)
	}

	refreshToken := "rt_" + crypto.MustRandomString(32)
	if err := s.store.Issue(ctx, userID, refreshToken, s.cfg.RefreshTTL()); err != nil {
		return nil, domainErrors.InternalWrap("failed to persist refresh token", err)
	}

	return &TokenPair{
		AccessToken:  tokenStr,
		RefreshToken: refreshToken,
		ExpiresIn:    s.cfg.JWTExpiryMinutes * 60,
	}, nil
}

// validEmail performs a basic structural check. The struct validator on
// the DTO performs the authoritative validation.
func validEmail(s string) bool {
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	sub := s[at:]
	dot := strings.IndexByte(sub, '.')
	return dot > 0 && dot < len(sub)-1
}

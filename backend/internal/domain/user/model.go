package user

import (
	"errors"
	"time"
)

// Role enumerates the principals that can authenticate against the API.
type Role string

const (
	RoleCustomer Role = "CUSTOMER"
	RoleAgent    Role = "AGENT"
	RoleAdmin    Role = "ADMIN"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleCustomer, RoleAgent, RoleAdmin:
		return true
	}
	return false
}

// ErrUnknownRole is returned when a role string is not recognized.
var ErrUnknownRole = errors.New("unknown role")

type User struct {
	ID              int64
	Email           string
	PasswordHash    string
	FirstName       string
	LastName        string
	Phone           *string
	Role            Role
	EmailVerifiedAt *time.Time
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TokenPair is the domain token result returned to callers of the auth
// service. Handlers map it to the transport DTO.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds until the access token expires
}

// ProfileUpdate carries the mutable profile fields.
type ProfileUpdate struct {
	FirstName *string
	LastName  *string
	Phone     *string
}

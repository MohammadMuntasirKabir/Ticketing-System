package errors

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrValidation              = errors.New("validation error")
	ErrNotFound                = errors.New("not found")
	ErrConflict                = errors.New("conflict")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrForbidden               = errors.New("forbidden")
	ErrRateLimited             = errors.New("rate limited")
	ErrInternal                = errors.New("internal server error")
	ErrQuoteExpired            = errors.New("price quote has expired")
	ErrInventoryConflict       = errors.New("inventory conflict: seats no longer available")
	ErrInventoryContention     = errors.New("inventory contention: please try again")
	ErrPaymentFailed           = errors.New("payment failed")
	ErrPaymentNotConfirmed     = errors.New("payment not confirmed")
	ErrInvalidToken            = errors.New("invalid token")
	ErrTokenExpired            = errors.New("token expired")
	ErrPNRNotFound             = errors.New("PNR not found")
	ErrBookingAlreadyConfirmed = errors.New("booking already confirmed")
	ErrSeatNotAvailable        = errors.New("seat not available")
	ErrInvalidCredentials      = errors.New("invalid credentials")
)

// DomainError carries a stable machine-readable code alongside a
// user-facing message and the underlying cause. It is intentionally an
// opaque error type so callers must use HTTPStatus to translate it.
type DomainError struct {
	Code    string
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error { return e.Err }

func New(code, message string, err error) *DomainError {
	if err == nil {
		err = ErrInternal
	}
	return &DomainError{Code: code, Message: message, Err: err}
}

func Validation(message string) *DomainError { return New("VALIDATION_ERROR", message, ErrValidation) }
func NotFound(message string) *DomainError   { return New("NOT_FOUND", message, ErrNotFound) }
func Conflict(message string) *DomainError   { return New("CONFLICT", message, ErrConflict) }
func Unauthorized(message string) *DomainError {
	return New("UNAUTHORIZED", message, ErrUnauthorized)
}
func Forbidden(message string) *DomainError   { return New("FORBIDDEN", message, ErrForbidden) }
func RateLimited(message string) *DomainError { return New("RATE_LIMITED", message, ErrRateLimited) }
func Internal(message string) *DomainError    { return New("INTERNAL_ERROR", message, ErrInternal) }
func InternalWrap(message string, err error) *DomainError {
	return New("INTERNAL_ERROR", message, err)
}

// HTTPStatus maps an error to the most appropriate HTTP status code.
// DomainError wrappers are translated via the sentinel they carry; bare
// sentinel errors (returned directly by a few domain services) are
// translated via errors.Is. Anything else defaults to 500 so internal
// details never leak through without explicit intent.
func HTTPStatus(err error) int {
	var de *DomainError
	if errors.As(err, &de) {
		if st := sentinelStatus(de.Err); st != http.StatusInternalServerError {
			return st
		}
		return http.StatusInternalServerError
	}
	return sentinelStatus(err)
}

func sentinelStatus(err error) int {
	switch {
	case errors.Is(err, ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrInvalidToken), errors.Is(err, ErrTokenExpired),
		errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrQuoteExpired), errors.Is(err, ErrInventoryConflict),
		errors.Is(err, ErrInventoryContention), errors.Is(err, ErrPaymentFailed),
		errors.Is(err, ErrPaymentNotConfirmed), errors.Is(err, ErrBookingAlreadyConfirmed),
		errors.Is(err, ErrSeatNotAvailable), errors.Is(err, ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

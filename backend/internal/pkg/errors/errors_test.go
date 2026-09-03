package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestHTTPStatusDomainErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"validation", Validation("bad"), http.StatusBadRequest},
		{"not found", NotFound("nope"), http.StatusNotFound},
		{"conflict", Conflict("dup"), http.StatusConflict},
		{"unauthorized", Unauthorized("no"), http.StatusUnauthorized},
		{"forbidden", Forbidden("no"), http.StatusForbidden},
		{"rate limited", RateLimited("slow"), http.StatusTooManyRequests},
		{"internal", Internal("boom"), http.StatusInternalServerError},
		{"internal wrap", InternalWrap("boom", fmt.Errorf("cause")), http.StatusInternalServerError},
	}
	for _, c := range cases {
		if got := HTTPStatus(c.err); got != c.want {
			t.Errorf("%s: HTTPStatus = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestHTTPStatusSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"quote expired", ErrQuoteExpired, http.StatusConflict},
		{"inventory conflict", ErrInventoryConflict, http.StatusConflict},
		{"inventory contention", ErrInventoryContention, http.StatusConflict},
		{"payment failed", ErrPaymentFailed, http.StatusConflict},
		{"payment not confirmed", ErrPaymentNotConfirmed, http.StatusConflict},
		{"booking already confirmed", ErrBookingAlreadyConfirmed, http.StatusConflict},
		{"invalid token", ErrInvalidToken, http.StatusUnauthorized},
		{"token expired", ErrTokenExpired, http.StatusUnauthorized},
		{"pnr not found", ErrPNRNotFound, http.StatusInternalServerError},
		{"unknown", errors.New("random"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		if got := HTTPStatus(c.err); got != c.want {
			t.Errorf("%s: HTTPStatus = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestHTTPStatusWrappedSentinel(t *testing.T) {
	// A DomainError wrapping a bare sentinel keeps the sentinel's status.
	err := InternalWrap("oops", ErrQuoteExpired)
	if got := HTTPStatus(err); got != http.StatusConflict {
		t.Fatalf("wrapped sentinel: got %d, want %d", got, http.StatusConflict)
	}
}

func TestDomainErrorUnwrap(t *testing.T) {
	cause := fmt.Errorf("root")
	de := InternalWrap("msg", cause)
	if !errors.Is(de, cause) {
		t.Fatal("errors.Is should find the wrapped cause")
	}
	if de.Code != "INTERNAL_ERROR" {
		t.Fatalf("code = %q", de.Code)
	}
	if de.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
}

func TestHTTPStatusNil(t *testing.T) {
	if got := HTTPStatus(nil); got != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus(nil) = %d, want 500", got)
	}
}

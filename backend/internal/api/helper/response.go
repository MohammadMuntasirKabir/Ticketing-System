package helper

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"

	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
)

// JSON renders a value as JSON with the given status, setting
// Content-Type and a conservative Content-Length.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// Error renders a domain error as a JSON envelope and the correct HTTP
// status using domainErrors.HTTPStatus.
func Error(w http.ResponseWriter, err error) {
	status := domainErrors.HTTPStatus(err)
	body := map[string]string{"error": userMessage(err)}
	if status >= 500 {
		body = map[string]string{"error": "internal server error"}
	}
	JSON(w, status, body)
}

// userMessage returns a safe, generic message for server errors and the
// domain message otherwise so internal details never leak.
func userMessage(err error) string {
	var de *domainErrors.DomainError
	if errors.As(err, &de) {
		return de.Message
	}
	return err.Error()
}

// Decode parses and validates a JSON request body. Empty bodies on
// non-GET methods are rejected so handlers never silently operate on
// zero-value requests.
func Decode[T any](r *http.Request, v *T) error {
	if r.Body == nil {
		return domainErrors.Validation("request body is required")
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return domainErrors.Validation("malformed JSON body")
		}
		return domainErrors.Validation("invalid JSON body")
	}
	if err := validator.New().Struct(v); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) && len(ve) > 0 {
			return domainErrors.Validation(fieldError(ve[0]))
		}
		return domainErrors.Validation("validation failed")
	}
	return nil
}

func fieldError(ve validator.FieldError) string {
	return "field '" + ve.Field() + "' " + ve.Tag()
}

func QueryInt(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

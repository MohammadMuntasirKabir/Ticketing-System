package handler

import (
	"net/http"
	"strings"
	"time"
)

// parseDateLoose parses a date string in RFC3339 or YYYY-MM-DD form.
// An empty string yields the zero time, which callers treat as "unspecified".
func parseDateLoose(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02 15:04", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// listEnvelope wraps a collection response with a total count.
type listEnvelope struct {
	Data interface{} `json:"data"`
	Meta listMeta    `json:"meta"`
}

type listMeta struct {
	Total int `json:"total"`
}

// listParams carries pagination inputs from query strings.
type listParams struct {
	Limit  int
	Offset int
}

func queryListParams(r *http.Request) listParams {
	return listParams{Limit: queryInt(r, "limit", 100), Offset: queryInt(r, "offset", 0)}
}

func queryInt(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	var n int
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	if n < 0 {
		return 0
	}
	return n
}

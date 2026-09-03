package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/you/airline-ticketing/internal/pkg/store"
)

// IdempotencyCheck deduplicates non-idempotent POSTs by caching the
// response keyed on the Idempotency-Key header. Only successful (non-5xx)
// responses are cached. Replays are flagged with X-Idempotency-Replay.
func IdempotencyCheck(kv store.KV) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}
			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}
			ctx := r.Context()
			if cached, err := kv.GetString(ctx, "idem:"+key); err == nil && cached != "" {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Idempotency-Replay", "true")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(cached))
				return
			}
			rec := &capture{ResponseWriter: w, buf: &bytesBuf{}}
			next.ServeHTTP(rec, r.WithContext(ctx))
			if rec.status < 500 && rec.buf.size() > 0 {
				_ = kv.Set(context.Background(), "idem:"+key, rec.buf.String(), 24*time.Hour)
			}
		})
	}
}

// capture buffers the response so it can be replayed.
type capture struct {
	http.ResponseWriter
	status int
	buf    *bytesBuf
}

func (c *capture) WriteHeader(code int) {
	c.status = code
	c.ResponseWriter.WriteHeader(code)
}

func (c *capture) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	c.buf.Write(b)
	return c.ResponseWriter.Write(b)
}

// bytesBuf is a tiny growable buffer to avoid importing bytes.
type bytesBuf struct {
	b []byte
	n int
}

func (b *bytesBuf) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	b.b = append(b.b, p...)
	b.n = len(b.b)
	return len(p), nil
}

func (b *bytesBuf) String() string { return string(b.b) }
func (b *bytesBuf) size() int      { return b.n }

package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// RandomString returns a URL-safe random string with n bytes of entropy.
// It uses crypto/rand, never time-based entropy.
func RandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// MustRandomString is the panic-on-failure variant for init-only use.
func MustRandomString(n int) string {
	s, err := RandomString(n)
	if err != nil {
		panic(err)
	}
	return s
}

// Sha256Hex returns a hex digest used to index opaque tokens (e.g.
// session/refresh identifiers) in the store so the raw value is never
// persisted server-side.
func Sha256Hex(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex(sum[:])
}

func hex(b []byte) string {
	const hexd = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = hexd[c>>4]
		out[i*2+1] = hexd[c&0x0f]
	}
	return string(out)
}

package main

import (
	"crypto/subtle"
	"net/http"
	"os"
)

// RequireAuth wraps next and requires a valid X-API-Key header on every
// request. The expected key is read once from the FLAG_API_KEY environment
// variable when the middleware is constructed. A missing/empty FLAG_API_KEY,
// a missing/empty X-API-Key header, or a header that does not match the
// expected value all answer 401 {"error":"unauthorized"}.
func RequireAuth(next http.Handler) http.Handler {
	apiKey := os.Getenv("FLAG_API_KEY")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := r.Header.Get("X-API-Key")
		if apiKey == "" || provided == "" ||
			subtle.ConstantTimeCompare([]byte(provided), []byte(apiKey)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

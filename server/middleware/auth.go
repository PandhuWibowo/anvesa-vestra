package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/PandhuWibowo/oss-portable/auth"
)

type ctxKey string

const ClaimsKey ctxKey = "claims"

// RequireAuth returns a middleware that validates a Bearer JWT from the
// Authorization header and injects the parsed claims into the request context.
func RequireAuth(jwtSecret string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing or invalid authorization header"})
				return
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.ValidateToken(tokenStr, jwtSecret)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired token"})
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next(w, r.WithContext(ctx))
		}
	}
}

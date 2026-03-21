package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/PandhuWibowo/oss-portable/auth"
)

// RequireRole returns middleware that ensures the request context contains
// claims with a role in the allowed roles list. Returns 403 if not authorized.
func RequireRole(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	allowed := make(map[string]bool)
	for _, r := range roles {
		allowed[strings.TrimSpace(r)] = true
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			val := r.Context().Value(ClaimsKey)
			claims, ok := val.(*auth.Claims)
			if !ok || claims == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "insufficient permissions"})
				return
			}

			if !allowed[claims.Role] {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "insufficient permissions"})
				return
			}

			next(w, r)
		}
	}
}

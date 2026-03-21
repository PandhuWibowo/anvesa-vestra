package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/PandhuWibowo/oss-portable/auth"
	"github.com/PandhuWibowo/oss-portable/crypto"
	appdb "github.com/PandhuWibowo/oss-portable/db"
	"github.com/PandhuWibowo/oss-portable/middleware"
)

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterHandler creates the first user (initial setup only).
func RegisterHandler(jwtSecret string, jwtExpiry time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		req.Username = strings.TrimSpace(req.Username)
		if len(req.Username) < 3 {
			jsonError(w, "username must be at least 3 characters", http.StatusBadRequest)
			return
		}
		if len(req.Password) < 8 {
			jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
			return
		}

		var count int
		if err := appdb.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
			jsonError(w, "database error", http.StatusInternalServerError)
			return
		}
		if count > 0 {
			jsonError(w, "registration is closed, a user already exists", http.StatusForbidden)
			return
		}

		hash, err := crypto.HashPassword(req.Password)
		if err != nil {
			jsonError(w, "failed to hash password", http.StatusInternalServerError)
			return
		}

		result, err := appdb.DB.Exec(
			"INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, 'admin', ?)",
			req.Username, hash, time.Now().UTC(),
		)
		if err != nil {
			jsonError(w, "failed to create user", http.StatusInternalServerError)
			return
		}

		userID, _ := result.LastInsertId()

		token, err := auth.GenerateToken(userID, req.Username, "admin", jwtSecret, jwtExpiry)
		if err != nil {
			jsonError(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		jsonOK(w, map[string]any{
			"token":    token,
			"user_id":  userID,
			"username": req.Username,
			"role":     "admin",
		})
	}
}

// LoginHandler authenticates a user and returns a JWT.
func LoginHandler(jwtSecret string, jwtExpiry time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		req.Username = strings.TrimSpace(req.Username)
		if req.Username == "" || req.Password == "" {
			jsonError(w, "username and password are required", http.StatusBadRequest)
			return
		}

		var userID int64
		var passwordHash, role string
		err := appdb.DB.QueryRow(
			"SELECT id, password_hash, role FROM users WHERE username = ?", req.Username,
		).Scan(&userID, &passwordHash, &role)
		if err != nil {
			jsonError(w, "invalid username or password", http.StatusUnauthorized)
			return
		}

		if !crypto.CheckPassword(req.Password, passwordHash) {
			jsonError(w, "invalid username or password", http.StatusUnauthorized)
			return
		}

		token, err := auth.GenerateToken(userID, req.Username, role, jwtSecret, jwtExpiry)
		if err != nil {
			jsonError(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		jsonOK(w, map[string]any{
			"token":    token,
			"user_id":  userID,
			"username": req.Username,
			"role":     role,
		})
	}
}

// SetupStatusHandler returns whether initial setup (user registration) is needed.
func SetupStatusHandler(authEnabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var count int
		if err := appdb.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
			jsonError(w, "database error", http.StatusInternalServerError)
			return
		}
		jsonOK(w, map[string]any{
			"setup_required": authEnabled && count == 0,
			"auth_enabled":   authEnabled,
		})
	}
}

// MeHandler returns the current authenticated user's info from the JWT claims.
func MeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(middleware.ClaimsKey).(*auth.Claims)
	if !ok || claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	jsonOK(w, map[string]any{
		"user_id":  claims.UserID,
		"username": claims.Username,
		"role":     claims.Role,
	})
}

package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/PandhuWibowo/oss-portable/auth"
	appdb "github.com/PandhuWibowo/oss-portable/db"
	"github.com/PandhuWibowo/oss-portable/middleware"
)

// TOTPSetupHandler generates a new TOTP secret for the current user.
// POST /api/auth/2fa/setup
func TOTPSetupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(middleware.ClaimsKey).(*auth.Claims)
	if !ok || claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Anveesa Vestra",
		AccountName: claims.Username,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		jsonError(w, "failed to generate TOTP key", http.StatusInternalServerError)
		return
	}

	// Store the pending secret (not yet confirmed)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := appdb.DB.Exec(
		"UPDATE users SET totp_secret = ?, totp_enabled = 0, updated_at = ? WHERE id = ?",
		key.Secret(), now, claims.UserID,
	); err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{
		"secret":   key.Secret(),
		"otpauth":  key.URL(),
		"issuer":   "Anveesa Vestra",
		"account":  claims.Username,
	})
}

// TOTPVerifyHandler confirms the TOTP code and enables 2FA.
// POST /api/auth/2fa/verify
func TOTPVerifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(middleware.ClaimsKey).(*auth.Claims)
	if !ok || claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var secret string
	if err := appdb.DB.QueryRow(
		"SELECT COALESCE(totp_secret, '') FROM users WHERE id = ?", claims.UserID,
	).Scan(&secret); err != nil || secret == "" {
		jsonError(w, "no TOTP setup pending", http.StatusBadRequest)
		return
	}

	valid, err := totp.ValidateCustom(req.Code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil || !valid {
		jsonError(w, "invalid TOTP code", http.StatusUnauthorized)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := appdb.DB.Exec(
		"UPDATE users SET totp_enabled = 1, updated_at = ? WHERE id = ?",
		now, claims.UserID,
	); err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]bool{"enabled": true})
}

// TOTPDisableHandler disables 2FA for the current user after verifying their password.
// POST /api/auth/2fa/disable
func TOTPDisableHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(middleware.ClaimsKey).(*auth.Claims)
	if !ok || claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var secret string
	var enabled int
	if err := appdb.DB.QueryRow(
		"SELECT COALESCE(totp_secret, ''), COALESCE(totp_enabled, 0) FROM users WHERE id = ?", claims.UserID,
	).Scan(&secret, &enabled); err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	if enabled == 0 || secret == "" {
		jsonError(w, "2FA is not enabled", http.StatusBadRequest)
		return
	}

	valid, err := totp.ValidateCustom(req.Code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil || !valid {
		jsonError(w, "invalid TOTP code", http.StatusUnauthorized)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := appdb.DB.Exec(
		"UPDATE users SET totp_secret = NULL, totp_enabled = 0, updated_at = ? WHERE id = ?",
		now, claims.UserID,
	); err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]bool{"disabled": true})
}

// TOTPStatusHandler returns the current 2FA status for the authenticated user.
// GET /api/auth/2fa/status
func TOTPStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(middleware.ClaimsKey).(*auth.Claims)
	if !ok || claims == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var enabled int
	if err := appdb.DB.QueryRow(
		"SELECT COALESCE(totp_enabled, 0) FROM users WHERE id = ?", claims.UserID,
	).Scan(&enabled); err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]bool{"enabled": enabled == 1})
}

// TOTPChallengeHandler validates a TOTP code during login (before issuing JWT).
// POST /api/auth/2fa/challenge
func TOTPChallengeHandler(jwtSecret string, jwtExpiry time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			UserID int64  `json:"user_id"`
			Code   string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.UserID == 0 {
			jsonError(w, "user_id is required", http.StatusBadRequest)
			return
		}

		var username, role, secret string
		var enabled int
		if err := appdb.DB.QueryRow(
			"SELECT username, role, COALESCE(totp_secret, ''), COALESCE(totp_enabled, 0) FROM users WHERE id = ?",
			req.UserID,
		).Scan(&username, &role, &secret, &enabled); err != nil {
			jsonError(w, "user not found", http.StatusUnauthorized)
			return
		}

		if enabled == 0 || secret == "" {
			jsonError(w, "2FA is not enabled for this user", http.StatusBadRequest)
			return
		}

		valid, err := totp.ValidateCustom(req.Code, secret, time.Now().UTC(), totp.ValidateOpts{
			Period:    30,
			Skew:      1,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})
		if err != nil || !valid {
			jsonError(w, "invalid TOTP code", http.StatusUnauthorized)
			return
		}

		token, err := auth.GenerateToken(req.UserID, username, role, jwtSecret, jwtExpiry)
		if err != nil {
			jsonError(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		jsonOK(w, map[string]any{
			"token":    token,
			"user_id":  req.UserID,
			"username": username,
			"role":     role,
		})
	}
}

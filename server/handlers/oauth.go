package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PandhuWibowo/oss-portable/auth"
	appdb "github.com/PandhuWibowo/oss-portable/db"
)

func oauthRedirectBase() string {
	if v := os.Getenv("OAUTH_REDIRECT_BASE"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:5173"
}

// OAuthConfigHandler returns which OAuth providers are enabled and their client IDs.
func OAuthConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	googleID := os.Getenv("GOOGLE_CLIENT_ID")
	githubID := os.Getenv("GITHUB_CLIENT_ID")

	jsonOK(w, map[string]any{
		"google": map[string]any{
			"enabled":   googleID != "",
			"client_id": googleID,
		},
		"github": map[string]any{
			"enabled":   githubID != "",
			"client_id": githubID,
		},
	})
}

// OAuthGoogleCallback handles the Google OAuth2 callback and redirects with a JWT.
func OAuthGoogleCallback(jwtSecret string, jwtExpiry time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			jsonError(w, "missing code parameter", http.StatusBadRequest)
			return
		}

		clientID := os.Getenv("GOOGLE_CLIENT_ID")
		clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			jsonError(w, "Google OAuth not configured", http.StatusInternalServerError)
			return
		}

		redirectURI := oauthRedirectBase() + "/api/auth/oauth/google/callback"

		// Exchange authorization code for access token
		tokenResp, err := http.PostForm("https://oauth2.googleapis.com/token", url.Values{
			"code":          {code},
			"client_id":     {clientID},
			"client_secret": {clientSecret},
			"redirect_uri":  {redirectURI},
			"grant_type":    {"authorization_code"},
		})
		if err != nil {
			jsonError(w, "failed to exchange code", http.StatusBadGateway)
			return
		}
		defer tokenResp.Body.Close()

		body, _ := io.ReadAll(tokenResp.Body)
		var tokenData struct {
			AccessToken string `json:"access_token"`
			Error       string `json:"error"`
		}
		if err := json.Unmarshal(body, &tokenData); err != nil || tokenData.AccessToken == "" {
			msg := "failed to obtain access token"
			if tokenData.Error != "" {
				msg = tokenData.Error
			}
			jsonError(w, msg, http.StatusBadGateway)
			return
		}

		// Fetch user info
		req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
		userResp, err := http.DefaultClient.Do(req)
		if err != nil {
			jsonError(w, "failed to fetch user info", http.StatusBadGateway)
			return
		}
		defer userResp.Body.Close()

		var userInfo struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil || userInfo.Email == "" {
			jsonError(w, "failed to parse user info", http.StatusBadGateway)
			return
		}

		userID, username, role, err := findOrCreateOAuthUser(userInfo.Email, "google")
		if err != nil {
			jsonError(w, "failed to find or create user", http.StatusInternalServerError)
			return
		}

		token, err := auth.GenerateToken(userID, username, role, jwtSecret, jwtExpiry)
		if err != nil {
			jsonError(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, oauthRedirectBase()+"/?oauth_token="+url.QueryEscape(token), http.StatusTemporaryRedirect)
	}
}

// OAuthGitHubCallback handles the GitHub OAuth2 callback and redirects with a JWT.
func OAuthGitHubCallback(jwtSecret string, jwtExpiry time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			jsonError(w, "missing code parameter", http.StatusBadRequest)
			return
		}

		clientID := os.Getenv("GITHUB_CLIENT_ID")
		clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			jsonError(w, "GitHub OAuth not configured", http.StatusInternalServerError)
			return
		}

		// Exchange authorization code for access token
		form := url.Values{
			"code":          {code},
			"client_id":     {clientID},
			"client_secret": {clientSecret},
		}
		tokenReq, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
		tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		tokenReq.Header.Set("Accept", "application/json")

		tokenResp, err := http.DefaultClient.Do(tokenReq)
		if err != nil {
			jsonError(w, "failed to exchange code", http.StatusBadGateway)
			return
		}
		defer tokenResp.Body.Close()

		var tokenData struct {
			AccessToken string `json:"access_token"`
			Error       string `json:"error"`
		}
		if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil || tokenData.AccessToken == "" {
			msg := "failed to obtain access token"
			if tokenData.Error != "" {
				msg = tokenData.Error
			}
			jsonError(w, msg, http.StatusBadGateway)
			return
		}

		// Fetch user info
		userReq, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
		userReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
		userReq.Header.Set("User-Agent", "AnveesaVestra")
		userResp, err := http.DefaultClient.Do(userReq)
		if err != nil {
			jsonError(w, "failed to fetch user info", http.StatusBadGateway)
			return
		}
		defer userResp.Body.Close()

		var userInfo struct {
			Login string `json:"login"`
		}
		if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil || userInfo.Login == "" {
			jsonError(w, "failed to parse user info", http.StatusBadGateway)
			return
		}

		userID, username, role, err := findOrCreateOAuthUser(userInfo.Login, "github")
		if err != nil {
			jsonError(w, "failed to find or create user", http.StatusInternalServerError)
			return
		}

		token, err := auth.GenerateToken(userID, username, role, jwtSecret, jwtExpiry)
		if err != nil {
			jsonError(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, oauthRedirectBase()+"/?oauth_token="+url.QueryEscape(token), http.StatusTemporaryRedirect)
	}
}

// findOrCreateOAuthUser looks up a user by username; if not found, inserts a new
// OAuth user. The first user ever created gets the 'admin' role.
func findOrCreateOAuthUser(username, provider string) (int64, string, string, error) {
	var userID int64
	var role string
	err := appdb.DB.QueryRow(
		"SELECT id, role FROM users WHERE username = ?", username,
	).Scan(&userID, &role)
	if err == nil {
		return userID, username, role, nil
	}

	// Determine role: admin if this is the very first user.
	var count int
	if err := appdb.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return 0, "", "", fmt.Errorf("db count: %w", err)
	}
	role = "user"
	if count == 0 {
		role = "admin"
	}

	result, err := appdb.DB.Exec(
		"INSERT INTO users (username, password_hash, role, created_at) VALUES (?, '', ?, ?)",
		username, role, time.Now().UTC(),
	)
	if err != nil {
		return 0, "", "", fmt.Errorf("insert user: %w", err)
	}
	userID, _ = result.LastInsertId()
	return userID, username, role, nil
}

package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/PandhuWibowo/oss-portable/crypto"
	appdb "github.com/PandhuWibowo/oss-portable/db"
	"github.com/PandhuWibowo/oss-portable/middleware"
)

// providerTable maps a provider name to its DB table.
var providerTable = map[string]string{
	"aws":     "aws_connections",
	"alibaba": "alibaba_connections",
	"huawei":  "huawei_connections",
	"gcp":     "gcp_connections",
	"azure":   "azure_connections",
	"gdrive":  "gdrive_connections",
	"b2":      "b2_connections",
	"do":      "do_connections",
}

// s3Providers maps provider names to their S3Provider instances.
var s3Providers = map[string]*S3Provider{
	"aws":     AWS,
	"alibaba": Alibaba,
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateSharedLink creates a new shared download link.
func CreateSharedLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ConnectionID int64  `json:"connection_id"`
		Provider     string `json:"provider"`
		Object       string `json:"object"`
		Password     string `json:"password"`
		ExpiresHours int    `json:"expires_hours"`
		MaxDownloads int    `json:"max_downloads"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := requireFields(map[string]string{"provider": req.Provider, "object": req.Object}); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.ConnectionID == 0 {
		jsonError(w, "connection_id is required", http.StatusBadRequest)
		return
	}

	token, err := generateToken()
	if err != nil {
		jsonError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	var passwordHash sql.NullString
	if req.Password != "" {
		hash, err := crypto.HashPassword(req.Password)
		if err != nil {
			jsonError(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		passwordHash = sql.NullString{String: hash, Valid: true}
	}

	var expiresAt sql.NullString
	if req.ExpiresHours > 0 {
		exp := time.Now().UTC().Add(time.Duration(req.ExpiresHours) * time.Hour)
		expiresAt = sql.NullString{String: exp.Format(time.RFC3339), Valid: true}
	}

	var createdBy sql.NullInt64
	if claims, ok := r.Context().Value(middleware.ClaimsKey).(interface{ GetUserID() int64 }); ok {
		createdBy = sql.NullInt64{Int64: claims.GetUserID(), Valid: true}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = appdb.DB.Exec(
		`INSERT INTO shared_links
			(token, connection_id, provider, object, password_hash, expires_at, max_downloads, download_count, created_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		token, req.ConnectionID, req.Provider, req.Object,
		passwordHash, expiresAt, req.MaxDownloads, createdBy, now,
	)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{
		"token": token,
		"url":   "/api/share/" + token,
	})
}

// AccessSharedLink validates and serves a shared link download.
func AccessSharedLink(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		jsonError(w, "invalid share URL", http.StatusBadRequest)
		return
	}
	token := parts[3]

	var (
		id            int64
		connectionID  int64
		provider      string
		object        string
		passwordHash  sql.NullString
		expiresAt     sql.NullString
		maxDownloads  int
		downloadCount int
	)
	err := appdb.DB.QueryRow(
		`SELECT id, connection_id, provider, object, password_hash, expires_at, max_downloads, download_count
		 FROM shared_links WHERE token = ?`, token,
	).Scan(&id, &connectionID, &provider, &object, &passwordHash, &expiresAt, &maxDownloads, &downloadCount)
	if err != nil {
		jsonError(w, "link not found", http.StatusNotFound)
		return
	}

	if expiresAt.Valid {
		exp, err := time.Parse(time.RFC3339, expiresAt.String)
		if err == nil && time.Now().UTC().After(exp) {
			jsonError(w, "link has expired", http.StatusGone)
			return
		}
	}

	if maxDownloads > 0 && downloadCount >= maxDownloads {
		jsonError(w, "download limit reached", http.StatusGone)
		return
	}

	if passwordHash.Valid && passwordHash.String != "" {
		pw := r.URL.Query().Get("password")
		if pw == "" {
			jsonError(w, "password required", http.StatusUnauthorized)
			return
		}
		if !crypto.CheckPassword(pw, passwordHash.String) {
			jsonError(w, "incorrect password", http.StatusForbidden)
			return
		}
	}

	table, ok := providerTable[provider]
	if !ok {
		jsonError(w, "unsupported provider", http.StatusBadRequest)
		return
	}

	bucket, credJSON, err := lookupConnection(table, connectionID)
	if err != nil {
		jsonError(w, "connection not found", http.StatusNotFound)
		return
	}

	// For S3-compatible providers, generate a presigned URL and redirect.
	sp := s3Providers[provider]
	if sp != nil {
		creds, err := sp.CredsFunc(credJSON)
		if err != nil {
			jsonError(w, "invalid credentials", http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, err := sp.ClientFunc(ctx, creds)
		if err != nil {
			jsonError(w, "failed to create client", http.StatusInternalServerError)
			return
		}
		psClient := s3.NewPresignClient(client)
		presigned, err := psClient.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(object),
		}, func(o *s3.PresignOptions) { o.Expires = 15 * time.Minute })
		if err != nil {
			jsonError(w, "failed to generate download URL", http.StatusInternalServerError)
			return
		}

		appdb.DB.Exec("UPDATE shared_links SET download_count = download_count + 1 WHERE id = ?", id)
		http.Redirect(w, r, presigned.URL, http.StatusTemporaryRedirect)
		return
	}

	// For non-S3 providers, return a JSON response with the download info.
	appdb.DB.Exec("UPDATE shared_links SET download_count = download_count + 1 WHERE id = ?", id)
	jsonOK(w, map[string]any{
		"provider":      provider,
		"connection_id": connectionID,
		"object":        object,
		"bucket":        bucket,
	})
}

// ListSharedLinks returns all shared links for the current user.
func ListSharedLinks(w http.ResponseWriter, r *http.Request) {
	rows, err := appdb.DB.Query(
		`SELECT id, token, provider, object, expires_at, max_downloads, download_count, created_at
		 FROM shared_links ORDER BY created_at DESC`,
	)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type link struct {
		ID            int64   `json:"id"`
		Token         string  `json:"token"`
		Provider      string  `json:"provider"`
		Object        string  `json:"object"`
		ExpiresAt     *string `json:"expires_at"`
		MaxDownloads  int     `json:"max_downloads"`
		DownloadCount int     `json:"download_count"`
		CreatedAt     string  `json:"created_at"`
	}

	links := []link{}
	for rows.Next() {
		var l link
		var expiresAt sql.NullString
		if err := rows.Scan(&l.ID, &l.Token, &l.Provider, &l.Object, &expiresAt, &l.MaxDownloads, &l.DownloadCount, &l.CreatedAt); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if expiresAt.Valid {
			l.ExpiresAt = &expiresAt.String
		}
		links = append(links, l)
	}
	jsonOK(w, links)
}

// DeleteSharedLink removes a shared link by ID.
func DeleteSharedLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := parseID(r, 3) // /api/shares/{id}
	if err != nil {
		jsonError(w, "invalid share id", http.StatusBadRequest)
		return
	}

	res, err := appdb.DB.Exec("DELETE FROM shared_links WHERE id = ?", id)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		jsonError(w, "link not found", http.StatusNotFound)
		return
	}

	jsonOK(w, map[string]string{"status": "deleted"})
}

// providerTableName returns the DB table name for a given provider.
func providerTableName(provider string) (string, bool) {
	t, ok := providerTable[provider]
	return t, ok
}

// resolveS3Provider returns the S3Provider for a given provider name, or nil.
func resolveS3Provider(name string) *S3Provider {
	return s3Providers[name]
}

func init() {
	// Huawei uses its own client functions but is still S3-compatible.
	s3Providers["huawei"] = &S3Provider{
		Name:       "huawei",
		Table:      "huawei_connections",
		CredsFunc:  obsCredsFromJSON,
		ClientFunc: obsS3Client,
		TestFunc:   testOBS,
	}
}

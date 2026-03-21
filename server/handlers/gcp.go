package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

// ── helpers ──────────────────────────────────────────────────────

func gcpClient(ctx context.Context, credentials string) (*storage.Client, error) {
	if strings.TrimSpace(credentials) == "" {
		return storage.NewClient(ctx, option.WithoutAuthentication())
	}
	return storage.NewClient(ctx, option.WithCredentialsJSON([]byte(credentials)))
}

// testGCP verifies GCP bucket access.
func testGCP(bucket, credentialsJSON string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := gcpClient(ctx, credentialsJSON)
	if err != nil {
		return err
	}
	defer client.Close()

	if _, attrsErr := client.Bucket(bucket).Attrs(ctx); attrsErr == nil {
		return nil
	}

	it := client.Bucket(bucket).Objects(ctx, &storage.Query{})
	_, listErr := it.Next()
	if listErr == nil || listErr == iterator.Done {
		return nil
	}
	return fmt.Errorf("bucket not accessible")
}

// ── connection CRUD ───────────────────────────────────────────────

func ListGCP(w http.ResponseWriter, r *http.Request) {
	rows, err := appdb.DB.Query(
		"SELECT id, name, bucket, credentials, created_at FROM gcp_connections ORDER BY created_at DESC",
	)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type GCPConnection struct {
		ID          int64     `json:"id"`
		Name        string    `json:"name"`
		Bucket      string    `json:"bucket"`
		Credentials string    `json:"credentials"`
		CreatedAt   time.Time `json:"created_at"`
	}

	conns := []GCPConnection{}
	for rows.Next() {
		var c GCPConnection
		var created string
		if err := rows.Scan(&c.ID, &c.Name, &c.Bucket, &c.Credentials, &created); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		c.Credentials, _ = decryptCredentials(c.Credentials)
		c.CreatedAt, _ = time.Parse(time.RFC3339, created)
		conns = append(conns, c)
	}
	jsonOK(w, conns)
}

func CreateGCP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := testGCP(req.Bucket, req.Credentials); err != nil {
		http.Error(w, fmt.Sprintf("test failed: %v", err), http.StatusBadRequest)
		return
	}
	encrypted, err := encryptCredentials(req.Credentials)
	if err != nil {
		http.Error(w, "failed to encrypt credentials", http.StatusInternalServerError)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := appdb.DB.Exec(
		"INSERT INTO gcp_connections (name, bucket, credentials, created_at) VALUES (?, ?, ?, ?)",
		req.Name, req.Bucket, encrypted, now,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
}

// GCPConnByID handles both DELETE and PUT for /api/gcp/connection/{id}.
func GCPConnByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		DeleteGCPConn(w, r)
	case http.MethodPut:
		UpdateGCPConn(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func DeleteGCPConn(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if _, err = appdb.DB.Exec("DELETE FROM gcp_connections WHERE id = ?", id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func UpdateGCPConn(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := testGCP(req.Bucket, req.Credentials); err != nil {
		http.Error(w, fmt.Sprintf("test failed: %v", err), http.StatusBadRequest)
		return
	}
	encrypted, err := encryptCredentials(req.Credentials)
	if err != nil {
		http.Error(w, "failed to encrypt credentials", http.StatusInternalServerError)
		return
	}
	if _, err := appdb.DB.Exec(
		"UPDATE gcp_connections SET name=?, bucket=?, credentials=? WHERE id=?",
		req.Name, req.Bucket, encrypted, id,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func TestGCP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := testGCP(req.Bucket, req.Credentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ── bucket operations ─────────────────────────────────────────────

type gcpEntry struct {
	Type        string    `json:"type"` // "dir" | "file"
	Name        string    `json:"name"`
	Display     string    `json:"display"`
	Size        int64     `json:"size,omitempty"`
	Updated     time.Time `json:"updated,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
}

// BrowseGCPBucket lists entries (files + virtual folders) at a given prefix with pagination.
func BrowseGCPBucket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Prefix      string `json:"prefix"`
		PageToken   string `json:"page_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := gcpClient(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	it := client.Bucket(req.Bucket).Objects(ctx, &storage.Query{
		Prefix:    req.Prefix,
		Delimiter: "/",
	})
	if req.PageToken != "" {
		it.PageInfo().Token = req.PageToken
	}
	it.PageInfo().MaxSize = 200

	var entries []gcpEntry
	for i := 0; i < 200; i++ {
		attrs, iterErr := it.Next()
		if iterErr == iterator.Done {
			break
		}
		if iterErr != nil {
			break
		}
		if attrs.Prefix != "" {
			display := strings.TrimSuffix(strings.TrimPrefix(attrs.Prefix, req.Prefix), "/")
			entries = append(entries, gcpEntry{Type: "dir", Name: attrs.Prefix, Display: display})
		} else if attrs.Name != req.Prefix {
			display := strings.TrimPrefix(attrs.Name, req.Prefix)
			entries = append(entries, gcpEntry{
				Type:        "file",
				Name:        attrs.Name,
				Display:     display,
				Size:        attrs.Size,
				Updated:     attrs.Updated,
				ContentType: attrs.ContentType,
			})
		}
	}
	if entries == nil {
		entries = []gcpEntry{}
	}
	nextToken := it.PageInfo().Token
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"prefix":          req.Prefix,
		"entries":         entries,
		"next_page_token": nextToken,
	})
}

// ListGCPObjects is kept for backward compat (flat listing).
func ListGCPObjects(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := gcpClient(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	type gcpObject struct {
		Name        string    `json:"name"`
		Size        int64     `json:"size"`
		Updated     time.Time `json:"updated"`
		ContentType string    `json:"content_type"`
	}

	const maxResults = 1000
	it := client.Bucket(req.Bucket).Objects(ctx, nil)
	var objects []gcpObject
	for len(objects) < maxResults {
		attrs, iterErr := it.Next()
		if iterErr == iterator.Done || iterErr != nil {
			break
		}
		objects = append(objects, gcpObject{
			Name: attrs.Name, Size: attrs.Size,
			Updated: attrs.Updated, ContentType: attrs.ContentType,
		})
	}
	if objects == nil {
		objects = []gcpObject{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"objects":   objects,
		"truncated": len(objects) == maxResults,
	})
}

// GCPDownloadURL returns a public or signed download URL for an object.
// Optional expires_in (seconds) controls signed URL lifetime; defaults to 900s.
func GCPDownloadURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Object      string `json:"object"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Public bucket — direct CDN URL
	if strings.TrimSpace(req.Credentials) == "" {
		url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", req.Bucket, req.Object)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"url": url})
		return
	}

	expiry := time.Duration(req.ExpiresIn) * time.Second
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}

	// Authenticated — signed URL
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := gcpClient(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	signed, err := client.Bucket(req.Bucket).SignedURL(req.Object, &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expiry),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"url": signed})
}

// DeleteGCPObject deletes a single GCS object.
func DeleteGCPObject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Object      string `json:"object"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := gcpClient(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	if err := client.Bucket(req.Bucket).Object(req.Object).Delete(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CopyGCPObject copies (and optionally deletes) a GCS object — used for rename/move.
func CopyGCPObject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Source      string `json:"source"`
		Destination string `json:"destination"`
		Delete      bool   `json:"delete_source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := gcpClient(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	src := client.Bucket(req.Bucket).Object(req.Source)
	dst := client.Bucket(req.Bucket).Object(req.Destination)

	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if req.Delete {
		if err := src.Delete(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// UploadGCPObject uploads a file to GCS via multipart form.
func UploadGCPObject(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bucket := r.FormValue("bucket")
	creds := r.FormValue("credentials")
	prefix := r.FormValue("prefix")

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client, err := gcpClient(ctx, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	objectName := prefix + header.Filename
	wc := client.Bucket(bucket).Object(objectName).NewWriter(ctx)
	wc.ContentType = header.Header.Get("Content-Type")
	if wc.ContentType == "" {
		wc.ContentType = "application/octet-stream"
	}

	if _, err := io.Copy(wc, file); err != nil {
		_ = wc.Close()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := wc.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"name": objectName})
}

// GCPBucketStats returns sampled object count and total size.
func GCPBucketStats(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := gcpClient(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	const maxSample = 10000
	it := client.Bucket(req.Bucket).Objects(ctx, nil)
	var count int64
	var totalSize int64
	for count < maxSample {
		attrs, iterErr := it.Next()
		if iterErr == iterator.Done || iterErr != nil {
			break
		}
		count++
		totalSize += attrs.Size
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object_count": count,
		"total_size":   totalSize,
		"truncated":    count == maxSample,
	})
}

// GetGCPMetadata returns full metadata for a GCS object.
func GetGCPMetadata(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Object      string `json:"object"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := gcpClient(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	attrs, err := client.Bucket(req.Bucket).Object(req.Object).Attrs(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	md := attrs.Metadata
	if md == nil {
		md = map[string]string{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"content_type":  attrs.ContentType,
		"cache_control": attrs.CacheControl,
		"metadata":      md,
		"size":          attrs.Size,
		"updated":       attrs.Updated,
		"etag":          attrs.Etag,
		"md5":           fmt.Sprintf("%x", attrs.MD5),
	})
}

// DeletePrefixGCP deletes all objects under a given prefix (recursive folder delete).
func DeletePrefixGCP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Prefix      string `json:"prefix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client, err := gcpClient(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	it := client.Bucket(req.Bucket).Objects(ctx, &storage.Query{Prefix: req.Prefix})
	deleted := 0
	for {
		attrs, iterErr := it.Next()
		if iterErr == iterator.Done {
			break
		}
		if iterErr != nil {
			http.Error(w, iterErr.Error(), http.StatusInternalServerError)
			return
		}
		client.Bucket(req.Bucket).Object(attrs.Name).Delete(ctx)
		deleted++
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"deleted": deleted})
}

// UpdateGCPMetadata patches content-type, cache-control, and custom metadata on a GCS object.
func UpdateGCPMetadata(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket       string            `json:"bucket"`
		Credentials  string            `json:"credentials"`
		Object       string            `json:"object"`
		ContentType  string            `json:"content_type"`
		CacheControl string            `json:"cache_control"`
		Metadata     map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := gcpClient(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	uattrs := storage.ObjectAttrsToUpdate{
		ContentType:  req.ContentType,
		CacheControl: req.CacheControl,
		Metadata:     req.Metadata,
	}
	if _, err := client.Bucket(req.Bucket).Object(req.Object).Update(ctx, uattrs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

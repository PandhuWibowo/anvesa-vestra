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

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

// ── helpers ──────────────────────────────────────────────────────

// gdriveService builds an authenticated Drive v3 service from Service Account JSON.
func gdriveService(ctx context.Context, credentials string) (*drive.Service, error) {
	return drive.NewService(ctx, option.WithCredentialsJSON([]byte(credentials)))
}

// testGDrive verifies Google Drive folder access.
func testGDrive(folderID, credentialsJSON string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	srv, err := gdriveService(ctx, credentialsJSON)
	if err != nil {
		return err
	}

	// Try to get the folder metadata
	f, err := srv.Files.Get(folderID).Fields("id, name, mimeType").SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("folder not accessible: %v", err)
	}
	if f.MimeType != "application/vnd.google-apps.folder" {
		return fmt.Errorf("ID %q is not a folder (type: %s)", folderID, f.MimeType)
	}
	return nil
}

// ── connection CRUD ───────────────────────────────────────────────

func ListGDrive(w http.ResponseWriter, r *http.Request) {
	rows, err := appdb.DB.Query(
		"SELECT id, name, bucket, credentials, created_at FROM gdrive_connections ORDER BY created_at DESC",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type GDriveConnection struct {
		ID          int64     `json:"id"`
		Name        string    `json:"name"`
		Bucket      string    `json:"bucket"`
		Credentials string    `json:"credentials"`
		CreatedAt   time.Time `json:"created_at"`
	}

	conns := []GDriveConnection{}
	for rows.Next() {
		var c GDriveConnection
		var created string
		if err := rows.Scan(&c.ID, &c.Name, &c.Bucket, &c.Credentials, &created); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, created)
		conns = append(conns, c)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(conns)
}

func CreateGDrive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := testGDrive(req.Bucket, req.Credentials); err != nil {
		http.Error(w, fmt.Sprintf("test failed: %v", err), http.StatusBadRequest)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := appdb.DB.Exec(
		"INSERT INTO gdrive_connections (name, bucket, credentials, created_at) VALUES (?, ?, ?, ?)",
		req.Name, req.Bucket, req.Credentials, now,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
}

// GDriveConnByID handles both DELETE and PUT for /api/gdrive/connection/{id}.
func GDriveConnByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		DeleteGDriveConn(w, r)
	case http.MethodPut:
		UpdateGDriveConn(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func DeleteGDriveConn(w http.ResponseWriter, r *http.Request) {
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
	if _, err = appdb.DB.Exec("DELETE FROM gdrive_connections WHERE id = ?", id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func UpdateGDriveConn(w http.ResponseWriter, r *http.Request) {
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
	if err := testGDrive(req.Bucket, req.Credentials); err != nil {
		http.Error(w, fmt.Sprintf("test failed: %v", err), http.StatusBadRequest)
		return
	}
	if _, err := appdb.DB.Exec(
		"UPDATE gdrive_connections SET name=?, bucket=?, credentials=? WHERE id=?",
		req.Name, req.Bucket, req.Credentials, id,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func TestGDrive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := testGDrive(req.Bucket, req.Credentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ── file operations ───────────────────────────────────────────────

type gdriveEntry struct {
	Type    string    `json:"type"` // "dir" | "file"
	Name    string    `json:"name"`
	Display string    `json:"display"`
	Size    int64     `json:"size,omitempty"`
	Updated time.Time `json:"updated,omitempty"`
}

// resolveParentID resolves a "prefix" path to a Drive folder ID.
// The "prefix" uses forward-slash-separated folder names relative to the root folder.
// For example, prefix "photos/2024/" means: rootFolder → "photos" subfolder → "2024" subfolder.
// If prefix is empty, it returns the root folder ID.
func resolveParentID(ctx context.Context, srv *drive.Service, rootID, prefix string) (string, error) {
	if prefix == "" {
		return rootID, nil
	}
	prefix = strings.TrimSuffix(prefix, "/")
	parts := strings.Split(prefix, "/")

	currentID := rootID
	for _, part := range parts {
		if part == "" {
			continue
		}
		q := fmt.Sprintf("'%s' in parents and name = '%s' and mimeType = 'application/vnd.google-apps.folder' and trashed = false",
			currentID, strings.ReplaceAll(part, "'", "\\'"))
		result, err := srv.Files.List().Q(q).Fields("files(id)").PageSize(1).SupportsAllDrives(true).IncludeItemsFromAllDrives(true).Context(ctx).Do()
		if err != nil {
			return "", fmt.Errorf("resolving path %q: %v", part, err)
		}
		if len(result.Files) == 0 {
			return "", fmt.Errorf("folder %q not found in path", part)
		}
		currentID = result.Files[0].Id
	}
	return currentID, nil
}

// BrowseGDriveBucket lists entries (files + folders) in a given parent folder with pagination.
func BrowseGDriveBucket(w http.ResponseWriter, r *http.Request) {
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

	srv, err := gdriveService(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	parentID, err := resolveParentID(ctx, srv, req.Bucket, req.Prefix)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q := fmt.Sprintf("'%s' in parents and trashed = false", parentID)
	call := srv.Files.List().Q(q).
		Fields("nextPageToken, files(id, name, mimeType, size, modifiedTime)").
		PageSize(200).
		OrderBy("folder,name").
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true)
	if req.PageToken != "" {
		call = call.PageToken(req.PageToken)
	}

	result, err := call.Context(ctx).Do()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	entries := make([]gdriveEntry, 0, len(result.Files))
	for _, f := range result.Files {
		if f.MimeType == "application/vnd.google-apps.folder" {
			folderPrefix := req.Prefix + f.Name + "/"
			entries = append(entries, gdriveEntry{
				Type:    "dir",
				Name:    folderPrefix,
				Display: f.Name,
			})
		} else {
			var updated time.Time
			if f.ModifiedTime != "" {
				updated, _ = time.Parse(time.RFC3339, f.ModifiedTime)
			}
			// Name stores "prefix + fileID" so the backend can look up files by ID later.
			// Display shows the human-readable filename.
			entries = append(entries, gdriveEntry{
				Type:    "file",
				Name:    req.Prefix + f.Id,
				Display: f.Name,
				Size:    f.Size,
				Updated: updated,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"prefix":          req.Prefix,
		"entries":         entries,
		"next_page_token": result.NextPageToken,
	})
}

// ListGDriveObjects is kept for backward compat (flat listing).
func ListGDriveObjects(w http.ResponseWriter, r *http.Request) {
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

	srv, err := gdriveService(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type gdriveObject struct {
		Name    string    `json:"name"`
		Size    int64     `json:"size"`
		Updated time.Time `json:"updated"`
	}

	const maxResults = 1000
	var objects []gdriveObject
	var pageToken string

	for len(objects) < maxResults {
		q := fmt.Sprintf("'%s' in parents and trashed = false and mimeType != 'application/vnd.google-apps.folder'", req.Bucket)
		call := srv.Files.List().Q(q).
			Fields("nextPageToken, files(id, name, size, modifiedTime)").
			PageSize(200).
			SupportsAllDrives(true).
			IncludeItemsFromAllDrives(true)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		result, err := call.Context(ctx).Do()
		if err != nil {
			break
		}
		for _, f := range result.Files {
			var updated time.Time
			if f.ModifiedTime != "" {
				updated, _ = time.Parse(time.RFC3339, f.ModifiedTime)
			}
			objects = append(objects, gdriveObject{
				Name:    f.Name,
				Size:    f.Size,
				Updated: updated,
			})
			if len(objects) >= maxResults {
				break
			}
		}
		pageToken = result.NextPageToken
		if pageToken == "" {
			break
		}
	}
	if objects == nil {
		objects = []gdriveObject{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"objects":   objects,
		"truncated": len(objects) == maxResults,
	})
}

// extractFileID extracts the Drive file ID from the "name" field.
// The name format is "prefix/fileID" where prefix may be empty or end with "/".
func extractFileID(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

// GDriveDownloadURL returns a download URL for a Google Drive file.
func GDriveDownloadURL(w http.ResponseWriter, r *http.Request) {
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

	srv, err := gdriveService(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileID := extractFileID(req.Object)

	// Get file metadata to check if it's a Google Workspace file
	f, err := srv.Files.Get(fileID).Fields("id, name, mimeType, webContentLink").SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// For Google Workspace files (Docs, Sheets, etc.), export as PDF
	if strings.HasPrefix(f.MimeType, "application/vnd.google-apps.") {
		exportMime := "application/pdf"
		url := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s/export?mimeType=%s", fileID, exportMime)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"url": url})
		return
	}

	// For regular files, use the direct download link
	url := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?alt=media", fileID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"url": url})
}

// DeleteGDriveObject deletes a single Google Drive file.
func DeleteGDriveObject(w http.ResponseWriter, r *http.Request) {
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

	srv, err := gdriveService(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileID := extractFileID(req.Object)
	if err := srv.Files.Delete(fileID).SupportsAllDrives(true).Context(ctx).Do(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CopyGDriveObject copies (and optionally deletes) a Google Drive file — used for rename/move.
func CopyGDriveObject(w http.ResponseWriter, r *http.Request) {
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

	srv, err := gdriveService(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sourceID := extractFileID(req.Source)

	// For rename: just update the file name via Files.Update
	// Extract the desired new name from the destination
	newName := req.Destination
	if i := strings.LastIndex(newName, "/"); i >= 0 {
		newName = newName[i+1:]
	}

	if req.Delete {
		// Rename = update the file name in-place
		update := &drive.File{Name: newName}
		if _, err := srv.Files.Update(sourceID, update).SupportsAllDrives(true).Context(ctx).Do(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Pure copy: copy the file with a new name
		copyFile := &drive.File{Name: newName}
		if _, err := srv.Files.Copy(sourceID, copyFile).SupportsAllDrives(true).Context(ctx).Do(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// UploadGDriveObject uploads a file to Google Drive via multipart form.
func UploadGDriveObject(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	folderID := r.FormValue("bucket")
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

	srv, err := gdriveService(ctx, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Resolve the target parent folder
	parentID, err := resolveParentID(ctx, srv, folderID, prefix)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	driveFile := &drive.File{
		Name:     header.Filename,
		Parents:  []string{parentID},
		MimeType: mimeType,
	}

	created, err := srv.Files.Create(driveFile).SupportsAllDrives(true).Media(file).Context(ctx).Do()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"name": created.Name})
}

// GDriveBucketStats — Google Drive does not support efficient aggregate stats.
func GDriveBucketStats(w http.ResponseWriter, r *http.Request) {
	// Drain the request body so connections can be reused
	io.Copy(io.Discard, r.Body)
	r.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object_count": 0,
		"total_size":   0,
		"truncated":    false,
		"supported":    false,
	})
}

// GetGDriveMetadata returns metadata for a Google Drive file.
func GetGDriveMetadata(w http.ResponseWriter, r *http.Request) {
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

	srv, err := gdriveService(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileID := extractFileID(req.Object)
	f, err := srv.Files.Get(fileID).
		Fields("id, name, mimeType, size, modifiedTime, description, starred, md5Checksum, webViewLink").
		SupportsAllDrives(true).
		Context(ctx).Do()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var updated time.Time
	if f.ModifiedTime != "" {
		updated, _ = time.Parse(time.RFC3339, f.ModifiedTime)
	}

	md := map[string]string{}
	if f.Description != "" {
		md["description"] = f.Description
	}
	if f.WebViewLink != "" {
		md["web_view_link"] = f.WebViewLink
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"content_type":  f.MimeType,
		"cache_control": "",
		"metadata":      md,
		"size":          f.Size,
		"updated":       updated,
		"etag":          "",
		"md5":           f.Md5Checksum,
	})
}

// UpdateGDriveMetadata patches description on a Google Drive file.
func UpdateGDriveMetadata(w http.ResponseWriter, r *http.Request) {
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

	srv, err := gdriveService(ctx, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileID := extractFileID(req.Object)

	update := &drive.File{}
	if desc, ok := req.Metadata["description"]; ok {
		update.Description = desc
	}
	// MimeType can also be updated (for non-Google Workspace files)
	if req.ContentType != "" {
		update.MimeType = req.ContentType
	}

	if _, err := srv.Files.Update(fileID, update).SupportsAllDrives(true).Context(ctx).Do(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

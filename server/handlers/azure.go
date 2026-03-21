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

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

// ── helpers ──────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }
func i32Ptr(i int32) *int32   { return &i }

func azureCredsFromJSON(raw string) (accountName, accountKey string, err error) {
	var creds struct {
		AccountName string `json:"account_name"`
		AccountKey  string `json:"account_key"`
	}
	if err = json.Unmarshal([]byte(raw), &creds); err != nil {
		return
	}
	if creds.AccountName == "" || creds.AccountKey == "" {
		err = fmt.Errorf("missing account_name or account_key")
		return
	}
	return creds.AccountName, creds.AccountKey, nil
}

// azureCred creates a SharedKeyCredential from account credentials.
func azureCred(accountName, accountKey string) (*azcontainer.SharedKeyCredential, error) {
	return azcontainer.NewSharedKeyCredential(accountName, accountKey)
}

// azureContainerClient creates a container client for a specific container.
func azureContainerClient(accountName, accountKey, containerName string) (*azcontainer.Client, *azcontainer.SharedKeyCredential, error) {
	cred, err := azureCred(accountName, accountKey)
	if err != nil {
		return nil, nil, err
	}
	containerURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName)
	client, err := azcontainer.NewClientWithSharedKeyCredential(containerURL, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	return client, cred, nil
}

func testAzure(containerName, credentialsJSON string) error {
	accountName, accountKey, err := azureCredsFromJSON(credentialsJSON)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containerClient, _, err := azureContainerClient(accountName, accountKey, containerName)
	if err != nil {
		return err
	}
	pager := containerClient.NewListBlobsFlatPager(&azcontainer.ListBlobsFlatOptions{
		MaxResults: i32Ptr(1),
	})
	_, err = pager.NextPage(ctx)
	return err
}

func toAzureMetadata(m map[string]string) map[string]*string {
	result := make(map[string]*string, len(m))
	for k, v := range m {
		v := v
		result[k] = &v
	}
	return result
}

func fromAzureMetadata(m map[string]*string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}

// ── connection CRUD ───────────────────────────────────────────────

func ListAzure(w http.ResponseWriter, r *http.Request) {
	rows, err := appdb.DB.Query(
		"SELECT id, name, bucket, credentials, created_at FROM azure_connections ORDER BY created_at DESC",
	)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type AzureConnection struct {
		ID          int64     `json:"id"`
		Name        string    `json:"name"`
		Bucket      string    `json:"bucket"`
		Credentials string    `json:"credentials"`
		CreatedAt   time.Time `json:"created_at"`
	}

	conns := []AzureConnection{}
	for rows.Next() {
		var c AzureConnection
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

func CreateAzure(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := testAzure(req.Bucket, req.Credentials); err != nil {
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
		"INSERT INTO azure_connections (name, bucket, credentials, created_at) VALUES (?, ?, ?, ?)",
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

// AzureConnByID handles DELETE and PUT for /api/azure/connection/{id}.
func AzureConnByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		DeleteAzureConn(w, r)
	case http.MethodPut:
		UpdateAzureConn(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func DeleteAzureConn(w http.ResponseWriter, r *http.Request) {
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
	if _, err = appdb.DB.Exec("DELETE FROM azure_connections WHERE id = ?", id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func UpdateAzureConn(w http.ResponseWriter, r *http.Request) {
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
	if err := testAzure(req.Bucket, req.Credentials); err != nil {
		http.Error(w, fmt.Sprintf("test failed: %v", err), http.StatusBadRequest)
		return
	}
	encrypted, err := encryptCredentials(req.Credentials)
	if err != nil {
		http.Error(w, "failed to encrypt credentials", http.StatusInternalServerError)
		return
	}
	if _, err := appdb.DB.Exec(
		"UPDATE azure_connections SET name=?, bucket=?, credentials=? WHERE id=?",
		req.Name, req.Bucket, encrypted, id,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func TestAzure(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := testAzure(req.Bucket, req.Credentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ── bucket operations ─────────────────────────────────────────────

type azureEntry struct {
	Type    string    `json:"type"` // "dir" | "file"
	Name    string    `json:"name"`
	Display string    `json:"display"`
	Size    int64     `json:"size,omitempty"`
	Updated time.Time `json:"updated,omitempty"`
}

// BrowseAzureBucket lists blobs in a container at a given prefix (hierarchy).
func BrowseAzureBucket(w http.ResponseWriter, r *http.Request) {
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

	accountName, accountKey, err := azureCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	containerClient, _, err := azureContainerClient(accountName, accountKey, req.Bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	opts := &azcontainer.ListBlobsHierarchyOptions{
		Prefix:     strPtr(req.Prefix),
		MaxResults: i32Ptr(200),
	}
	if req.PageToken != "" {
		opts.Marker = strPtr(req.PageToken)
	}

	pager := containerClient.NewListBlobsHierarchyPager("/", opts)

	var entries []azureEntry
	nextToken := ""

	if pager.More() {
		page, pageErr := pager.NextPage(ctx)
		if pageErr != nil {
			http.Error(w, pageErr.Error(), http.StatusBadRequest)
			return
		}
		for _, p := range page.Segment.BlobPrefixes {
			if p.Name == nil {
				continue
			}
			display := strings.TrimSuffix(strings.TrimPrefix(*p.Name, req.Prefix), "/")
			entries = append(entries, azureEntry{Type: "dir", Name: *p.Name, Display: display})
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name == nil || *item.Name == req.Prefix {
				continue
			}
			display := strings.TrimPrefix(*item.Name, req.Prefix)
			var size int64
			if item.Properties != nil && item.Properties.ContentLength != nil {
				size = *item.Properties.ContentLength
			}
			var updated time.Time
			if item.Properties != nil && item.Properties.LastModified != nil {
				updated = *item.Properties.LastModified
			}
			entries = append(entries, azureEntry{Type: "file", Name: *item.Name, Display: display, Size: size, Updated: updated})
		}
		if page.NextMarker != nil && *page.NextMarker != "" {
			nextToken = *page.NextMarker
		}
	}

	if entries == nil {
		entries = []azureEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"prefix":          req.Prefix,
		"entries":         entries,
		"next_page_token": nextToken,
	})
}

// ListAzureObjects is a flat listing (backward compat).
func ListAzureObjects(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	accountName, accountKey, err := azureCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	containerClient, _, err := azureContainerClient(accountName, accountKey, req.Bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	type azureObject struct {
		Name    string    `json:"name"`
		Size    int64     `json:"size"`
		Updated time.Time `json:"updated"`
	}

	pager := containerClient.NewListBlobsFlatPager(&azcontainer.ListBlobsFlatOptions{
		MaxResults: i32Ptr(1000),
	})

	const maxResults = 1000
	var objects []azureObject
	for pager.More() && len(objects) < maxResults {
		page, pageErr := pager.NextPage(ctx)
		if pageErr != nil {
			break
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name == nil {
				continue
			}
			var size int64
			if item.Properties != nil && item.Properties.ContentLength != nil {
				size = *item.Properties.ContentLength
			}
			var updated time.Time
			if item.Properties != nil && item.Properties.LastModified != nil {
				updated = *item.Properties.LastModified
			}
			objects = append(objects, azureObject{Name: *item.Name, Size: size, Updated: updated})
		}
	}
	if objects == nil {
		objects = []azureObject{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"objects":   objects,
		"truncated": len(objects) == maxResults,
	})
}

// AzureDownloadURL generates a SAS download URL (default 15 min; customisable via expires_in seconds).
func AzureDownloadURL(w http.ResponseWriter, r *http.Request) {
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

	accountName, accountKey, err := azureCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	expiry := time.Duration(req.ExpiresIn) * time.Second
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}

	cred, err := azureCred(accountName, accountKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	perms := sas.BlobPermissions{Read: true}
	sasValues := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     time.Now().UTC().Add(-10 * time.Second),
		ExpiryTime:    time.Now().UTC().Add(expiry),
		Permissions:   perms.String(),
		ContainerName: req.Bucket,
		BlobName:      req.Object,
	}
	qp, err := sasValues.SignWithSharedKey(cred)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sasURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s",
		accountName, req.Bucket, req.Object, qp.Encode())

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"url": sasURL})
}

// DeleteAzureObject deletes a single Azure blob.
func DeleteAzureObject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Object      string `json:"object"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	accountName, accountKey, err := azureCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	containerClient, _, err := azureContainerClient(accountName, accountKey, req.Bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	blobClient := containerClient.NewBlobClient(req.Object)
	if _, err = blobClient.Delete(ctx, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CopyAzureObject copies (and optionally deletes) an Azure blob — used for rename/move.
func CopyAzureObject(w http.ResponseWriter, r *http.Request) {
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

	accountName, accountKey, err := azureCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	containerClient, _, err := azureContainerClient(accountName, accountKey, req.Bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	srcURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		accountName, req.Bucket, req.Source)
	destBlobClient := containerClient.NewBlobClient(req.Destination)
	if _, err = destBlobClient.StartCopyFromURL(ctx, srcURL, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.Delete {
		srcBlobClient := containerClient.NewBlobClient(req.Source)
		if _, err = srcBlobClient.Delete(ctx, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// UploadAzureObject uploads a file to Azure Blob Storage via multipart form.
func UploadAzureObject(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bucketName := r.FormValue("bucket")
	rawCreds := r.FormValue("credentials")
	prefix := r.FormValue("prefix")

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	accountName, accountKey, err := azureCredsFromJSON(rawCreds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	containerClient, _, err := azureContainerClient(accountName, accountKey, bucketName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	objectName := prefix + header.Filename
	blobClient := containerClient.NewBlockBlobClient(objectName)
	_, err = blobClient.UploadBuffer(ctx, data, &blockblob.UploadBufferOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: strPtr(contentType),
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"name": objectName})
}

// AzureBucketStats returns sampled object count and total size.
func AzureBucketStats(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	accountName, accountKey, err := azureCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	containerClient, _, err := azureContainerClient(accountName, accountKey, req.Bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pager := containerClient.NewListBlobsFlatPager(&azcontainer.ListBlobsFlatOptions{
		MaxResults: i32Ptr(1000),
	})

	const maxSample = 10000
	var count, totalSize int64
	for pager.More() && count < maxSample {
		page, pageErr := pager.NextPage(ctx)
		if pageErr != nil {
			break
		}
		for _, item := range page.Segment.BlobItems {
			count++
			if item.Properties != nil && item.Properties.ContentLength != nil {
				totalSize += *item.Properties.ContentLength
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object_count": count,
		"total_size":   totalSize,
		"truncated":    count == maxSample,
	})
}

// GetAzureMetadata returns full metadata for an Azure blob.
func GetAzureMetadata(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Object      string `json:"object"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	accountName, accountKey, err := azureCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containerClient, _, err := azureContainerClient(accountName, accountKey, req.Bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	blobClient := containerClient.NewBlobClient(req.Object)
	resp, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	contentType := ""
	if resp.ContentType != nil {
		contentType = *resp.ContentType
	}
	cacheControl := ""
	if resp.CacheControl != nil {
		cacheControl = *resp.CacheControl
	}
	etag := ""
	if resp.ETag != nil {
		etag = strings.Trim(string(*resp.ETag), `"`)
	}
	var size int64
	if resp.ContentLength != nil {
		size = *resp.ContentLength
	}
	var updated time.Time
	if resp.LastModified != nil {
		updated = *resp.LastModified
	}
	md := fromAzureMetadata(resp.Metadata)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"content_type":  contentType,
		"cache_control": cacheControl,
		"metadata":      md,
		"size":          size,
		"updated":       updated,
		"etag":          etag,
	})
}

// DeletePrefixAzure deletes all blobs under a given prefix (recursive folder delete).
func DeletePrefixAzure(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Prefix      string `json:"prefix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	accountName, accountKey, err := azureCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	containerClient, _, err := azureContainerClient(accountName, accountKey, req.Bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pager := containerClient.NewListBlobsFlatPager(&azcontainer.ListBlobsFlatOptions{
		Prefix: strPtr(req.Prefix),
	})
	deleted := 0
	for pager.More() {
		page, pageErr := pager.NextPage(ctx)
		if pageErr != nil {
			http.Error(w, pageErr.Error(), http.StatusInternalServerError)
			return
		}
		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			containerClient.NewBlobClient(*blob.Name).Delete(ctx, nil)
			deleted++
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"deleted": deleted})
}

// UpdateAzureMetadata patches an Azure blob's metadata in-place (no copy-to-self needed).
func UpdateAzureMetadata(w http.ResponseWriter, r *http.Request) {
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

	accountName, accountKey, err := azureCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	containerClient, _, err := azureContainerClient(accountName, accountKey, req.Bucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	blobClient := containerClient.NewBlobClient(req.Object)

	// Update HTTP headers (ContentType, CacheControl)
	headers := blob.HTTPHeaders{}
	if req.ContentType != "" {
		headers.BlobContentType = strPtr(req.ContentType)
	}
	if req.CacheControl != "" {
		headers.BlobCacheControl = strPtr(req.CacheControl)
	}
	if _, err = blobClient.SetHTTPHeaders(ctx, headers, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update custom metadata
	if req.Metadata != nil {
		azMeta := toAzureMetadata(req.Metadata)
		if _, err = blobClient.SetMetadata(ctx, azMeta, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsauth "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

// ── helpers ──────────────────────────────────────────────────────

func obsCredsFromJSON(raw string) (map[string]string, error) {
	var creds map[string]string
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return nil, err
	}
	return creds, nil
}

// obsS3Client builds an S3-compatible client pointed at Huawei OBS.
// The "endpoint" key is required (e.g. https://obs.cn-north-4.myhuaweicloud.com).
func obsS3Client(ctx context.Context, creds map[string]string) (*s3.Client, error) {
	accessKey := creds["access_key_id"]
	secretKey := creds["secret_access_key"]
	endpoint := creds["endpoint"]
	region := creds["region"]
	if region == "" {
		region = "cn-north-4"
	}
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("missing access_key_id or secret_access_key")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("missing endpoint (e.g. https://obs.cn-north-4.myhuaweicloud.com)")
	}

	provider := awsauth.NewStaticCredentialsProvider(accessKey, secretKey, "")
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(provider),
	)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		// Huawei OBS uses virtual-hosted style by default:
		// https://<bucket>.obs.<region>.myhuaweicloud.com
		o.UsePathStyle = false
	}), nil
}

func testOBS(bucket, credentialsJSON string) error {
	creds, err := obsCredsFromJSON(credentialsJSON)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := obsS3Client(ctx, creds)
	if err != nil {
		return err
	}
	// Use ListObjectsV2 (obs:ListBucket) instead of HeadBucket (obs:GetBucketMetadata)
	// because list permission is more commonly granted in Huawei OBS IAM policies.
	_, err = client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int32(1),
	})
	return err
}

// ── connection CRUD ───────────────────────────────────────────────

func ListHuawei(w http.ResponseWriter, r *http.Request) {
	rows, err := appdb.DB.Query(
		"SELECT id, name, bucket, credentials, created_at FROM huawei_connections ORDER BY created_at DESC",
	)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type HuaweiConnection struct {
		ID          int64     `json:"id"`
		Name        string    `json:"name"`
		Bucket      string    `json:"bucket"`
		Credentials string    `json:"credentials"`
		CreatedAt   time.Time `json:"created_at"`
	}

	conns := []HuaweiConnection{}
	for rows.Next() {
		var c HuaweiConnection
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

func CreateHuawei(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := testOBS(req.Bucket, req.Credentials); err != nil {
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
		"INSERT INTO huawei_connections (name, bucket, credentials, created_at) VALUES (?, ?, ?, ?)",
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

// HuaweiConnByID handles DELETE and PUT for /api/huawei/connection/{id}.
func HuaweiConnByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		DeleteHuaweiConn(w, r)
	case http.MethodPut:
		UpdateHuaweiConn(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func DeleteHuaweiConn(w http.ResponseWriter, r *http.Request) {
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
	if _, err = appdb.DB.Exec("DELETE FROM huawei_connections WHERE id = ?", id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func UpdateHuaweiConn(w http.ResponseWriter, r *http.Request) {
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
	if err := testOBS(req.Bucket, req.Credentials); err != nil {
		http.Error(w, fmt.Sprintf("test failed: %v", err), http.StatusBadRequest)
		return
	}
	encrypted, err := encryptCredentials(req.Credentials)
	if err != nil {
		http.Error(w, "failed to encrypt credentials", http.StatusInternalServerError)
		return
	}
	if _, err := appdb.DB.Exec(
		"UPDATE huawei_connections SET name=?, bucket=?, credentials=? WHERE id=?",
		req.Name, req.Bucket, encrypted, id,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func TestHuawei(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := testOBS(req.Bucket, req.Credentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ── bucket operations ─────────────────────────────────────────────

type obsEntry struct {
	Type    string    `json:"type"` // "dir" | "file"
	Name    string    `json:"name"`
	Display string    `json:"display"`
	Size    int64     `json:"size,omitempty"`
	Updated time.Time `json:"updated,omitempty"`
}

// BrowseHuaweiBucket lists entries at a given prefix with pagination.
func BrowseHuaweiBucket(w http.ResponseWriter, r *http.Request) {
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

	creds, err := obsCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := obsS3Client(ctx, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(req.Bucket),
		Prefix:    aws.String(req.Prefix),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int32(200),
	}
	if req.PageToken != "" {
		input.ContinuationToken = aws.String(req.PageToken)
	}

	result, err := client.ListObjectsV2(ctx, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var entries []obsEntry
	for _, p := range result.CommonPrefixes {
		if p.Prefix == nil {
			continue
		}
		display := strings.TrimSuffix(strings.TrimPrefix(*p.Prefix, req.Prefix), "/")
		entries = append(entries, obsEntry{Type: "dir", Name: *p.Prefix, Display: display})
	}
	for _, obj := range result.Contents {
		if obj.Key == nil || *obj.Key == req.Prefix {
			continue
		}
		display := strings.TrimPrefix(*obj.Key, req.Prefix)
		var size int64
		if obj.Size != nil {
			size = *obj.Size
		}
		var updated time.Time
		if obj.LastModified != nil {
			updated = *obj.LastModified
		}
		entries = append(entries, obsEntry{Type: "file", Name: *obj.Key, Display: display, Size: size, Updated: updated})
	}
	if entries == nil {
		entries = []obsEntry{}
	}

	nextToken := ""
	if result.NextContinuationToken != nil {
		nextToken = *result.NextContinuationToken
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"prefix":          req.Prefix,
		"entries":         entries,
		"next_page_token": nextToken,
	})
}

// ListHuaweiObjects is a flat listing (backward compat).
func ListHuaweiObjects(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	creds, err := obsCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := obsS3Client(ctx, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	type obsObject struct {
		Name    string    `json:"name"`
		Size    int64     `json:"size"`
		Updated time.Time `json:"updated"`
	}

	const maxResults = 1000
	var objects []obsObject
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket:  aws.String(req.Bucket),
		MaxKeys: aws.Int32(maxResults),
	})
	for paginator.HasMorePages() && len(objects) < maxResults {
		page, pageErr := paginator.NextPage(ctx)
		if pageErr != nil {
			break
		}
		for _, obj := range page.Contents {
			var updated time.Time
			if obj.LastModified != nil {
				updated = *obj.LastModified
			}
			var size int64
			if obj.Size != nil {
				size = *obj.Size
			}
			var name string
			if obj.Key != nil {
				name = *obj.Key
			}
			objects = append(objects, obsObject{Name: name, Size: size, Updated: updated})
		}
	}
	if objects == nil {
		objects = []obsObject{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"objects":   objects,
		"truncated": len(objects) == maxResults,
	})
}

// HuaweiDownloadURL generates a presigned GET URL (default 15 min; customisable via expires_in seconds).
func HuaweiDownloadURL(w http.ResponseWriter, r *http.Request) {
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

	creds, err := obsCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	expiry := time.Duration(req.ExpiresIn) * time.Second
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := obsS3Client(ctx, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	psClient := s3.NewPresignClient(client)
	presigned, err := psClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(req.Bucket),
		Key:    aws.String(req.Object),
	}, func(o *s3.PresignOptions) { o.Expires = expiry })
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"url": presigned.URL})
}

// DeleteHuaweiObject deletes a single OBS object.
func DeleteHuaweiObject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Object      string `json:"object"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	creds, err := obsCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := obsS3Client(ctx, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(req.Bucket),
		Key:    aws.String(req.Object),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CopyHuaweiObject copies (and optionally deletes) an OBS object — used for rename/move.
func CopyHuaweiObject(w http.ResponseWriter, r *http.Request) {
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

	creds, err := obsCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := obsS3Client(ctx, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	copySource := req.Bucket + "/" + req.Source
	if _, err := client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(req.Bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(req.Destination),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.Delete {
		if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(req.Bucket),
			Key:    aws.String(req.Source),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// UploadHuaweiObject uploads a file to OBS via multipart form.
func UploadHuaweiObject(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bucket := r.FormValue("bucket")
	rawCreds := r.FormValue("credentials")
	prefix := r.FormValue("prefix")

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	creds, err := obsCredsFromJSON(rawCreds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client, err := obsS3Client(ctx, creds)
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
	if _, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(objectName),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   aws.String(contentType),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"name": objectName})
}

// HuaweiBucketStats returns sampled object count and total size.
func HuaweiBucketStats(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	creds, err := obsCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := obsS3Client(ctx, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	const maxSample = 10000
	var count, totalSize int64
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket:  aws.String(req.Bucket),
		MaxKeys: aws.Int32(1000),
	})
	for paginator.HasMorePages() && count < maxSample {
		page, pageErr := paginator.NextPage(ctx)
		if pageErr != nil {
			break
		}
		for _, obj := range page.Contents {
			count++
			if obj.Size != nil {
				totalSize += *obj.Size
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

// GetHuaweiMetadata returns full metadata for an OBS object via HeadObject.
func GetHuaweiMetadata(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Object      string `json:"object"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	creds, err := obsCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := obsS3Client(ctx, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(req.Bucket),
		Key:    aws.String(req.Object),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	contentType := ""
	if head.ContentType != nil {
		contentType = *head.ContentType
	}
	cacheControl := ""
	if head.CacheControl != nil {
		cacheControl = *head.CacheControl
	}
	etag := ""
	if head.ETag != nil {
		etag = strings.Trim(*head.ETag, `"`)
	}
	var size int64
	if head.ContentLength != nil {
		size = *head.ContentLength
	}
	var updated time.Time
	if head.LastModified != nil {
		updated = *head.LastModified
	}
	md := head.Metadata
	if md == nil {
		md = map[string]string{}
	}

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

// DeletePrefixHuawei deletes all objects under a given prefix (recursive folder delete).
func DeletePrefixHuawei(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Prefix      string `json:"prefix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	creds, err := obsCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client, err := obsS3Client(ctx, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	deleted := 0
	var token *string
	for {
		out, listErr := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(req.Bucket),
			Prefix:            aws.String(req.Prefix),
			ContinuationToken: token,
		})
		if listErr != nil {
			http.Error(w, listErr.Error(), http.StatusInternalServerError)
			return
		}
		if len(out.Contents) > 0 {
			ids := make([]types.ObjectIdentifier, len(out.Contents))
			for i, o := range out.Contents {
				ids[i] = types.ObjectIdentifier{Key: o.Key}
			}
			client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(req.Bucket),
				Delete: &types.Delete{Objects: ids},
			})
		}
		deleted += len(out.Contents)
		if out.IsTruncated == nil || !*out.IsTruncated {
			break
		}
		token = out.NextContinuationToken
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"deleted": deleted})
}

// UpdateHuaweiMetadata patches an OBS object's metadata via copy-to-self.
func UpdateHuaweiMetadata(w http.ResponseWriter, r *http.Request) {
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

	creds, err := obsCredsFromJSON(req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := obsS3Client(ctx, creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	copySource := req.Bucket + "/" + req.Object
	input := &s3.CopyObjectInput{
		Bucket:            aws.String(req.Bucket),
		CopySource:        aws.String(copySource),
		Key:               aws.String(req.Object),
		MetadataDirective: types.MetadataDirectiveReplace,
		Metadata:          req.Metadata,
	}
	if req.ContentType != "" {
		input.ContentType = aws.String(req.ContentType)
	}
	if req.CacheControl != "" {
		input.CacheControl = aws.String(req.CacheControl)
	}

	if _, err := client.CopyObject(ctx, input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

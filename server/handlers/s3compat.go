package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

// S3Provider encapsulates the provider-specific logic for an S3-compatible backend.
type S3Provider struct {
	Name       string // e.g. "aws", "alibaba", "huawei", "b2", "do"
	Table      string // DB table name, e.g. "aws_connections"
	CredsFunc  func(raw string) (map[string]string, error)
	ClientFunc func(ctx context.Context, creds map[string]string) (*s3.Client, error)
	TestFunc   func(bucket, credJSON string) error
}

// s3Entry is a generic directory/file entry for S3-compatible providers.
type s3Entry struct {
	Type    string    `json:"type"` // "dir" | "file"
	Name    string    `json:"name"`
	Display string    `json:"display"`
	Size    int64     `json:"size,omitempty"`
	Updated time.Time `json:"updated,omitempty"`
}

// ── Connection CRUD ──────────────────────────────────────────────

func (p *S3Provider) ListConnections() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := appdb.DB.Query(
			fmt.Sprintf("SELECT id, name, bucket, credentials, created_at FROM %s ORDER BY created_at DESC", p.Table),
		)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type connection struct {
			ID          int64     `json:"id"`
			Name        string    `json:"name"`
			Bucket      string    `json:"bucket"`
			Credentials string    `json:"credentials"`
			CreatedAt   time.Time `json:"created_at"`
		}
		conns := []connection{}
		for rows.Next() {
			var c connection
			var created string
			if err := rows.Scan(&c.ID, &c.Name, &c.Bucket, &c.Credentials, &created); err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			c.Credentials, _ = decryptCredentials(c.Credentials)
			var parseErr error
			c.CreatedAt, parseErr = time.Parse(time.RFC3339, created)
			if parseErr != nil {
				log.Printf("s3compat: ListConnections: failed to parse created_at %q for id %d: %v", created, c.ID, parseErr)
			}
			conns = append(conns, c)
		}
		if err := rows.Err(); err != nil {
			jsonError(w, "database error", http.StatusInternalServerError)
			return
		}
		jsonOK(w, conns)
	}
}

func (p *S3Provider) CreateConnection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string `json:"name"`
			Bucket      string `json:"bucket"`
			Credentials string `json:"credentials"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := requireFields(map[string]string{"name": req.Name, "bucket": req.Bucket, "credentials": req.Credentials}); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := p.TestFunc(req.Bucket, req.Credentials); err != nil {
			jsonError(w, "test failed: "+err.Error(), http.StatusBadRequest)
			return
		}
		encrypted, err := encryptCredentials(req.Credentials)
		if err != nil {
			jsonError(w, "encryption error", http.StatusInternalServerError)
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		res, err := appdb.DB.Exec(
			fmt.Sprintf("INSERT INTO %s (name, bucket, credentials, created_at) VALUES (?, ?, ?, ?)", p.Table),
			req.Name, req.Bucket, encrypted, now,
		)
		if err != nil {
			jsonError(w, "database error", http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		jsonOK(w, map[string]any{"id": id})
	}
}

func (p *S3Provider) ConnByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			p.deleteConnection(w, r)
		case http.MethodPut:
			p.updateConnection(w, r)
		default:
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func (p *S3Provider) deleteConnection(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, 4)
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if _, err = appdb.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", p.Table), id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (p *S3Provider) updateConnection(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, 4)
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := requireFields(map[string]string{"name": req.Name, "bucket": req.Bucket, "credentials": req.Credentials}); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := p.TestFunc(req.Bucket, req.Credentials); err != nil {
		jsonError(w, "test failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	encrypted, encErr := encryptCredentials(req.Credentials)
	if encErr != nil {
		jsonError(w, "encryption error", http.StatusInternalServerError)
		return
	}
	if _, err := appdb.DB.Exec(
		fmt.Sprintf("UPDATE %s SET name=?, bucket=?, credentials=? WHERE id=?", p.Table),
		req.Name, req.Bucket, encrypted, id,
	); err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (p *S3Provider) TestConnection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Bucket      string `json:"bucket"`
			Credentials string `json:"credentials"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := p.TestFunc(req.Bucket, req.Credentials); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		jsonOK(w, map[string]string{"status": "ok"})
	}
}

// ── helpers for connection-ID based lookups ──────────────────────

func (p *S3Provider) getClient(ctx context.Context, connID int64) (*s3.Client, string, error) {
	bucket, credJSON, err := lookupConnection(p.Table, connID)
	if err != nil {
		return nil, "", fmt.Errorf("connection not found: %w", err)
	}
	creds, err := p.CredsFunc(credJSON)
	if err != nil {
		return nil, "", err
	}
	client, err := p.ClientFunc(ctx, creds)
	if err != nil {
		return nil, "", err
	}
	return client, bucket, nil
}

// ── Bucket operations ────────────────────────────────────────────

func (p *S3Provider) Browse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ConnectionID int64  `json:"connection_id"`
			Prefix       string `json:"prefix"`
			PageToken    string `json:"page_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client, bucket, err := p.getClient(ctx, req.ConnectionID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		input := &s3.ListObjectsV2Input{
			Bucket:    aws.String(bucket),
			Prefix:    aws.String(req.Prefix),
			Delimiter: aws.String("/"),
			MaxKeys:   aws.Int32(int32(DefaultPageSize)),
		}
		if req.PageToken != "" {
			input.ContinuationToken = aws.String(req.PageToken)
		}

		result, err := client.ListObjectsV2(ctx, input)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		var entries []s3Entry
		for _, pfx := range result.CommonPrefixes {
			if pfx.Prefix == nil {
				continue
			}
			display := strings.TrimSuffix(strings.TrimPrefix(*pfx.Prefix, req.Prefix), "/")
			entries = append(entries, s3Entry{Type: "dir", Name: *pfx.Prefix, Display: display})
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
			entries = append(entries, s3Entry{Type: "file", Name: *obj.Key, Display: display, Size: size, Updated: updated})
		}
		if entries == nil {
			entries = []s3Entry{}
		}

		nextToken := ""
		if result.NextContinuationToken != nil {
			nextToken = *result.NextContinuationToken
		}

		jsonOK(w, map[string]any{
			"prefix":          req.Prefix,
			"entries":         entries,
			"next_page_token": nextToken,
		})
	}
}

func (p *S3Provider) ListObjects() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ConnectionID int64 `json:"connection_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client, bucket, err := p.getClient(ctx, req.ConnectionID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		type s3Object struct {
			Name    string    `json:"name"`
			Size    int64     `json:"size"`
			Updated time.Time `json:"updated"`
		}

	const maxResults = 1000
	var objects []s3Object
	var partial bool
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int32(maxResults),
	})
	for paginator.HasMorePages() && len(objects) < maxResults {
		page, pageErr := paginator.NextPage(ctx)
		if pageErr != nil {
			log.Printf("s3compat: ListObjects: pagination error: %v", pageErr)
			partial = true
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
			objects = append(objects, s3Object{Name: name, Size: size, Updated: updated})
		}
	}
	if objects == nil {
		objects = []s3Object{}
	}
	jsonOK(w, map[string]any{
		"objects":   objects,
		"truncated": len(objects) == maxResults,
		"partial":   partial,
	})
	}
}

func (p *S3Provider) Download() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ConnectionID int64  `json:"connection_id"`
			Object       string `json:"object"`
			ExpiresIn    int64  `json:"expires_in"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

	expiry := time.Duration(req.ExpiresIn) * time.Second
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}
	const maxExpiry = 7 * 24 * time.Hour
	if expiry > maxExpiry {
		expiry = maxExpiry
	}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, bucket, err := p.getClient(ctx, req.ConnectionID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		psClient := s3.NewPresignClient(client)
		presigned, err := psClient.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(req.Object),
		}, func(o *s3.PresignOptions) { o.Expires = expiry })
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, map[string]string{"url": presigned.URL})
	}
}

func (p *S3Provider) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ConnectionID int64  `json:"connection_id"`
			Object       string `json:"object"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		client, bucket, err := p.getClient(ctx, req.ConnectionID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		if _, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(req.Object),
		}); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (p *S3Provider) Copy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ConnectionID int64  `json:"connection_id"`
			Source       string `json:"source"`
			Destination  string `json:"destination"`
			DeleteSource bool   `json:"delete_source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client, bucket, err := p.getClient(ctx, req.ConnectionID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

	copySource := bucket + "/" + url.PathEscape(req.Source)
	if _, err := client.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket:     aws.String(bucket),
			CopySource: aws.String(copySource),
			Key:        aws.String(req.Destination),
		}); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if req.DeleteSource {
			if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(req.Source),
			}); err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (p *S3Provider) Upload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		connIDStr := r.FormValue("connection_id")
		prefix := r.FormValue("prefix")

		var connID int64
		if _, err := fmt.Sscanf(connIDStr, "%d", &connID); err != nil {
			jsonError(w, "invalid connection_id", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		if header.Size > MaxUploadSize {
			jsonError(w, "file too large (max 500MB)", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		client, bucket, err := p.getClient(ctx, connID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
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
		Body:          file,
		ContentLength: aws.Int64(header.Size),
		ContentType:   aws.String(contentType),
	}); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, map[string]string{"name": objectName})
	}
}

func (p *S3Provider) Stats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ConnectionID int64 `json:"connection_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		client, bucket, err := p.getClient(ctx, req.ConnectionID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		const maxSample = 10000
		var count, totalSize int64
		paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
			Bucket:  aws.String(bucket),
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
		jsonOK(w, map[string]any{
			"object_count": count,
			"total_size":   totalSize,
			"truncated":    count == maxSample,
		})
	}
}

func (p *S3Provider) GetMetadata() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ConnectionID int64  `json:"connection_id"`
			Object       string `json:"object"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, bucket, err := p.getClient(ctx, req.ConnectionID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(req.Object),
		})
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
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

		jsonOK(w, map[string]any{
			"content_type":  contentType,
			"cache_control": cacheControl,
			"metadata":      md,
			"size":          size,
			"updated":       updated,
			"etag":          etag,
		})
	}
}

func (p *S3Provider) UpdateMetadata() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ConnectionID int64             `json:"connection_id"`
			Object       string            `json:"object"`
			ContentType  string            `json:"content_type"`
			CacheControl string            `json:"cache_control"`
			Metadata     map[string]string `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client, bucket, err := p.getClient(ctx, req.ConnectionID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

	copySource := bucket + "/" + url.PathEscape(req.Object)
	input := &s3.CopyObjectInput{
			Bucket:            aws.String(bucket),
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
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (p *S3Provider) DeletePrefix() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ConnectionID int64  `json:"connection_id"`
			Prefix       string `json:"prefix"`
		}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Prefix) == "" {
		jsonError(w, "prefix is required to prevent accidental deletion of entire bucket", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client, bucket, err := p.getClient(ctx, req.ConnectionID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	deleted := 0
		var token *string
		for {
			out, listErr := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
				Bucket:            aws.String(bucket),
				Prefix:            aws.String(req.Prefix),
				ContinuationToken: token,
			})
			if listErr != nil {
				jsonError(w, listErr.Error(), http.StatusInternalServerError)
				return
			}
			if len(out.Contents) > 0 {
				ids := make([]types.ObjectIdentifier, len(out.Contents))
				for i, o := range out.Contents {
					ids[i] = types.ObjectIdentifier{Key: o.Key}
				}
				if _, delErr := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
					Bucket: aws.String(bucket),
					Delete: &types.Delete{Objects: ids},
				}); delErr != nil {
					jsonError(w, delErr.Error(), http.StatusInternalServerError)
					return
				}
			}
			deleted += len(out.Contents)
			if out.IsTruncated == nil || !*out.IsTruncated {
				break
			}
			token = out.NextContinuationToken
		}
		jsonOK(w, map[string]int{"deleted": deleted})
	}
}

// ── Download/Upload helpers for transfer.go and zip.go ──────────

// DownloadObject downloads an object from this S3 provider (used by transfer/zip).
func (p *S3Provider) DownloadObject(ctx context.Context, connID int64, object string) ([]byte, string, error) {
	client, bucket, err := p.getClient(ctx, connID)
	if err != nil {
		return nil, "", err
	}
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
	})
	if err != nil {
		return nil, "", err
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, "", err
	}
	ct := "application/octet-stream"
	if out.ContentType != nil && *out.ContentType != "" {
		ct = *out.ContentType
	}
	return data, ct, nil
}

// UploadObject uploads data to this S3 provider (used by transfer).
func (p *S3Provider) UploadObject(ctx context.Context, connID int64, key string, data []byte, contentType string) error {
	client, bucket, err := p.getClient(ctx, connID)
	if err != nil {
		return err
	}
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   aws.String(contentType),
	})
	return err
}

// ListKeys lists all object keys under a prefix (used by zip).
func (p *S3Provider) ListKeys(ctx context.Context, connID int64, prefix string) ([]string, error) {
	client, bucket, err := p.getClient(ctx, connID)
	if err != nil {
		return nil, err
	}
	var keys []string
	var token *string
	for {
		out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: token,
		})
		if err != nil {
			return nil, err
		}
		for _, o := range out.Contents {
			if o.Key != nil {
				keys = append(keys, *o.Key)
			}
		}
		if out.IsTruncated == nil || !*out.IsTruncated {
			break
		}
		token = out.NextContinuationToken
	}
	return keys, nil
}

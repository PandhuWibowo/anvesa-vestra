package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	awslib "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	driveapi "google.golang.org/api/drive/v3"
)

// ── Transfer progress store ────────────────────────────────────────────────

type transferProgress struct {
	ID          string  `json:"id"`
	Filename    string  `json:"filename"`
	Loaded      int64   `json:"loaded"`
	Total       int64   `json:"total"`
	Done        bool    `json:"done"`
	Error       string  `json:"error,omitempty"`
	Destination string  `json:"destination,omitempty"`
}

var (
	transfersMu sync.Mutex
	transfers   = map[string]*transferProgress{}
)

func setTransfer(p *transferProgress) {
	transfersMu.Lock()
	transfers[p.ID] = p
	transfersMu.Unlock()
}

func getTransfer(id string) (*transferProgress, bool) {
	transfersMu.Lock()
	defer transfersMu.Unlock()
	p, ok := transfers[id]
	return p, ok
}

func deleteTransfer(id string) {
	transfersMu.Lock()
	delete(transfers, id)
	transfersMu.Unlock()
}

// progressReader wraps an io.Reader and tracks bytes read.
type progressReader struct {
	r       io.Reader
	prog    *transferProgress
	mu      *sync.Mutex
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	if n > 0 {
		pr.mu.Lock()
		pr.prog.Loaded += int64(n)
		pr.mu.Unlock()
	}
	return n, err
}

// TransferRequest uses connection IDs instead of raw credentials.
type TransferRequest struct {
	SrcProvider     string `json:"src_provider"`
	SrcConnectionID int64  `json:"src_connection_id"`
	SrcObject       string `json:"src_object"`
	DstProvider     string `json:"dst_provider"`
	DstConnectionID int64  `json:"dst_connection_id"`
	DstPrefix       string `json:"dst_prefix"`
	// Legacy fields (deprecated, kept for backward compat)
	SrcBucket      string `json:"src_bucket"`
	SrcCredentials string `json:"src_credentials"`
	DstBucket      string `json:"dst_bucket"`
	DstCredentials string `json:"dst_credentials"`
}

func TransferObject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r = limitBody(r, MaxBodySize)
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	srcBucket, srcCreds, err := resolveProviderCreds(req.SrcProvider, req.SrcConnectionID, req.SrcBucket, req.SrcCredentials)
	if err != nil {
		jsonError(w, "source connection error: "+err.Error(), http.StatusBadRequest)
		return
	}

	dstBucket, dstCreds, err := resolveProviderCreds(req.DstProvider, req.DstConnectionID, req.DstBucket, req.DstCredentials)
	if err != nil {
		jsonError(w, "destination connection error: "+err.Error(), http.StatusBadRequest)
		return
	}

	filename := path.Base(req.SrcObject)
	prefix := strings.TrimSuffix(req.DstPrefix, "/")
	var destKey string
	if prefix == "" {
		destKey = filename
	} else {
		destKey = prefix + "/" + filename
	}

	// Generate a transfer ID and register progress tracking.
	transferID := fmt.Sprintf("%d", time.Now().UnixNano())
	prog := &transferProgress{
		ID:       transferID,
		Filename: filename,
		Total:    -1, // unknown until download completes
	}
	setTransfer(prog)

	// Run the transfer in the background.
	go func() {
		defer func() {
			// Auto-clean after 5 minutes.
			time.AfterFunc(5*time.Minute, func() { deleteTransfer(transferID) })
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		rc, contentType, err := streamDownloadObject(ctx, req.SrcProvider, srcBucket, srcCreds, req.SrcObject)
		if err != nil {
			log.Printf("transfer: download failed (provider=%s bucket=%s object=%s): %v", req.SrcProvider, srcBucket, req.SrcObject, err)
			transfersMu.Lock()
			prog.Error = "download failed: " + err.Error()
			prog.Done = true
			transfersMu.Unlock()
			return
		}
		defer rc.Close()

		// Wrap reader to track bytes as they are read from source.
		mu := &sync.Mutex{}
		tracked := &progressReader{r: rc, prog: prog, mu: mu}

		data, err := io.ReadAll(tracked)
		if err != nil {
			transfersMu.Lock()
			prog.Error = "read failed: " + err.Error()
			prog.Done = true
			transfersMu.Unlock()
			return
		}

		// Now we know the total size.
		transfersMu.Lock()
		prog.Total = int64(len(data))
		prog.Loaded = int64(len(data)) // download complete
		transfersMu.Unlock()

		if err := streamUploadObject(ctx, req.DstProvider, dstBucket, dstCreds, destKey, bytes.NewReader(data), int64(len(data)), contentType); err != nil {
			log.Printf("transfer: upload failed (provider=%s bucket=%s key=%s): %v", req.DstProvider, dstBucket, destKey, err)
			transfersMu.Lock()
			prog.Error = "upload failed: " + err.Error()
			prog.Done = true
			transfersMu.Unlock()
			return
		}

		transfersMu.Lock()
		prog.Done = true
		prog.Destination = destKey
		transfersMu.Unlock()
	}()

	jsonOK(w, map[string]string{"transfer_id": transferID, "destination": destKey})
}

// TransferProgress streams Server-Sent Events for a transfer job.
func TransferProgress(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		jsonError(w, "missing id", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	sendEvent := func(p *transferProgress) {
		data, _ := json.Marshal(p)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(11 * time.Minute)
	defer deadline.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-deadline.C:
			return
		case <-ticker.C:
			p, ok := getTransfer(id)
			if !ok {
				return
			}
			sendEvent(p)
			if p.Done {
				return
			}
		}
	}
}

// resolveProviderCreds resolves bucket/credentials from either a connection ID or legacy direct credentials.
func resolveProviderCreds(provider string, connID int64, legacyBucket, legacyCreds string) (bucket, creds string, err error) {
	if connID > 0 {
		table, ok := providerTable[provider]
		if !ok {
			return "", "", fmt.Errorf("unsupported provider: %s", provider)
		}
		return lookupConnection(table, connID)
	}
	if legacyBucket != "" && legacyCreds != "" {
		return legacyBucket, legacyCreds, nil
	}
	return "", "", fmt.Errorf("connection_id or bucket+credentials required")
}

// ── Streaming download helpers ────────────────────────────────────────────────

// streamDownloadObject returns a streaming reader instead of buffering the entire file in memory.
func streamDownloadObject(ctx context.Context, provider, bucket, credentials, object string) (io.ReadCloser, string, error) {
	switch provider {
	case "aws":
		return streamDownloadS3(ctx, bucket, credentials, object, awsS3Client)
	case "alibaba":
		return streamDownloadS3(ctx, bucket, credentials, object, ossS3Client)
	case "huawei":
		return streamDownloadS3(ctx, bucket, credentials, object, obsS3Client)
	case "gcp":
		return streamDownloadGCS(ctx, bucket, credentials, object)
	case "azure":
		return streamDownloadAzureBlob(ctx, bucket, credentials, object)
	case "gdrive":
		return streamDownloadGDriveFile(ctx, bucket, credentials, object)
	default:
		return nil, "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

func streamDownloadS3(ctx context.Context, bucket, credentialsJSON, object string, clientFn func(context.Context, map[string]string) (*s3.Client, error)) (io.ReadCloser, string, error) {
	creds, err := awsCredsFromJSON(credentialsJSON)
	if err != nil {
		return nil, "", err
	}
	client, err := clientFn(ctx, creds)
	if err != nil {
		return nil, "", err
	}
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: awslib.String(bucket),
		Key:    awslib.String(object),
	})
	if err != nil {
		return nil, "", err
	}
	ct := "application/octet-stream"
	if out.ContentType != nil && *out.ContentType != "" {
		ct = *out.ContentType
	}
	return out.Body, ct, nil
}

func streamDownloadGCS(ctx context.Context, bucket, credentials, object string) (io.ReadCloser, string, error) {
	client, err := gcpClient(ctx, credentials)
	if err != nil {
		return nil, "", err
	}
	obj := client.Bucket(bucket).Object(object)
	ct := "application/octet-stream"
	if attrs, attrErr := obj.Attrs(ctx); attrErr == nil && attrs.ContentType != "" {
		ct = attrs.ContentType
	}
	rc, err := obj.NewReader(ctx)
	if err != nil {
		client.Close()
		return nil, "", err
	}
	return &gcsReadCloser{rc: rc, client: client}, ct, nil
}

type gcsReadCloser struct {
	rc     io.ReadCloser
	client *storage.Client
}

func (g *gcsReadCloser) Read(p []byte) (int, error) { return g.rc.Read(p) }
func (g *gcsReadCloser) Close() error {
	err1 := g.rc.Close()
	err2 := g.client.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func streamDownloadAzureBlob(ctx context.Context, container, credentials, object string) (io.ReadCloser, string, error) {
	accountName, accountKey, err := azureCredsFromJSON(credentials)
	if err != nil {
		return nil, "", err
	}
	containerClient, _, err := azureContainerClient(accountName, accountKey, container)
	if err != nil {
		return nil, "", err
	}
	blobClient := containerClient.NewBlobClient(object)
	ct := "application/octet-stream"
	if props, propErr := blobClient.GetProperties(ctx, nil); propErr == nil && props.ContentType != nil && *props.ContentType != "" {
		ct = *props.ContentType
	}
	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	return resp.Body, ct, nil
}

func streamDownloadGDriveFile(ctx context.Context, folderID, credentials, object string) (io.ReadCloser, string, error) {
	srv, err := gdriveService(ctx, credentials)
	if err != nil {
		return nil, "", err
	}
	fileID := extractFileID(object)
	resp, err := srv.Files.Get(fileID).Download()
	if err != nil {
		return nil, "", err
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	return resp.Body, ct, nil
}

// ── Streaming upload helpers ──────────────────────────────────────────────────

// streamUploadObject uploads from a reader instead of a []byte buffer.
func streamUploadObject(ctx context.Context, provider, bucket, credentials, destKey string, body io.Reader, size int64, contentType string) error {
	switch provider {
	case "aws":
		return streamUploadS3(ctx, bucket, credentials, destKey, body, size, contentType, awsS3Client)
	case "alibaba":
		return streamUploadS3(ctx, bucket, credentials, destKey, body, size, contentType, ossS3Client)
	case "huawei":
		return streamUploadS3(ctx, bucket, credentials, destKey, body, size, contentType, obsS3Client)
	case "gcp":
		return streamUploadGCS(ctx, bucket, credentials, destKey, body, contentType)
	case "azure":
		data, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		return uploadAzureBlob(ctx, bucket, credentials, destKey, data, contentType)
	case "gdrive":
		data, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		return uploadGDriveFile(ctx, bucket, credentials, destKey, data, contentType)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

func streamUploadS3(ctx context.Context, bucket, credentialsJSON, key string, body io.Reader, size int64, contentType string, clientFn func(context.Context, map[string]string) (*s3.Client, error)) error {
	creds, err := awsCredsFromJSON(credentialsJSON)
	if err != nil {
		return err
	}
	client, err := clientFn(ctx, creds)
	if err != nil {
		return err
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("reading source: %w", err)
	}

	// Upload via a presigned PUT URL so that no SDK middleware runs on the
	// actual HTTP request. Newer AWS SDK versions add x-amz-checksum-* and
	// aws-chunked Transfer-Encoding headers that some S3 regions (e.g.
	// ap-southeast-3) reject with 501 NotImplemented. Presigned URLs use
	// UNSIGNED-PAYLOAD by design, so none of those headers are ever added.
	presigner := s3.NewPresignClient(client)
	presigned, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: awslib.String(bucket),
		Key:    awslib.String(key),
	}, func(o *s3.PresignOptions) {
		o.Expires = 15 * time.Minute
	})
	if err != nil {
		return fmt.Errorf("presign: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, presigned.URL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	httpReq.ContentLength = int64(len(data))
	httpReq.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http put: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("S3 upload returned HTTP %d: %s", resp.StatusCode, string(errBody))
	}
	return nil
}

func streamUploadGCS(ctx context.Context, bucket, credentials, key string, body io.Reader, contentType string) error {
	client, err := gcpClient(ctx, credentials)
	if err != nil {
		return err
	}
	defer client.Close()
	wc := client.Bucket(bucket).Object(key).NewWriter(ctx)
	wc.ContentType = contentType
	if _, err := io.Copy(wc, body); err != nil {
		wc.Close()
		return err
	}
	return wc.Close()
}

// ── Buffered download helpers (kept for backward compat) ──────────────────────

func downloadObjectData(ctx context.Context, provider, bucket, credentials, object string) ([]byte, string, error) {
	switch provider {
	case "aws":
		return downloadS3(ctx, bucket, credentials, object, awsS3Client)
	case "alibaba":
		return downloadS3(ctx, bucket, credentials, object, ossS3Client)
	case "huawei":
		return downloadS3(ctx, bucket, credentials, object, obsS3Client)
	case "gcp":
		return downloadGCS(ctx, bucket, credentials, object)
	case "azure":
		return downloadAzureBlob(ctx, bucket, credentials, object)
	case "gdrive":
		return downloadGDriveFile(ctx, bucket, credentials, object)
	default:
		return nil, "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

func downloadS3(ctx context.Context, bucket, credentialsJSON, object string, clientFn func(context.Context, map[string]string) (*s3.Client, error)) ([]byte, string, error) {
	creds, err := awsCredsFromJSON(credentialsJSON)
	if err != nil {
		return nil, "", err
	}
	client, err := clientFn(ctx, creds)
	if err != nil {
		return nil, "", err
	}
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: awslib.String(bucket),
		Key:    awslib.String(object),
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

func downloadGCS(ctx context.Context, bucket, credentials, object string) ([]byte, string, error) {
	client, err := gcpClient(ctx, credentials)
	if err != nil {
		return nil, "", err
	}
	defer client.Close()

	obj := client.Bucket(bucket).Object(object)
	ct := "application/octet-stream"
	if attrs, attrErr := obj.Attrs(ctx); attrErr == nil && attrs.ContentType != "" {
		ct = attrs.ContentType
	}
	rc, err := obj.NewReader(ctx)
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	return data, ct, err
}

func downloadAzureBlob(ctx context.Context, container, credentials, object string) ([]byte, string, error) {
	accountName, accountKey, err := azureCredsFromJSON(credentials)
	if err != nil {
		return nil, "", err
	}
	containerClient, _, err := azureContainerClient(accountName, accountKey, container)
	if err != nil {
		return nil, "", err
	}
	blobClient := containerClient.NewBlobClient(object)

	ct := "application/octet-stream"
	if props, propErr := blobClient.GetProperties(ctx, nil); propErr == nil && props.ContentType != nil && *props.ContentType != "" {
		ct = *props.ContentType
	}

	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return data, ct, err
}

// ── Buffered upload helpers (kept for backward compat) ────────────────────────

func uploadObjectData(ctx context.Context, provider, bucket, credentials, destKey string, data []byte, contentType string) error {
	switch provider {
	case "aws":
		return uploadS3(ctx, bucket, credentials, destKey, data, contentType, awsS3Client)
	case "alibaba":
		return uploadS3(ctx, bucket, credentials, destKey, data, contentType, ossS3Client)
	case "huawei":
		return uploadS3(ctx, bucket, credentials, destKey, data, contentType, obsS3Client)
	case "gcp":
		return uploadGCS(ctx, bucket, credentials, destKey, data, contentType)
	case "azure":
		return uploadAzureBlob(ctx, bucket, credentials, destKey, data, contentType)
	case "gdrive":
		return uploadGDriveFile(ctx, bucket, credentials, destKey, data, contentType)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

// ── Google Drive transfer helpers ─────────────────────────────────────────────

func downloadGDriveFile(ctx context.Context, folderID, credentials, object string) ([]byte, string, error) {
	srv, err := gdriveService(ctx, credentials)
	if err != nil {
		return nil, "", err
	}
	fileID := extractFileID(object)
	resp, err := srv.Files.Get(fileID).Download()
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	return data, ct, nil
}

func uploadGDriveFile(ctx context.Context, folderID, credentials, destKey string, data []byte, contentType string) error {
	srv, err := gdriveService(ctx, credentials)
	if err != nil {
		return err
	}
	filename := path.Base(destKey)
	driveFile := &driveapi.File{
		Name:     filename,
		Parents:  []string{folderID},
		MimeType: contentType,
	}
	_, err = srv.Files.Create(driveFile).Media(bytes.NewReader(data)).Context(ctx).Do()
	return err
}

func uploadS3(ctx context.Context, bucket, credentialsJSON, key string, data []byte, contentType string, clientFn func(context.Context, map[string]string) (*s3.Client, error)) error {
	creds, err := awsCredsFromJSON(credentialsJSON)
	if err != nil {
		return err
	}
	client, err := clientFn(ctx, creds)
	if err != nil {
		return err
	}
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        awslib.String(bucket),
		Key:           awslib.String(key),
		Body:          bytes.NewReader(data),
		ContentLength: awslib.Int64(int64(len(data))),
		ContentType:   awslib.String(contentType),
	})
	return err
}

func uploadGCS(ctx context.Context, bucket, credentials, key string, data []byte, contentType string) error {
	client, err := gcpClient(ctx, credentials)
	if err != nil {
		return err
	}
	defer client.Close()

	wc := client.Bucket(bucket).Object(key).NewWriter(ctx)
	wc.ContentType = contentType
	if _, err := io.Copy(wc, bytes.NewReader(data)); err != nil {
		wc.Close()
		return err
	}
	return wc.Close()
}

func uploadAzureBlob(ctx context.Context, container, credentials, key string, data []byte, contentType string) error {
	accountName, accountKey, err := azureCredsFromJSON(credentials)
	if err != nil {
		return err
	}
	containerClient, _, err := azureContainerClient(accountName, accountKey, container)
	if err != nil {
		return err
	}
	blobClient := containerClient.NewBlockBlobClient(key)
	_, err = blobClient.UploadBuffer(ctx, data, &blockblob.UploadBufferOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: strPtr(contentType),
		},
	})
	return err
}

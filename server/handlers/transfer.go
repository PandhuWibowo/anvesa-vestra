package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	awslib "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	driveapi "google.golang.org/api/drive/v3"
)

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Resolve source credentials from connection ID
	srcBucket, srcCreds, err := resolveProviderCreds(req.SrcProvider, req.SrcConnectionID, req.SrcBucket, req.SrcCredentials)
	if err != nil {
		jsonError(w, "source connection error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Resolve destination credentials from connection ID
	dstBucket, dstCreds, err := resolveProviderCreds(req.DstProvider, req.DstConnectionID, req.DstBucket, req.DstCredentials)
	if err != nil {
		jsonError(w, "destination connection error: "+err.Error(), http.StatusBadRequest)
		return
	}

	data, contentType, err := downloadObjectData(ctx, req.SrcProvider, srcBucket, srcCreds, req.SrcObject)
	if err != nil {
		jsonError(w, "download failed", http.StatusInternalServerError)
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

	if err := uploadObjectData(ctx, req.DstProvider, dstBucket, dstCreds, destKey, data, contentType); err != nil {
		jsonError(w, "upload failed", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"destination": destKey})
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

// ── Internal download helpers ─────────────────────────────────────────────────

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

// ── Internal upload helpers ───────────────────────────────────────────────────

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

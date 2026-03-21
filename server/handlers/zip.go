package handlers

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	awslib "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"cloud.google.com/go/storage"
	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"google.golang.org/api/iterator"
)

const maxZipObjects = 10000

type zipRequest struct {
	Provider     string   `json:"provider"`
	ConnectionID int64    `json:"connection_id"`
	Prefix       string   `json:"prefix"`
	Objects      []string `json:"objects"`
	// Legacy fields (deprecated)
	Bucket      string `json:"bucket"`
	Credentials string `json:"credentials"`
}

var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func ZipObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r = limitBody(r, MaxBodySize)
	var req zipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	bucket, creds, err := resolveProviderCreds(req.Provider, req.ConnectionID, req.Bucket, req.Credentials)
	if err != nil {
		jsonError(w, "connection error: "+err.Error(), http.StatusBadRequest)
		return
	}

	keys, err := collectZipKeys(ctx, req.Provider, bucket, creds, req.Prefix, req.Objects)
	if err != nil {
		jsonError(w, "failed to list objects", http.StatusInternalServerError)
		return
	}
	if len(keys) == 0 {
		jsonError(w, "no objects found", http.StatusNotFound)
		return
	}
	if len(keys) > maxZipObjects {
		jsonError(w, fmt.Sprintf("too many objects (%d), max is %d", len(keys), maxZipObjects), http.StatusBadRequest)
		return
	}

	archiveName := strings.Trim(req.Prefix, "/")
	if idx := strings.LastIndex(archiveName, "/"); idx >= 0 {
		archiveName = archiveName[idx+1:]
	}
	if archiveName == "" {
		archiveName = bucket
	}
	archiveName = unsafeFilenameChars.ReplaceAllString(archiveName, "_")

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, archiveName))

	zw := zip.NewWriter(w)
	defer zw.Close()

	var skipped int
	for _, key := range keys {
		data, _, dlErr := downloadObjectData(ctx, req.Provider, bucket, creds, key)
		if dlErr != nil {
			log.Printf("zip: skipping %q: %v", key, dlErr)
			skipped++
			continue
		}

		entryName := strings.TrimPrefix(key, req.Prefix)
		entryName = strings.TrimPrefix(entryName, "/")
		if entryName == "" {
			entryName = path.Base(key)
		}

		fw, createErr := zw.Create(entryName)
		if createErr != nil {
			continue
		}
		fw.Write(data) //nolint:errcheck
	}

	if skipped > 0 {
		log.Printf("zip: %d files skipped due to download errors", skipped)
	}
}

func collectZipKeys(ctx context.Context, provider, bucket, creds, prefix string, objects []string) ([]string, error) {
	if len(objects) > 0 {
		return objects, nil
	}
	switch provider {
	case "aws", "alibaba", "huawei":
		return listS3ZipKeys(ctx, provider, bucket, creds, prefix)
	case "gcp":
		return listGCSZipKeys(ctx, bucket, creds, prefix)
	case "azure":
		return listAzureZipKeys(ctx, bucket, creds, prefix)
	}
	return nil, fmt.Errorf("unknown provider: %s", provider)
}

func listS3ZipKeys(ctx context.Context, provider, bucket, creds, prefix string) ([]string, error) {
	parsedCreds, err := awsCredsFromJSON(creds)
	if err != nil {
		return nil, err
	}
	var clientFn func(context.Context, map[string]string) (*s3.Client, error)
	switch provider {
	case "aws":
		clientFn = awsS3Client
	case "alibaba":
		clientFn = ossS3Client
	case "huawei":
		clientFn = obsS3Client
	}
	client, err := clientFn(ctx, parsedCreds)
	if err != nil {
		return nil, err
	}
	var keys []string
	var token *string
	for {
		out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            awslib.String(bucket),
			Prefix:            awslib.String(prefix),
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
		if len(keys) > maxZipObjects {
			break
		}
		if out.IsTruncated == nil || !*out.IsTruncated {
			break
		}
		token = out.NextContinuationToken
	}
	return keys, nil
}

func listGCSZipKeys(ctx context.Context, bucket, creds, prefix string) ([]string, error) {
	client, err := gcpClient(ctx, creds)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	it := client.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: prefix})
	var keys []string
	for {
		attrs, iterErr := it.Next()
		if iterErr == iterator.Done {
			break
		}
		if iterErr != nil {
			return nil, iterErr
		}
		keys = append(keys, attrs.Name)
		if len(keys) > maxZipObjects {
			break
		}
	}
	return keys, nil
}

func listAzureZipKeys(ctx context.Context, container, creds, prefix string) ([]string, error) {
	accountName, accountKey, err := azureCredsFromJSON(creds)
	if err != nil {
		return nil, err
	}
	containerClient, _, err := azureContainerClient(accountName, accountKey, container)
	if err != nil {
		return nil, err
	}
	pager := containerClient.NewListBlobsFlatPager(&azcontainer.ListBlobsFlatOptions{
		Prefix: strPtr(prefix),
	})
	var keys []string
	for pager.More() {
		page, pageErr := pager.NextPage(ctx)
		if pageErr != nil {
			return nil, pageErr
		}
		for _, blob := range page.Segment.BlobItems {
			if blob.Name != nil {
				keys = append(keys, *blob.Name)
			}
		}
		if len(keys) > maxZipObjects {
			break
		}
	}
	return keys, nil
}

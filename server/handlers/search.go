package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

const maxSearchResults = 100

type searchResult struct {
	ConnectionID   int64     `json:"connection_id"`
	ConnectionName string    `json:"connection_name"`
	Provider       string    `json:"provider"`
	Key            string    `json:"key"`
	Size           int64     `json:"size"`
	Updated        time.Time `json:"updated"`
}

// SearchObjects searches for objects by prefix across one or more connections.
func SearchObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query        string `json:"query"`
		Provider     string `json:"provider"`
		ConnectionID int64  `json:"connection_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := requireFields(map[string]string{"query": req.Query, "provider": req.Provider}); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	table, ok := providerTableName(req.Provider)
	if !ok {
		jsonError(w, fmt.Sprintf("unsupported provider: %s", req.Provider), http.StatusBadRequest)
		return
	}

	type conn struct {
		ID          int64
		Name        string
		Bucket      string
		Credentials string
	}

	var conns []conn
	if req.ConnectionID > 0 {
		row := appdb.DB.QueryRow(
			fmt.Sprintf("SELECT id, name, bucket, credentials FROM %s WHERE id = ?", table),
			req.ConnectionID,
		)
		var c conn
		if err := row.Scan(&c.ID, &c.Name, &c.Bucket, &c.Credentials); err != nil {
			jsonError(w, "connection not found", http.StatusNotFound)
			return
		}
		conns = append(conns, c)
	} else {
		rows, err := appdb.DB.Query(
			fmt.Sprintf("SELECT id, name, bucket, credentials FROM %s", table),
		)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var c conn
			if err := rows.Scan(&c.ID, &c.Name, &c.Bucket, &c.Credentials); err != nil {
				continue
			}
			conns = append(conns, c)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var results []searchResult
	for _, c := range conns {
		if len(results) >= maxSearchResults {
			break
		}
		remaining := maxSearchResults - len(results)

		switch req.Provider {
		case "aws", "alibaba", "huawei":
			found := searchS3(ctx, req.Provider, c.ID, c.Name, c.Bucket, c.Credentials, req.Query, remaining)
			results = append(results, found...)
		case "gcp":
			found := searchGCP(ctx, c.ID, c.Name, c.Bucket, c.Credentials, req.Query, remaining)
			results = append(results, found...)
		case "azure":
			found := searchAzure(ctx, c.ID, c.Name, c.Bucket, c.Credentials, req.Query, remaining)
			results = append(results, found...)
		}
	}

	if results == nil {
		results = []searchResult{}
	}
	jsonOK(w, map[string]any{"results": results})
}

func searchS3(ctx context.Context, provider string, connID int64, connName, bucket, credJSON, query string, limit int) []searchResult {
	sp := resolveS3Provider(provider)
	if sp == nil {
		return nil
	}
	creds, err := sp.CredsFunc(credJSON)
	if err != nil {
		return nil
	}
	client, err := sp.ClientFunc(ctx, creds)
	if err != nil {
		return nil
	}

	out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(query),
		MaxKeys: aws.Int32(int32(limit)),
	})
	if err != nil {
		return nil
	}

	var results []searchResult
	for _, obj := range out.Contents {
		var key string
		if obj.Key != nil {
			key = *obj.Key
		}
		var size int64
		if obj.Size != nil {
			size = *obj.Size
		}
		var updated time.Time
		if obj.LastModified != nil {
			updated = *obj.LastModified
		}
		results = append(results, searchResult{
			ConnectionID:   connID,
			ConnectionName: connName,
			Provider:       provider,
			Key:            key,
			Size:           size,
			Updated:        updated,
		})
	}
	return results
}

func searchGCP(ctx context.Context, connID int64, connName, bucket, credJSON, query string, limit int) []searchResult {
	client, err := gcpClient(ctx, credJSON)
	if err != nil {
		return nil
	}
	defer client.Close()

	it := client.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: query})
	var results []searchResult
	for len(results) < limit {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		results = append(results, searchResult{
			ConnectionID:   connID,
			ConnectionName: connName,
			Provider:       "gcp",
			Key:            attrs.Name,
			Size:           attrs.Size,
			Updated:        attrs.Updated,
		})
	}
	return results
}

func searchAzure(ctx context.Context, connID int64, connName, containerName, credJSON, query string, limit int) []searchResult {
	accountName, accountKey, err := azureCredsFromJSON(credJSON)
	if err != nil {
		return nil
	}

	containerClient, _, err := azureContainerClient(accountName, accountKey, containerName)
	if err != nil {
		return nil
	}

	pager := containerClient.NewListBlobsFlatPager(&azcontainer.ListBlobsFlatOptions{
		Prefix:     strPtr(query),
		MaxResults: i32Ptr(int32(limit)),
	})

	var results []searchResult
	for pager.More() && len(results) < limit {
		page, err := pager.NextPage(ctx)
		if err != nil {
			break
		}
		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			var size int64
			if blob.Properties != nil && blob.Properties.ContentLength != nil {
				size = *blob.Properties.ContentLength
			}
			var updated time.Time
			if blob.Properties != nil && blob.Properties.LastModified != nil {
				updated = *blob.Properties.LastModified
			}
			results = append(results, searchResult{
				ConnectionID:   connID,
				ConnectionName: connName,
				Provider:       "azure",
				Key:            *blob.Name,
				Size:           size,
				Updated:        updated,
			})
			if len(results) >= limit {
				break
			}
		}
	}
	return results
}

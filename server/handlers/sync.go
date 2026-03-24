package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	awslib "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

// ── Handlers ──────────────────────────────────────────────────────────────────

// SyncJobsRoute dispatches GET (list) and POST (create) for /api/sync.
func SyncJobsRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listSyncJobs(w, r)
	case http.MethodPost:
		createSyncJob(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func listSyncJobs(w http.ResponseWriter, r *http.Request) {
	rows, err := appdb.DB.Query(
		`SELECT id, name, src_connection_id, src_provider, dst_connection_id, dst_provider,
		        src_prefix, dst_prefix, schedule, last_run, next_run, status, created_at
		 FROM sync_jobs ORDER BY created_at DESC`,
	)
	if err != nil {
		jsonError(w, safeError(err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var jobs []map[string]any
	for rows.Next() {
		var (
			id, srcConnID, dstConnID                          int64
			name, srcProvider, dstProvider, srcPrefix          string
			dstPrefix, schedule, status, createdAt             string
			lastRun, nextRun                                   sql.NullString
		)
		if err := rows.Scan(&id, &name, &srcConnID, &srcProvider, &dstConnID, &dstProvider,
			&srcPrefix, &dstPrefix, &schedule, &lastRun, &nextRun, &status, &createdAt); err != nil {
			jsonError(w, safeError(err), http.StatusInternalServerError)
			return
		}
		job := map[string]any{
			"id":                id,
			"name":              name,
			"src_connection_id": srcConnID,
			"src_provider":      srcProvider,
			"dst_connection_id": dstConnID,
			"dst_provider":      dstProvider,
			"src_prefix":        srcPrefix,
			"dst_prefix":        dstPrefix,
			"schedule":          schedule,
			"status":            status,
			"created_at":        createdAt,
		}
		if lastRun.Valid {
			job["last_run"] = lastRun.String
		}
		if nextRun.Valid {
			job["next_run"] = nextRun.String
		}
		jobs = append(jobs, job)
	}

	if jobs == nil {
		jobs = []map[string]any{}
	}
	jsonOK(w, jobs)
}

type createSyncJobReq struct {
	Name            string `json:"name"`
	SrcConnectionID int64  `json:"src_connection_id"`
	SrcProvider     string `json:"src_provider"`
	DstConnectionID int64  `json:"dst_connection_id"`
	DstProvider     string `json:"dst_provider"`
	SrcPrefix       string `json:"src_prefix"`
	DstPrefix       string `json:"dst_prefix"`
	Schedule        string `json:"schedule"`
}

func createSyncJob(w http.ResponseWriter, r *http.Request) {
	r = limitBody(r, MaxBodySize)
	var req createSyncJobReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := requireFields(map[string]string{
		"name":         req.Name,
		"src_provider": req.SrcProvider,
		"dst_provider": req.DstProvider,
	}); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.SrcConnectionID == 0 {
		jsonError(w, "src_connection_id is required", http.StatusBadRequest)
		return
	}
	if req.DstConnectionID == 0 {
		jsonError(w, "dst_connection_id is required", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	createdAt := now.Format("2006-01-02 15:04:05")

	var nextRunVal sql.NullString
	if next := calculateNextRun(req.Schedule); !next.IsZero() {
		nextRunVal = sql.NullString{String: next.Format("2006-01-02 15:04:05"), Valid: true}
	}

	res, err := appdb.DB.Exec(
		`INSERT INTO sync_jobs (name, src_connection_id, src_provider, dst_connection_id, dst_provider,
		                        src_prefix, dst_prefix, schedule, next_run, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'idle', ?)`,
		req.Name, req.SrcConnectionID, req.SrcProvider, req.DstConnectionID, req.DstProvider,
		req.SrcPrefix, req.DstPrefix, req.Schedule, nextRunVal, createdAt,
	)
	if err != nil {
		jsonError(w, safeError(err), http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	result := map[string]any{
		"id":                id,
		"name":              req.Name,
		"src_connection_id": req.SrcConnectionID,
		"src_provider":      req.SrcProvider,
		"dst_connection_id": req.DstConnectionID,
		"dst_provider":      req.DstProvider,
		"src_prefix":        req.SrcPrefix,
		"dst_prefix":        req.DstPrefix,
		"schedule":          req.Schedule,
		"status":            "idle",
		"created_at":        createdAt,
	}
	if nextRunVal.Valid {
		result["next_run"] = nextRunVal.String
	}
	jsonOK(w, result)
}

// SyncJobByID handles GET/PUT/DELETE for /api/sync/{id}.
func SyncJobByID(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/run") {
		RunSyncJob(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		getSyncJob(w, r)
	case http.MethodPut:
		updateSyncJob(w, r)
	case http.MethodDelete:
		deleteSyncJob(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func getSyncJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, 3) // /api/sync/{id} → position 3
	if err != nil {
		jsonError(w, "invalid sync job id", http.StatusBadRequest)
		return
	}

	var (
		name, srcProvider, dstProvider, srcPrefix     string
		dstPrefix, schedule, status, createdAt         string
		srcConnID, dstConnID                           int64
		lastRun, nextRun                               sql.NullString
	)
	err = appdb.DB.QueryRow(
		`SELECT id, name, src_connection_id, src_provider, dst_connection_id, dst_provider,
		        src_prefix, dst_prefix, schedule, last_run, next_run, status, created_at
		 FROM sync_jobs WHERE id = ?`, id,
	).Scan(&id, &name, &srcConnID, &srcProvider, &dstConnID, &dstProvider,
		&srcPrefix, &dstPrefix, &schedule, &lastRun, &nextRun, &status, &createdAt)
	if err != nil {
		jsonError(w, "sync job not found", http.StatusNotFound)
		return
	}

	job := map[string]any{
		"id":                id,
		"name":              name,
		"src_connection_id": srcConnID,
		"src_provider":      srcProvider,
		"dst_connection_id": dstConnID,
		"dst_provider":      dstProvider,
		"src_prefix":        srcPrefix,
		"dst_prefix":        dstPrefix,
		"schedule":          schedule,
		"status":            status,
		"created_at":        createdAt,
	}
	if lastRun.Valid {
		job["last_run"] = lastRun.String
	}
	if nextRun.Valid {
		job["next_run"] = nextRun.String
	}
	jsonOK(w, job)
}

func updateSyncJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, 3)
	if err != nil {
		jsonError(w, "invalid sync job id", http.StatusBadRequest)
		return
	}

	r = limitBody(r, MaxBodySize)
	var req struct {
		Name      *string `json:"name"`
		Schedule  *string `json:"schedule"`
		SrcPrefix *string `json:"src_prefix"`
		DstPrefix *string `json:"dst_prefix"`
		Status    *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	var sets []string
	var args []any

	if req.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *req.Name)
	}
	if req.SrcPrefix != nil {
		sets = append(sets, "src_prefix = ?")
		args = append(args, *req.SrcPrefix)
	}
	if req.DstPrefix != nil {
		sets = append(sets, "dst_prefix = ?")
		args = append(args, *req.DstPrefix)
	}
	if req.Status != nil {
		sets = append(sets, "status = ?")
		args = append(args, *req.Status)
	}
	if req.Schedule != nil {
		sets = append(sets, "schedule = ?")
		args = append(args, *req.Schedule)
		if next := calculateNextRun(*req.Schedule); !next.IsZero() {
			sets = append(sets, "next_run = ?")
			args = append(args, next.Format("2006-01-02 15:04:05"))
		} else {
			sets = append(sets, "next_run = NULL")
		}
	}

	if len(sets) == 0 {
		getSyncJob(w, r)
		return
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE sync_jobs SET %s WHERE id = ?", strings.Join(sets, ", "))

	result, err := appdb.DB.Exec(query, args...)
	if err != nil {
		jsonError(w, safeError(err), http.StatusInternalServerError)
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		jsonError(w, "sync job not found", http.StatusNotFound)
		return
	}

	getSyncJob(w, r)
}

func deleteSyncJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, 3)
	if err != nil {
		jsonError(w, "invalid sync job id", http.StatusBadRequest)
		return
	}

	res, err := appdb.DB.Exec("DELETE FROM sync_jobs WHERE id = ?", id)
	if err != nil {
		jsonError(w, safeError(err), http.StatusInternalServerError)
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		jsonError(w, "sync job not found", http.StatusNotFound)
		return
	}

	jsonOK(w, map[string]string{"status": "deleted"})
}

// RunSyncJob handles POST /api/sync/{id}/run — manually triggers a sync job.
func RunSyncJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := parseID(r, 3) // /api/sync/{id}/run → position 3
	if err != nil {
		jsonError(w, "invalid sync job id", http.StatusBadRequest)
		return
	}

	var name string
	if err := appdb.DB.QueryRow("SELECT name FROM sync_jobs WHERE id = ?", id).Scan(&name); err != nil {
		jsonError(w, "sync job not found", http.StatusNotFound)
		return
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	appdb.DB.Exec("UPDATE sync_jobs SET status = 'running', last_run = ? WHERE id = ?", now, id)

	go executeSyncTransfer(id, name)

	jsonOK(w, map[string]any{"ok": true, "message": "sync job started"})
}

// ── Scheduler ─────────────────────────────────────────────────────────────────

// StartSyncScheduler polls every 60 seconds for due sync jobs and executes them.
func StartSyncScheduler() {
	log.Println("sync scheduler started")
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().UTC()
		nowStr := now.Format("2006-01-02 15:04:05")

		rows, err := appdb.DB.Query(
			`SELECT id, name, schedule FROM sync_jobs
			 WHERE next_run <= ? AND status = 'idle' AND schedule != ''`,
			nowStr,
		)
		if err != nil {
			log.Printf("sync scheduler: query error: %v", err)
			continue
		}

		for rows.Next() {
			var id int64
			var name, schedule string
			if err := rows.Scan(&id, &name, &schedule); err != nil {
				log.Printf("sync scheduler: scan error: %v", err)
				continue
			}

			nextRun := calculateNextRun(schedule)
			var nextRunVal sql.NullString
			if !nextRun.IsZero() {
				nextRunVal = sql.NullString{String: nextRun.Format("2006-01-02 15:04:05"), Valid: true}
			}

			appdb.DB.Exec(
				"UPDATE sync_jobs SET status = 'running', last_run = ?, next_run = ? WHERE id = ?",
				nowStr, nextRunVal, id,
			)
			log.Printf("sync scheduler: triggered job %d (%s)", id, name)

			go executeSyncTransfer(id, name)
		}
		rows.Close()
	}
}

func calculateNextRun(schedule string) time.Time {
	now := time.Now().UTC()
	switch schedule {
	case "hourly":
		return now.Add(1 * time.Hour)
	case "daily":
		return now.Add(24 * time.Hour)
	case "weekly":
		return now.Add(7 * 24 * time.Hour)
	default:
		return time.Time{}
	}
}

// ── Sync execution ────────────────────────────────────────────────────────────

func executeSyncTransfer(id int64, name string) {
	var srcConnID, dstConnID int64
	var srcProvider, dstProvider, srcPrefix, dstPrefix string
	err := appdb.DB.QueryRow(
		`SELECT src_connection_id, src_provider, dst_connection_id, dst_provider, src_prefix, dst_prefix
		 FROM sync_jobs WHERE id = ?`, id,
	).Scan(&srcConnID, &srcProvider, &dstConnID, &dstProvider, &srcPrefix, &dstPrefix)
	if err != nil {
		log.Printf("sync: failed to load job %d: %v", id, err)
		appdb.DB.Exec("UPDATE sync_jobs SET status = 'error' WHERE id = ?", id)
		return
	}

	srcTable, ok := providerTable[srcProvider]
	if !ok {
		log.Printf("sync: unsupported source provider %s for job %d", srcProvider, id)
		appdb.DB.Exec("UPDATE sync_jobs SET status = 'error' WHERE id = ?", id)
		return
	}
	srcBucket, srcCreds, err := lookupConnection(srcTable, srcConnID)
	if err != nil {
		log.Printf("sync: source connection error for job %d: %v", id, err)
		appdb.DB.Exec("UPDATE sync_jobs SET status = 'error' WHERE id = ?", id)
		return
	}

	dstTable, ok := providerTable[dstProvider]
	if !ok {
		log.Printf("sync: unsupported destination provider %s for job %d", dstProvider, id)
		appdb.DB.Exec("UPDATE sync_jobs SET status = 'error' WHERE id = ?", id)
		return
	}
	dstBucket, dstCreds, err := lookupConnection(dstTable, dstConnID)
	if err != nil {
		log.Printf("sync: destination connection error for job %d: %v", id, err)
		appdb.DB.Exec("UPDATE sync_jobs SET status = 'error' WHERE id = ?", id)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	keys, err := listProviderObjects(ctx, srcProvider, srcBucket, srcCreds, srcPrefix)
	if err != nil {
		log.Printf("sync: failed to list source objects for job %d: %v", id, err)
		appdb.DB.Exec("UPDATE sync_jobs SET status = 'error' WHERE id = ?", id)
		return
	}

	transferred := 0
	for _, key := range keys {
		rc, ct, err := streamDownloadObject(ctx, srcProvider, srcBucket, srcCreds, key)
		if err != nil {
			log.Printf("sync: job %d: failed to download %s: %v", id, key, err)
			continue
		}

		relKey := strings.TrimPrefix(key, srcPrefix)
		destKey := dstPrefix + relKey

		if err := streamUploadObject(ctx, dstProvider, dstBucket, dstCreds, destKey, rc, -1, ct); err != nil {
			rc.Close()
			log.Printf("sync: job %d: failed to upload %s: %v", id, destKey, err)
			continue
		}
		rc.Close()
		transferred++
	}

	log.Printf("sync: job %d (%s) completed, transferred %d objects", id, name, transferred)
	appdb.DB.Exec("UPDATE sync_jobs SET status = 'idle' WHERE id = ?", id)
}

// ── Object listing helpers ────────────────────────────────────────────────────

func listProviderObjects(ctx context.Context, provider, bucket, creds, prefix string) ([]string, error) {
	switch provider {
	case "aws":
		return listS3Keys(ctx, bucket, creds, prefix, awsS3Client)
	case "alibaba":
		return listS3Keys(ctx, bucket, creds, prefix, ossS3Client)
	case "huawei":
		return listS3Keys(ctx, bucket, creds, prefix, obsS3Client)
	case "gcp":
		return listGCSKeys(ctx, bucket, creds, prefix)
	case "azure":
		return listAzureKeys(ctx, bucket, creds, prefix)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func listS3Keys(ctx context.Context, bucket, credentialsJSON, prefix string, clientFn func(context.Context, map[string]string) (*s3.Client, error)) ([]string, error) {
	creds, err := awsCredsFromJSON(credentialsJSON)
	if err != nil {
		return nil, err
	}
	client, err := clientFn(ctx, creds)
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
		if out.IsTruncated == nil || !*out.IsTruncated {
			break
		}
		token = out.NextContinuationToken
	}
	return keys, nil
}

func listGCSKeys(ctx context.Context, bucket, credentials, prefix string) ([]string, error) {
	client, err := gcpClient(ctx, credentials)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: prefix})
	var keys []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		keys = append(keys, attrs.Name)
	}
	return keys, nil
}

func listAzureKeys(ctx context.Context, container, credentials, prefix string) ([]string, error) {
	accountName, accountKey, err := azureCredsFromJSON(credentials)
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
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name != nil {
				keys = append(keys, *item.Name)
			}
		}
	}
	return keys, nil
}

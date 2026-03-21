// Package handlers — background job queue.
//
// Start the worker from main.go after DB init:
//
//	go handlers.StartJobWorker()
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	awslib "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/PandhuWibowo/oss-portable/auth"
	appdb "github.com/PandhuWibowo/oss-portable/db"
	"github.com/PandhuWibowo/oss-portable/middleware"
)

type Job struct {
	ID        int64          `json:"id"`
	Type      string         `json:"type"`
	Status    string         `json:"status"`
	Payload   string         `json:"payload"`
	Result    sql.NullString `json:"-"`
	Error     sql.NullString `json:"-"`
	Progress  float64        `json:"progress"`
	UserID    sql.NullInt64  `json:"-"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

type jobJSON struct {
	ID        int64   `json:"id"`
	Type      string  `json:"type"`
	Status    string  `json:"status"`
	Payload   string  `json:"payload,omitempty"`
	Result    *string `json:"result,omitempty"`
	Error     *string `json:"error,omitempty"`
	Progress  float64 `json:"progress"`
	UserID    *int64  `json:"user_id,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func jobToJSON(j *Job, includePayload bool) jobJSON {
	out := jobJSON{
		ID:        j.ID,
		Type:      j.Type,
		Status:    j.Status,
		Progress:  j.Progress,
		CreatedAt: j.CreatedAt,
		UpdatedAt: j.UpdatedAt,
	}
	if includePayload {
		out.Payload = j.Payload
	}
	if j.Result.Valid {
		out.Result = &j.Result.String
	}
	if j.Error.Valid {
		out.Error = &j.Error.String
	}
	if j.UserID.Valid {
		out.UserID = &j.UserID.Int64
	}
	return out
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// JobsRoute dispatches /api/jobs based on HTTP method.
func JobsRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		CreateJob(w, r)
	case http.MethodGet:
		ListJobs(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

var supportedJobTypes = map[string]bool{
	"transfer":    true,
	"bulk_delete": true,
	"sync":        true,
}

// CreateJob handles POST /api/jobs.
func CreateJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if !supportedJobTypes[req.Type] {
		jsonError(w, "unsupported job type: "+req.Type, http.StatusBadRequest)
		return
	}
	if len(req.Payload) == 0 {
		jsonError(w, "payload is required", http.StatusBadRequest)
		return
	}

	var userID sql.NullInt64
	if claims, ok := r.Context().Value(middleware.ClaimsKey).(*auth.Claims); ok && claims != nil {
		userID = sql.NullInt64{Int64: claims.UserID, Valid: true}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := appdb.DB.Exec(
		`INSERT INTO jobs (type, status, payload, progress, user_id, created_at, updated_at)
		 VALUES (?, 'pending', ?, 0, ?, ?, ?)`,
		req.Type, string(req.Payload), userID, now, now,
	)
	if err != nil {
		jsonError(w, "failed to create job", http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	jsonOK(w, map[string]any{"id": id, "status": "pending"})
}

// ListJobs handles GET /api/jobs.
func ListJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := r.URL.Query().Get("status")
	limit := 50
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = v
	}
	if limit > 500 {
		limit = 500
	}

	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = appdb.DB.Query(
			`SELECT id, type, status, progress, error, created_at, updated_at
			 FROM jobs WHERE status = ? ORDER BY created_at DESC LIMIT ?`,
			status, limit,
		)
	} else {
		rows, err = appdb.DB.Query(
			`SELECT id, type, status, progress, error, created_at, updated_at
			 FROM jobs ORDER BY created_at DESC LIMIT ?`,
			limit,
		)
	}
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type listItem struct {
		ID        int64   `json:"id"`
		Type      string  `json:"type"`
		Status    string  `json:"status"`
		Progress  float64 `json:"progress"`
		Error     *string `json:"error,omitempty"`
		CreatedAt string  `json:"created_at"`
		UpdatedAt string  `json:"updated_at"`
	}

	items := []listItem{}
	for rows.Next() {
		var it listItem
		var errMsg sql.NullString
		if err := rows.Scan(&it.ID, &it.Type, &it.Status, &it.Progress, &errMsg, &it.CreatedAt, &it.UpdatedAt); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if errMsg.Valid {
			it.Error = &errMsg.String
		}
		items = append(items, it)
	}
	jsonOK(w, items)
}

// GetJob handles GET /api/jobs/{id}.
func GetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := parseID(r, 3) // /api/jobs/{id} → position 3
	if err != nil {
		jsonError(w, "invalid job id", http.StatusBadRequest)
		return
	}

	var j Job
	err = appdb.DB.QueryRow(
		`SELECT id, type, status, payload, result, error, progress, user_id, created_at, updated_at
		 FROM jobs WHERE id = ?`, id,
	).Scan(&j.ID, &j.Type, &j.Status, &j.Payload, &j.Result, &j.Error, &j.Progress, &j.UserID, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		jsonError(w, "job not found", http.StatusNotFound)
		return
	}

	jsonOK(w, jobToJSON(&j, true))
}

// ── Worker ────────────────────────────────────────────────────────────────────

const maxConcurrentJobs = 3

func getNextPendingJob() (*Job, error) {
	var j Job
	err := appdb.DB.QueryRow(
		`SELECT id, type, status, payload, result, error, progress, user_id, created_at, updated_at
		 FROM jobs WHERE status = 'pending' ORDER BY created_at ASC LIMIT 1`,
	).Scan(&j.ID, &j.Type, &j.Status, &j.Payload, &j.Result, &j.Error, &j.Progress, &j.UserID, &j.CreatedAt, &j.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func getPendingJobs(limit int) ([]*Job, error) {
	rows, err := appdb.DB.Query(
		`SELECT id, type, status, payload, result, error, progress, user_id, created_at, updated_at
		 FROM jobs WHERE status = 'pending' ORDER BY created_at ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.Type, &j.Status, &j.Payload, &j.Result, &j.Error, &j.Progress, &j.UserID, &j.CreatedAt, &j.UpdatedAt); err != nil {
			continue
		}
		jobs = append(jobs, &j)
	}
	return jobs, nil
}

func updateJobStatus(id int64, status string, progress float64, result, errMsg string) {
	now := time.Now().UTC().Format(time.RFC3339)
	var resVal, errVal sql.NullString
	if result != "" {
		resVal = sql.NullString{String: result, Valid: true}
	}
	if errMsg != "" {
		errVal = sql.NullString{String: errMsg, Valid: true}
	}
	appdb.DB.Exec(
		`UPDATE jobs SET status = ?, progress = ?, result = ?, error = ?, updated_at = ? WHERE id = ?`,
		status, progress, resVal, errVal, now, id,
	)
}

// StartJobWorker polls for pending jobs every 5 seconds and processes them concurrently.
func StartJobWorker() {
	log.Println("job worker started")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	sem := make(chan struct{}, maxConcurrentJobs)

	for range ticker.C {
		jobs, err := getPendingJobs(maxConcurrentJobs)
		if err != nil {
			log.Printf("job worker: poll error: %v", err)
			continue
		}
		for _, job := range jobs {
			sem <- struct{}{}
			go func(j *Job) {
				defer func() { <-sem }()
				processJob(j)
			}(job)
		}
	}
}

func processJob(job *Job) {
	log.Printf("job worker: processing job %d (type=%s)", job.ID, job.Type)
	updateJobStatus(job.ID, "running", 0, "", "")

	switch job.Type {
	case "transfer":
		executeTransferJob(job)
	case "bulk_delete":
		executeBulkDeleteJob(job)
	case "sync":
		executeSyncJobType(job)
	default:
		updateJobStatus(job.ID, "failed", 0, "", "unknown job type: "+job.Type)
	}
}

// ── Transfer job executor ─────────────────────────────────────────────────────

type transferPayload struct {
	SrcProvider     string `json:"src_provider"`
	SrcConnectionID int64  `json:"src_connection_id"`
	SrcObject       string `json:"src_object"`
	DstProvider     string `json:"dst_provider"`
	DstConnectionID int64  `json:"dst_connection_id"`
	DstPrefix       string `json:"dst_prefix"`
}

func executeTransferJob(job *Job) {
	var p transferPayload
	if err := json.Unmarshal([]byte(job.Payload), &p); err != nil {
		updateJobStatus(job.ID, "failed", 0, "", "invalid payload: "+err.Error())
		return
	}

	if p.SrcProvider == "" || p.DstProvider == "" || p.SrcObject == "" || p.SrcConnectionID == 0 || p.DstConnectionID == 0 {
		updateJobStatus(job.ID, "failed", 0, "", "missing required fields in payload")
		return
	}

	updateJobStatus(job.ID, "running", 0.1, "", "")

	srcTable, ok := providerTable[p.SrcProvider]
	if !ok {
		updateJobStatus(job.ID, "failed", 0, "", "unsupported source provider: "+p.SrcProvider)
		return
	}
	srcBucket, srcCreds, err := lookupConnection(srcTable, p.SrcConnectionID)
	if err != nil {
		updateJobStatus(job.ID, "failed", 0, "", "source connection error: "+err.Error())
		return
	}

	updateJobStatus(job.ID, "running", 0.2, "", "")

	dstTable, ok := providerTable[p.DstProvider]
	if !ok {
		updateJobStatus(job.ID, "failed", 0, "", "unsupported destination provider: "+p.DstProvider)
		return
	}
	dstBucket, dstCreds, err := lookupConnection(dstTable, p.DstConnectionID)
	if err != nil {
		updateJobStatus(job.ID, "failed", 0, "", "destination connection error: "+err.Error())
		return
	}

	updateJobStatus(job.ID, "running", 0.3, "", "")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	data, contentType, err := downloadObjectData(ctx, p.SrcProvider, srcBucket, srcCreds, p.SrcObject)
	if err != nil {
		updateJobStatus(job.ID, "failed", 0.3, "", "download failed: "+err.Error())
		return
	}

	updateJobStatus(job.ID, "running", 0.6, "", "")

	filename := path.Base(p.SrcObject)
	prefix := strings.TrimSuffix(p.DstPrefix, "/")
	var destKey string
	if prefix == "" {
		destKey = filename
	} else {
		destKey = prefix + "/" + filename
	}

	if err := uploadObjectData(ctx, p.DstProvider, dstBucket, dstCreds, destKey, data, contentType); err != nil {
		updateJobStatus(job.ID, "failed", 0.6, "", "upload failed: "+err.Error())
		return
	}

	resultJSON, _ := json.Marshal(map[string]string{"destination": destKey})
	updateJobStatus(job.ID, "completed", 1.0, string(resultJSON), "")
	log.Printf("job worker: job %d completed (destination=%s)", job.ID, destKey)
}

// ── Bulk delete job executor ──────────────────────────────────────────────────

type bulkDeletePayload struct {
	Provider     string   `json:"provider"`
	ConnectionID int64    `json:"connection_id"`
	Objects      []string `json:"objects"`
}

func executeBulkDeleteJob(job *Job) {
	var p bulkDeletePayload
	if err := json.Unmarshal([]byte(job.Payload), &p); err != nil {
		updateJobStatus(job.ID, "failed", 0, "", "invalid payload: "+err.Error())
		return
	}

	table, ok := providerTable[p.Provider]
	if !ok {
		updateJobStatus(job.ID, "failed", 0, "", "unsupported provider: "+p.Provider)
		return
	}
	bucket, creds, err := lookupConnection(table, p.ConnectionID)
	if err != nil {
		updateJobStatus(job.ID, "failed", 0, "", "connection error: "+err.Error())
		return
	}

	ctx := context.Background()
	deleted := 0
	for i, obj := range p.Objects {
		progress := float64(i) / float64(len(p.Objects))
		updateJobStatus(job.ID, "running", progress, "", "")

		if err := deleteProviderObject(ctx, p.Provider, bucket, creds, obj); err != nil {
			log.Printf("bulk_delete: failed to delete %s: %v", obj, err)
			continue
		}
		deleted++
	}

	resultJSON, _ := json.Marshal(map[string]int{"deleted": deleted, "total": len(p.Objects)})
	updateJobStatus(job.ID, "completed", 1.0, string(resultJSON), "")
}

func deleteProviderObject(ctx context.Context, provider, bucket, creds, object string) error {
	switch provider {
	case "aws":
		return deleteS3Object(ctx, bucket, creds, object, awsS3Client)
	case "alibaba":
		return deleteS3Object(ctx, bucket, creds, object, ossS3Client)
	case "huawei":
		return deleteS3Object(ctx, bucket, creds, object, obsS3Client)
	case "gcp":
		return deleteGCSObject(ctx, bucket, creds, object)
	case "azure":
		return deleteAzureBlobObject(ctx, bucket, creds, object)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

func deleteS3Object(ctx context.Context, bucket, credentialsJSON, object string, clientFn func(context.Context, map[string]string) (*s3.Client, error)) error {
	creds, err := awsCredsFromJSON(credentialsJSON)
	if err != nil {
		return err
	}
	client, err := clientFn(ctx, creds)
	if err != nil {
		return err
	}
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: awslib.String(bucket),
		Key:    awslib.String(object),
	})
	return err
}

func deleteGCSObject(ctx context.Context, bucket, credentials, object string) error {
	client, err := gcpClient(ctx, credentials)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Bucket(bucket).Object(object).Delete(ctx)
}

func deleteAzureBlobObject(ctx context.Context, container, credentials, object string) error {
	accountName, accountKey, err := azureCredsFromJSON(credentials)
	if err != nil {
		return err
	}
	containerClient, _, err := azureContainerClient(accountName, accountKey, container)
	if err != nil {
		return err
	}
	blobClient := containerClient.NewBlobClient(object)
	_, err = blobClient.Delete(ctx, nil)
	return err
}

// ── Sync job executor ─────────────────────────────────────────────────────────

func executeSyncJobType(job *Job) {
	var p transferPayload
	if err := json.Unmarshal([]byte(job.Payload), &p); err != nil {
		updateJobStatus(job.ID, "failed", 0, "", "invalid payload: "+err.Error())
		return
	}
	executeTransferJob(job)
}

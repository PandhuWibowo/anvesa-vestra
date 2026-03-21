package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

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

	// Verify the job exists.
	var exists int
	if err := appdb.DB.QueryRow("SELECT 1 FROM sync_jobs WHERE id = ?", id).Scan(&exists); err != nil {
		jsonError(w, "sync job not found", http.StatusNotFound)
		return
	}

	if req.Name != nil {
		appdb.DB.Exec("UPDATE sync_jobs SET name = ? WHERE id = ?", *req.Name, id)
	}
	if req.SrcPrefix != nil {
		appdb.DB.Exec("UPDATE sync_jobs SET src_prefix = ? WHERE id = ?", *req.SrcPrefix, id)
	}
	if req.DstPrefix != nil {
		appdb.DB.Exec("UPDATE sync_jobs SET dst_prefix = ? WHERE id = ?", *req.DstPrefix, id)
	}
	if req.Status != nil {
		appdb.DB.Exec("UPDATE sync_jobs SET status = ? WHERE id = ?", *req.Status, id)
	}
	if req.Schedule != nil {
		var nextRunVal sql.NullString
		if next := calculateNextRun(*req.Schedule); !next.IsZero() {
			nextRunVal = sql.NullString{String: next.Format("2006-01-02 15:04:05"), Valid: true}
		}
		appdb.DB.Exec("UPDATE sync_jobs SET schedule = ?, next_run = ? WHERE id = ?", *req.Schedule, nextRunVal, id)
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

	var exists int
	if err := appdb.DB.QueryRow("SELECT 1 FROM sync_jobs WHERE id = ?", id).Scan(&exists); err != nil {
		jsonError(w, "sync job not found", http.StatusNotFound)
		return
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	appdb.DB.Exec("UPDATE sync_jobs SET status = 'running', last_run = ? WHERE id = ?", now, id)

	jsonOK(w, map[string]any{"ok": true, "message": "sync job started"})
}

// ── Scheduler ─────────────────────────────────────────────────────────────────

// StartSyncScheduler polls every 60 seconds for due sync jobs and marks them as running.
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

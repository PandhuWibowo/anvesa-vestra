package handlers

import (
	"net/http"
	"strconv"
	"time"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

// LogAudit inserts a record into the audit_log table.
func LogAudit(userID int64, action, provider, bucket, object, details, ip string) {
	now := time.Now().UTC().Format(time.RFC3339)
	appdb.DB.Exec(
		`INSERT INTO audit_log (user_id, action, provider, bucket, object, details, ip, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, action, provider, bucket, object, details, ip, now,
	)
}

// ListAuditLog returns recent audit log entries with optional limit and offset.
func ListAuditLog(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 100
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 1000 {
		limit = 1000
	}

	offset := 0
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	rows, err := appdb.DB.Query(
		`SELECT id, user_id, action, provider, bucket, object, details, ip, created_at
		 FROM audit_log ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type entry struct {
		ID        int64   `json:"id"`
		UserID    *int64  `json:"user_id"`
		Action    string  `json:"action"`
		Provider  *string `json:"provider"`
		Bucket    *string `json:"bucket"`
		Object    *string `json:"object"`
		Details   *string `json:"details"`
		IP        *string `json:"ip"`
		CreatedAt string  `json:"created_at"`
	}

	entries := []entry{}
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Action, &e.Provider, &e.Bucket, &e.Object, &e.Details, &e.IP, &e.CreatedAt); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		entries = append(entries, e)
	}
	jsonOK(w, entries)
}

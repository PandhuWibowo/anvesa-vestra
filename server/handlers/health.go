package handlers

import (
	"net/http"
	"time"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

var startTime = time.Now()

// HealthCheck handles GET /api/health. Returns status, timestamp, db_status, and uptime.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dbStatus := "ok"
	if err := appdb.DB.Ping(); err != nil {
		dbStatus = "error"
	}

	jsonOK(w, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"db_status": dbStatus,
		"uptime":    time.Since(startTime).Seconds(),
	})
}

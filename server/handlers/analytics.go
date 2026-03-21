package handlers

import (
	"fmt"
	"net/http"
	"time"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

// AnalyticsSummary returns an aggregate overview of connections, recent
// activity, background jobs, and shared links.
func AnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	connections := countConnections()
	activity := countActivity24h()
	jobs := countJobs()
	shared := countSharedLinks()
	connDetails := connectionDetails()

	jsonOK(w, map[string]any{
		"connections":        connections,
		"activity_24h":       activity,
		"jobs":               jobs,
		"shared_links":       shared,
		"connection_details": connDetails,
	})
}

func countConnections() map[string]any {
	counts := map[string]any{}
	var total int64

	for provider, table := range providerTable {
		var n int64
		if err := appdb.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			n = 0
		}
		counts[provider] = n
		total += n
	}

	counts["total"] = total
	return counts
}

func countActivity24h() map[string]int64 {
	cutoff := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)

	actions := map[string]string{
		"uploads":   "upload",
		"downloads": "download",
		"deletes":   "delete",
		"transfers": "transfer",
	}

	result := make(map[string]int64, len(actions))
	for key, action := range actions {
		var n int64
		if err := appdb.DB.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE action = ? AND created_at >= ?",
			action, cutoff,
		).Scan(&n); err != nil {
			n = 0
		}
		result[key] = n
	}
	return result
}

func countJobs() map[string]int64 {
	statuses := []string{"pending", "running", "completed", "failed"}
	result := make(map[string]int64, len(statuses))

	for _, s := range statuses {
		var n int64
		if err := appdb.DB.QueryRow(
			"SELECT COUNT(*) FROM jobs WHERE status = ?", s,
		).Scan(&n); err != nil {
			n = 0
		}
		result[s] = n
	}
	return result
}

func countSharedLinks() map[string]any {
	var active int64
	if err := appdb.DB.QueryRow(
		"SELECT COUNT(*) FROM shared_links WHERE expires_at IS NULL OR expires_at > ?",
		time.Now().UTC().Format(time.RFC3339),
	).Scan(&active); err != nil {
		active = 0
	}

	var totalDownloads int64
	if err := appdb.DB.QueryRow(
		"SELECT COALESCE(SUM(download_count), 0) FROM shared_links",
	).Scan(&totalDownloads); err != nil {
		totalDownloads = 0
	}

	return map[string]any{
		"active":          active,
		"total_downloads": totalDownloads,
	}
}

func connectionDetails() []map[string]any {
	var details []map[string]any
	for provider, table := range providerTable {
		rows, err := appdb.DB.Query(fmt.Sprintf("SELECT id, name, bucket FROM %s", table))
		if err != nil {
			continue
		}
		for rows.Next() {
			var id int64
			var name, bucket string
			if err := rows.Scan(&id, &name, &bucket); err != nil {
				continue
			}
			details = append(details, map[string]any{
				"id":       id,
				"provider": provider,
				"name":     name,
				"bucket":   bucket,
			})
		}
		rows.Close()
	}
	if details == nil {
		details = []map[string]any{}
	}
	return details
}

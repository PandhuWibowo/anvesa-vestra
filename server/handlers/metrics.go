package handlers

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

var (
	metricsStartTime = time.Now()

	httpRequestsTotal atomic.Int64
	uploadsTotal      atomic.Int64
	downloadsTotal    atomic.Int64
	deletesTotal      atomic.Int64
)

// RecordRequest increments the total HTTP request counter.
func RecordRequest(method, path string, status int) {
	httpRequestsTotal.Add(1)
}

func RecordUpload()   { uploadsTotal.Add(1) }
func RecordDownload() { downloadsTotal.Add(1) }
func RecordDelete()   { deletesTotal.Add(1) }

// MetricsHandler serves Prometheus text exposition format at GET /api/metrics.
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Counters from atomics
	fmt.Fprintf(w, "# HELP anveesa_http_requests_total Total HTTP requests served.\n")
	fmt.Fprintf(w, "# TYPE anveesa_http_requests_total counter\n")
	fmt.Fprintf(w, "anveesa_http_requests_total %d\n\n", httpRequestsTotal.Load())

	fmt.Fprintf(w, "# HELP anveesa_uploads_total Total file uploads.\n")
	fmt.Fprintf(w, "# TYPE anveesa_uploads_total counter\n")
	fmt.Fprintf(w, "anveesa_uploads_total %d\n\n", uploadsTotal.Load())

	fmt.Fprintf(w, "# HELP anveesa_downloads_total Total file downloads.\n")
	fmt.Fprintf(w, "# TYPE anveesa_downloads_total counter\n")
	fmt.Fprintf(w, "anveesa_downloads_total %d\n\n", downloadsTotal.Load())

	fmt.Fprintf(w, "# HELP anveesa_deletes_total Total file deletes.\n")
	fmt.Fprintf(w, "# TYPE anveesa_deletes_total counter\n")
	fmt.Fprintf(w, "anveesa_deletes_total %d\n\n", deletesTotal.Load())

	// Gauges queried live from DB
	fmt.Fprintf(w, "# HELP anveesa_connections_total Number of configured provider connections.\n")
	fmt.Fprintf(w, "# TYPE anveesa_connections_total gauge\n")
	connectionTables := []string{
		"gcp_connections", "aws_connections", "huawei_connections",
		"alibaba_connections", "azure_connections", "gdrive_connections",
		"b2_connections", "do_connections",
	}
	for _, table := range connectionTables {
		var count int64
		if err := appdb.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err != nil {
			count = 0
		}
		fmt.Fprintf(w, "anveesa_connections_total{provider=%q} %d\n", table, count)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "# HELP anveesa_jobs_total Number of jobs by status.\n")
	fmt.Fprintf(w, "# TYPE anveesa_jobs_total gauge\n")
	jobRows, err := appdb.DB.Query("SELECT status, COUNT(*) FROM jobs GROUP BY status")
	if err == nil {
		defer jobRows.Close()
		for jobRows.Next() {
			var status string
			var count int64
			if err := jobRows.Scan(&status, &count); err == nil {
				fmt.Fprintf(w, "anveesa_jobs_total{status=%q} %d\n", status, count)
			}
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "# HELP anveesa_shared_links_active Number of active shared links.\n")
	fmt.Fprintf(w, "# TYPE anveesa_shared_links_active gauge\n")
	var sharedCount int64
	if err := appdb.DB.QueryRow("SELECT COUNT(*) FROM shared_links").Scan(&sharedCount); err != nil {
		sharedCount = 0
	}
	fmt.Fprintf(w, "anveesa_shared_links_active %d\n\n", sharedCount)

	fmt.Fprintf(w, "# HELP anveesa_uptime_seconds Seconds since server start.\n")
	fmt.Fprintf(w, "# TYPE anveesa_uptime_seconds gauge\n")
	fmt.Fprintf(w, "anveesa_uptime_seconds %.2f\n", time.Since(metricsStartTime).Seconds())
}

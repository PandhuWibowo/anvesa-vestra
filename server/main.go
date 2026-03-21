package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PandhuWibowo/oss-portable/config"
	appdb "github.com/PandhuWibowo/oss-portable/db"
	"github.com/PandhuWibowo/oss-portable/handlers"
	"github.com/PandhuWibowo/oss-portable/middleware"
)

func main() {
	cfg := config.Load()

	if err := appdb.Init(cfg.DBPath); err != nil {
		log.Fatalf("db init failed: %v", err)
	}

	go handlers.StartJobWorker()
	go handlers.StartSyncScheduler()

	cors := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.CORS(cfg.CORSOrigin, next)
	}
	rate := middleware.RateLimit(20, 60)
	auth := middleware.RequireAuth(cfg.JWTSecret)
	adminOnly := middleware.RequireRole("admin")

	protect := func(h http.HandlerFunc) http.HandlerFunc {
		wrapped := h
		if cfg.AuthEnabled {
			wrapped = auth(wrapped)
		}
		return cors(rate(wrapped))
	}

	admin := func(h http.HandlerFunc) http.HandlerFunc {
		wrapped := h
		if cfg.AuthEnabled {
			wrapped = auth(adminOnly(wrapped))
		}
		return cors(rate(wrapped))
	}

	public := func(h http.HandlerFunc) http.HandlerFunc {
		return cors(rate(h))
	}

	mux := http.NewServeMux()

	// ── Auth (always public) ─────────────────────────────────────
	mux.HandleFunc("/api/auth/register", public(handlers.RegisterHandler(cfg.JWTSecret, cfg.JWTExpiry)))
	mux.HandleFunc("/api/auth/login", public(handlers.LoginHandler(cfg.JWTSecret, cfg.JWTExpiry)))
	mux.HandleFunc("/api/auth/me", protect(handlers.MeHandler))
	mux.HandleFunc("/api/auth/setup-status", public(handlers.SetupStatusHandler(cfg.AuthEnabled)))

	// ── GCP ──────────────────────────────────────────────────────
	mux.HandleFunc("/api/gcp/connections", protect(handlers.ListGCP))
	mux.HandleFunc("/api/gcp/connection", protect(handlers.CreateGCP))
	mux.HandleFunc("/api/gcp/connection/", protect(handlers.GCPConnByID))
	mux.HandleFunc("/api/gcp/test", protect(handlers.TestGCP))
	mux.HandleFunc("/api/gcp/bucket/browse", protect(handlers.BrowseGCPBucket))
	mux.HandleFunc("/api/gcp/bucket/objects", protect(handlers.ListGCPObjects))
	mux.HandleFunc("/api/gcp/bucket/download", protect(handlers.GCPDownloadURL))
	mux.HandleFunc("/api/gcp/bucket/delete", protect(handlers.DeleteGCPObject))
	mux.HandleFunc("/api/gcp/bucket/copy", protect(handlers.CopyGCPObject))
	mux.HandleFunc("/api/gcp/bucket/upload", protect(handlers.UploadGCPObject))
	mux.HandleFunc("/api/gcp/bucket/stats", protect(handlers.GCPBucketStats))
	mux.HandleFunc("/api/gcp/bucket/metadata", protect(handlers.GetGCPMetadata))
	mux.HandleFunc("/api/gcp/bucket/metadata/update", protect(handlers.UpdateGCPMetadata))
	mux.HandleFunc("/api/gcp/bucket/delete-prefix", protect(handlers.DeletePrefixGCP))

	// ── AWS ──────────────────────────────────────────────────────
	mux.HandleFunc("/api/aws/connections", protect(handlers.ListAWS))
	mux.HandleFunc("/api/aws/connection", protect(handlers.CreateAWS))
	mux.HandleFunc("/api/aws/connection/", protect(handlers.AWSConnByID))
	mux.HandleFunc("/api/aws/test", protect(handlers.TestAWS))
	mux.HandleFunc("/api/aws/bucket/browse", protect(handlers.BrowseAWSBucket))
	mux.HandleFunc("/api/aws/bucket/objects", protect(handlers.ListAWSObjects))
	mux.HandleFunc("/api/aws/bucket/download", protect(handlers.AWSDownloadURL))
	mux.HandleFunc("/api/aws/bucket/delete", protect(handlers.DeleteAWSObject))
	mux.HandleFunc("/api/aws/bucket/copy", protect(handlers.CopyAWSObject))
	mux.HandleFunc("/api/aws/bucket/upload", protect(handlers.UploadAWSObject))
	mux.HandleFunc("/api/aws/bucket/stats", protect(handlers.AWSBucketStats))
	mux.HandleFunc("/api/aws/bucket/metadata", protect(handlers.GetAWSMetadata))
	mux.HandleFunc("/api/aws/bucket/metadata/update", protect(handlers.UpdateAWSMetadata))
	mux.HandleFunc("/api/aws/bucket/delete-prefix", protect(handlers.DeletePrefixAWS))

	// ── Huawei OBS ───────────────────────────────────────────────
	mux.HandleFunc("/api/huawei/connections", protect(handlers.ListHuawei))
	mux.HandleFunc("/api/huawei/connection", protect(handlers.CreateHuawei))
	mux.HandleFunc("/api/huawei/connection/", protect(handlers.HuaweiConnByID))
	mux.HandleFunc("/api/huawei/test", protect(handlers.TestHuawei))
	mux.HandleFunc("/api/huawei/bucket/browse", protect(handlers.BrowseHuaweiBucket))
	mux.HandleFunc("/api/huawei/bucket/objects", protect(handlers.ListHuaweiObjects))
	mux.HandleFunc("/api/huawei/bucket/download", protect(handlers.HuaweiDownloadURL))
	mux.HandleFunc("/api/huawei/bucket/delete", protect(handlers.DeleteHuaweiObject))
	mux.HandleFunc("/api/huawei/bucket/copy", protect(handlers.CopyHuaweiObject))
	mux.HandleFunc("/api/huawei/bucket/upload", protect(handlers.UploadHuaweiObject))
	mux.HandleFunc("/api/huawei/bucket/stats", protect(handlers.HuaweiBucketStats))
	mux.HandleFunc("/api/huawei/bucket/metadata", protect(handlers.GetHuaweiMetadata))
	mux.HandleFunc("/api/huawei/bucket/metadata/update", protect(handlers.UpdateHuaweiMetadata))
	mux.HandleFunc("/api/huawei/bucket/delete-prefix", protect(handlers.DeletePrefixHuawei))

	// ── Alibaba Cloud OSS ────────────────────────────────────────
	mux.HandleFunc("/api/alibaba/connections", protect(handlers.ListAlibaba))
	mux.HandleFunc("/api/alibaba/connection", protect(handlers.CreateAlibaba))
	mux.HandleFunc("/api/alibaba/connection/", protect(handlers.AlibabaConnByID))
	mux.HandleFunc("/api/alibaba/test", protect(handlers.TestAlibaba))
	mux.HandleFunc("/api/alibaba/bucket/browse", protect(handlers.BrowseAlibabaBucket))
	mux.HandleFunc("/api/alibaba/bucket/objects", protect(handlers.ListAlibabaObjects))
	mux.HandleFunc("/api/alibaba/bucket/download", protect(handlers.AlibabaDownloadURL))
	mux.HandleFunc("/api/alibaba/bucket/delete", protect(handlers.DeleteAlibabaObject))
	mux.HandleFunc("/api/alibaba/bucket/copy", protect(handlers.CopyAlibabaObject))
	mux.HandleFunc("/api/alibaba/bucket/upload", protect(handlers.UploadAlibabaObject))
	mux.HandleFunc("/api/alibaba/bucket/stats", protect(handlers.AlibabaBucketStats))
	mux.HandleFunc("/api/alibaba/bucket/metadata", protect(handlers.GetAlibabaMetadata))
	mux.HandleFunc("/api/alibaba/bucket/metadata/update", protect(handlers.UpdateAlibabaMetadata))
	mux.HandleFunc("/api/alibaba/bucket/delete-prefix", protect(handlers.DeletePrefixAlibaba))

	// ── Azure Blob Storage ───────────────────────────────────────
	mux.HandleFunc("/api/azure/connections", protect(handlers.ListAzure))
	mux.HandleFunc("/api/azure/connection", protect(handlers.CreateAzure))
	mux.HandleFunc("/api/azure/connection/", protect(handlers.AzureConnByID))
	mux.HandleFunc("/api/azure/test", protect(handlers.TestAzure))
	mux.HandleFunc("/api/azure/bucket/browse", protect(handlers.BrowseAzureBucket))
	mux.HandleFunc("/api/azure/bucket/objects", protect(handlers.ListAzureObjects))
	mux.HandleFunc("/api/azure/bucket/download", protect(handlers.AzureDownloadURL))
	mux.HandleFunc("/api/azure/bucket/delete", protect(handlers.DeleteAzureObject))
	mux.HandleFunc("/api/azure/bucket/copy", protect(handlers.CopyAzureObject))
	mux.HandleFunc("/api/azure/bucket/upload", protect(handlers.UploadAzureObject))
	mux.HandleFunc("/api/azure/bucket/stats", protect(handlers.AzureBucketStats))
	mux.HandleFunc("/api/azure/bucket/metadata", protect(handlers.GetAzureMetadata))
	mux.HandleFunc("/api/azure/bucket/metadata/update", protect(handlers.UpdateAzureMetadata))
	mux.HandleFunc("/api/azure/bucket/delete-prefix", protect(handlers.DeletePrefixAzure))

	// ── Google Drive ─────────────────────────────────────────────
	mux.HandleFunc("/api/gdrive/connections", protect(handlers.ListGDrive))
	mux.HandleFunc("/api/gdrive/connection", protect(handlers.CreateGDrive))
	mux.HandleFunc("/api/gdrive/connection/", protect(handlers.GDriveConnByID))
	mux.HandleFunc("/api/gdrive/test", protect(handlers.TestGDrive))
	mux.HandleFunc("/api/gdrive/bucket/browse", protect(handlers.BrowseGDriveBucket))
	mux.HandleFunc("/api/gdrive/bucket/objects", protect(handlers.ListGDriveObjects))
	mux.HandleFunc("/api/gdrive/bucket/download", protect(handlers.GDriveDownloadURL))
	mux.HandleFunc("/api/gdrive/bucket/delete", protect(handlers.DeleteGDriveObject))
	mux.HandleFunc("/api/gdrive/bucket/copy", protect(handlers.CopyGDriveObject))
	mux.HandleFunc("/api/gdrive/bucket/upload", protect(handlers.UploadGDriveObject))
	mux.HandleFunc("/api/gdrive/bucket/stats", protect(handlers.GDriveBucketStats))
	mux.HandleFunc("/api/gdrive/bucket/metadata", protect(handlers.GetGDriveMetadata))
	mux.HandleFunc("/api/gdrive/bucket/metadata/update", protect(handlers.UpdateGDriveMetadata))

	// ── Background jobs ─────────────────────────────────────────
	mux.HandleFunc("/api/jobs", protect(handlers.JobsRoute))  // POST=CreateJob, GET=ListJobs
	mux.HandleFunc("/api/jobs/", protect(handlers.GetJob))    // GET /api/jobs/{id}

	// ── File proxy (avoids CORS for preview) ────────────────────
	mux.HandleFunc("/api/proxy/download", protect(handlers.ProxyDownload))

	// ── Cross-connection transfer ────────────────────────────────
	mux.HandleFunc("/api/transfer", protect(handlers.TransferObject))

	// ── Zip download ─────────────────────────────────────────────
	mux.HandleFunc("/api/zip", protect(handlers.ZipObjects))

	// ── Shared links (public access by token) ────────────────────
	mux.HandleFunc("/api/share", protect(handlers.CreateSharedLink))
	mux.HandleFunc("/api/share/", public(handlers.AccessSharedLink))
	mux.HandleFunc("/api/shares", protect(handlers.ListSharedLinks))
	mux.HandleFunc("/api/shares/", protect(handlers.DeleteSharedLink))

	// ── Audit log ────────────────────────────────────────────────
	mux.HandleFunc("/api/audit", protect(handlers.ListAuditLog))

	// ── Search ───────────────────────────────────────────────────
	mux.HandleFunc("/api/search", protect(handlers.SearchObjects))

	// ── Analytics ────────────────────────────────────────────────
	mux.HandleFunc("/api/analytics", protect(handlers.AnalyticsSummary))

	// ── Health ───────────────────────────────────────────────────
	mux.HandleFunc("/api/health", public(handlers.HealthCheck))

	// ── Connection import/export ─────────────────────────────────
	mux.HandleFunc("/api/connections/export", protect(handlers.ExportConnections))
	mux.HandleFunc("/api/connections/import", protect(handlers.ImportConnections))

	// ── Bulk transfer ────────────────────────────────────────────
	mux.HandleFunc("/api/transfer/bulk", protect(handlers.BulkTransferObjects))

	// ── Webhooks ─────────────────────────────────────────────────
	mux.HandleFunc("/api/webhooks", protect(handlers.WebhooksRoute))
	mux.HandleFunc("/api/webhooks/", protect(handlers.DeleteWebhook))

	// ── Notifications ───────────────────────────────────────────
	mux.HandleFunc("/api/notifications", admin(handlers.NotificationChannelsRoute))
	mux.HandleFunc("/api/notifications/test", admin(handlers.TestNotificationChannel))
	mux.HandleFunc("/api/notifications/", admin(handlers.DeleteNotificationChannel))

	// ── Sync jobs ───────────────────────────────────────────────
	mux.HandleFunc("/api/sync", protect(handlers.SyncJobsRoute))
	mux.HandleFunc("/api/sync/", protect(handlers.SyncJobByID))

	// ── User management (admin only) ────────────────────────────
	mux.HandleFunc("/api/users", admin(handlers.UsersRoute))
	mux.HandleFunc("/api/users/", admin(handlers.UserByID))

	// ── OAuth2 ──────────────────────────────────────────────────
	mux.HandleFunc("/api/auth/oauth/config", public(handlers.OAuthConfigHandler))
	mux.HandleFunc("/api/auth/oauth/google/callback", public(handlers.OAuthGoogleCallback(cfg.JWTSecret, cfg.JWTExpiry)))
	mux.HandleFunc("/api/auth/oauth/github/callback", public(handlers.OAuthGitHubCallback(cfg.JWTSecret, cfg.JWTExpiry)))

	// ── Prometheus metrics ──────────────────────────────────────
	mux.HandleFunc("/api/metrics", public(handlers.MetricsHandler))

	// ── Docs (always public) ─────────────────────────────────────
	mux.HandleFunc("/api/docs/", public(handlers.ServeDocs))

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		log.Printf("starting backend on %s (auth=%v)", srv.Addr, cfg.AuthEnabled)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down gracefully…")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped")
}

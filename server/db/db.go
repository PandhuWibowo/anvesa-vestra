package db

import (
	"database/sql"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	DB   *sql.DB
	once sync.Once
)

func Init(dbPath string) error {
	var initErr error
	once.Do(func() {
		var err error
		DB, err = sql.Open("sqlite", "file:"+dbPath+"?_foreign_keys=1&_journal_mode=WAL&_busy_timeout=5000")
		if err != nil {
			initErr = err
			return
		}
		DB.SetMaxOpenConns(10)
		DB.SetMaxIdleConns(5)
		if _, err := DB.Exec("PRAGMA journal_mode=WAL"); err != nil {
			initErr = err
			return
		}
		initErr = createTables()
	})
	return initErr
}

func createTables() error {
	stmts := []string{
		// ── Provider connections ─────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS gcp_connections (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL,
			bucket      TEXT NOT NULL,
			credentials TEXT NOT NULL,
			created_at  DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS aws_connections (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL,
			bucket      TEXT NOT NULL,
			credentials TEXT NOT NULL,
			created_at  DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS huawei_connections (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL,
			bucket      TEXT NOT NULL,
			credentials TEXT NOT NULL,
			created_at  DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS alibaba_connections (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL,
			bucket      TEXT NOT NULL,
			credentials TEXT NOT NULL,
			created_at  DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS azure_connections (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL,
			bucket      TEXT NOT NULL,
			credentials TEXT NOT NULL,
			created_at  DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS gdrive_connections (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL,
			bucket      TEXT NOT NULL,
			credentials TEXT NOT NULL,
			created_at  DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS b2_connections (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL,
			bucket      TEXT NOT NULL,
			credentials TEXT NOT NULL,
			created_at  DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS do_connections (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL,
			bucket      TEXT NOT NULL,
			credentials TEXT NOT NULL,
			created_at  DATETIME NOT NULL
		)`,
		// ── Users ───────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS users (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			username      TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role          TEXT NOT NULL DEFAULT 'admin',
			created_at    DATETIME NOT NULL
		)`,
		// ── Audit log ───────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS audit_log (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER REFERENCES users(id),
			action     TEXT NOT NULL,
			provider   TEXT,
			bucket     TEXT,
			object     TEXT,
			details    TEXT,
			ip         TEXT,
			created_at DATETIME NOT NULL
		)`,
		// ── Shared links ────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS shared_links (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			token          TEXT NOT NULL UNIQUE,
			connection_id  INTEGER NOT NULL,
			provider       TEXT NOT NULL,
			object         TEXT NOT NULL,
			password_hash  TEXT,
			expires_at     DATETIME,
			max_downloads  INTEGER DEFAULT 0,
			download_count INTEGER DEFAULT 0,
			created_by     INTEGER REFERENCES users(id),
			created_at     DATETIME NOT NULL
		)`,
		// ── Webhooks ────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS webhooks (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			url        TEXT NOT NULL,
			events     TEXT NOT NULL,
			secret     TEXT,
			active     INTEGER DEFAULT 1,
			created_at DATETIME NOT NULL
		)`,
		// ── Sync jobs ───────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS sync_jobs (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			name               TEXT NOT NULL,
			src_connection_id  INTEGER NOT NULL,
			src_provider       TEXT NOT NULL,
			dst_connection_id  INTEGER NOT NULL,
			dst_provider       TEXT NOT NULL,
			src_prefix         TEXT DEFAULT '',
			dst_prefix         TEXT DEFAULT '',
			schedule           TEXT DEFAULT '',
			last_run           DATETIME,
			next_run           DATETIME,
			status             TEXT DEFAULT 'idle',
			created_at         DATETIME NOT NULL
		)`,
		// ── Background jobs ─────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS jobs (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			type       TEXT NOT NULL,
			status     TEXT NOT NULL DEFAULT 'pending',
			payload    TEXT NOT NULL,
			result     TEXT,
			error      TEXT,
			progress   REAL DEFAULT 0,
			user_id    INTEGER REFERENCES users(id),
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		// ── Notification channels ───────────────────────────────────
		`CREATE TABLE IF NOT EXISTS notification_channels (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			type       TEXT NOT NULL,
			config     TEXT NOT NULL,
			events     TEXT NOT NULL,
			active     INTEGER DEFAULT 1,
			created_at DATETIME NOT NULL
		)`,
		// ── Indexes ──────────────────────────────────────────────────
		`CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action)`,
		`CREATE INDEX IF NOT EXISTS idx_shared_links_token ON shared_links(token)`,
		`CREATE INDEX IF NOT EXISTS idx_shared_links_provider ON shared_links(provider, connection_id)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_user_id ON jobs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sync_jobs_status ON sync_jobs(status)`,
	}
	for _, ddl := range stmts {
		if _, err := DB.Exec(ddl); err != nil {
			return err
		}
	}
	return nil
}

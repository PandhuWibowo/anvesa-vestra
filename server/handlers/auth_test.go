package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PandhuWibowo/oss-portable/auth"
	"github.com/PandhuWibowo/oss-portable/crypto"
	db "github.com/PandhuWibowo/oss-portable/db"
	"github.com/PandhuWibowo/oss-portable/handlers"
	"github.com/PandhuWibowo/oss-portable/middleware"
	_ "modernc.org/sqlite"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "anveesa-test-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	var openErr error
	db.DB, openErr = sql.Open("sqlite", "file:"+filepath.Join(dir, "test.db")+"?_foreign_keys=1&_journal_mode=WAL")
	if openErr != nil {
		log.Fatal(openErr)
	}
	defer db.DB.Close()

	createTestTables()

	os.Exit(m.Run())
}

func createTestTables() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, role TEXT NOT NULL DEFAULT 'admin', created_at DATETIME NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS aws_connections (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, bucket TEXT NOT NULL, credentials TEXT NOT NULL, created_at DATETIME NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS gcp_connections (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, bucket TEXT NOT NULL, credentials TEXT NOT NULL, created_at DATETIME NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS audit_log (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER, action TEXT NOT NULL, provider TEXT, bucket TEXT, object TEXT, details TEXT, ip TEXT, created_at DATETIME NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS shared_links (id INTEGER PRIMARY KEY AUTOINCREMENT, token TEXT NOT NULL UNIQUE, connection_id INTEGER NOT NULL, provider TEXT NOT NULL, object TEXT NOT NULL, password_hash TEXT, expires_at DATETIME, max_downloads INTEGER DEFAULT 0, download_count INTEGER DEFAULT 0, created_by INTEGER, created_at DATETIME NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS webhooks (id INTEGER PRIMARY KEY AUTOINCREMENT, url TEXT NOT NULL, events TEXT NOT NULL, secret TEXT, active INTEGER DEFAULT 1, created_at DATETIME NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS jobs (id INTEGER PRIMARY KEY AUTOINCREMENT, type TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending', payload TEXT NOT NULL, result TEXT, error TEXT, progress REAL DEFAULT 0, user_id INTEGER, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL)`,
	}
	for _, s := range stmts {
		if _, err := db.DB.Exec(s); err != nil {
			log.Fatalf("createTestTables: %v", err)
		}
	}
}

func TestRegisterHandler(t *testing.T) {
	db.DB.Exec("DELETE FROM users")

	register := handlers.RegisterHandler("test-secret", time.Hour)

	body := map[string]string{"username": "testuser", "password": "testpassword123"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	register(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected token in response")
	}
	if resp["user_id"] == nil {
		t.Error("expected user_id in response")
	}
	if resp["username"] != "testuser" {
		t.Errorf("expected username testuser, got %v", resp["username"])
	}
	if resp["role"] != "admin" {
		t.Errorf("expected role admin, got %v", resp["role"])
	}

	// Second POST should fail with 403 (registration closed)
	req2 := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	register(rr2, req2)

	if rr2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on second register, got %d: %s", rr2.Code, rr2.Body.String())
	}
}

func TestLoginHandler(t *testing.T) {
	db.DB.Exec("DELETE FROM users")

	hash, err := crypto.HashPassword("testpassword123")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.DB.Exec("INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, 'admin', ?)",
		"loginuser", hash, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	login := handlers.LoginHandler("test-secret", time.Hour)

	// Correct creds -> 200
	body := map[string]string{"username": "loginuser", "password": "testpassword123"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Wrong password -> 401
	bodyBad := map[string]string{"username": "loginuser", "password": "wrongpassword"}
	bodyBadBytes, _ := json.Marshal(bodyBad)
	req2 := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(bodyBadBytes))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	login(rr2, req2)

	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d: %s", rr2.Code, rr2.Body.String())
	}

	// Empty body -> 400
	req3 := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader([]byte("{}")))
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	login(rr3, req3)

	if rr3.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d: %s", rr3.Code, rr3.Body.String())
	}
}

func TestSetupStatusHandler(t *testing.T) {
	setupStatus := handlers.SetupStatusHandler(true)

	// Before any user: setup_required: true
	db.DB.Exec("DELETE FROM users")

	req := httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)
	rr := httptest.NewRecorder()
	setupStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["setup_required"] != true {
		t.Errorf("expected setup_required true when no users, got %v", resp["setup_required"])
	}

	// After a user is created: setup_required: false
	hash, _ := crypto.HashPassword("pw")
	db.DB.Exec("INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, 'admin', ?)",
		"setupuser", hash, time.Now().UTC())

	req2 := httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)
	rr2 := httptest.NewRecorder()
	setupStatus(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}

	if err := json.NewDecoder(rr2.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["setup_required"] != false {
		t.Errorf("expected setup_required false when user exists, got %v", resp["setup_required"])
	}
}

func TestMeHandler(t *testing.T) {
	token, err := auth.GenerateToken(42, "meuser", "admin", "test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	claims := &auth.Claims{UserID: 42, Username: "meuser", Role: "admin"}
	ctx := context.WithValue(context.Background(), middleware.ClaimsKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handlers.MeHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["user_id"] != float64(42) {
		t.Errorf("expected user_id 42, got %v", resp["user_id"])
	}
	if resp["username"] != "meuser" {
		t.Errorf("expected username meuser, got %v", resp["username"])
	}
	if resp["role"] != "admin" {
		t.Errorf("expected role admin, got %v", resp["role"])
	}
}

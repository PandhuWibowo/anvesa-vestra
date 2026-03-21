package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

// ── Handlers ──────────────────────────────────────────────────────────────────

// WebhooksRoute dispatches GET (list) and POST (create) for /api/webhooks.
func WebhooksRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListWebhooks(w, r)
	case http.MethodPost:
		CreateWebhook(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ListWebhooks handles GET /api/webhooks. Returns all webhooks.
func ListWebhooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := appdb.DB.Query("SELECT id, url, events, active, created_at FROM webhooks ORDER BY id")
	if err != nil {
		jsonError(w, safeError(err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var webhooks []map[string]any
	for rows.Next() {
		var id int64
		var webhookURL, events, createdAt string
		var active int
		if err := rows.Scan(&id, &webhookURL, &events, &active, &createdAt); err != nil {
			jsonError(w, safeError(err), http.StatusInternalServerError)
			return
		}
		webhooks = append(webhooks, map[string]any{
			"id":         id,
			"url":        webhookURL,
			"events":     strings.Split(events, ","),
			"active":     active == 1,
			"created_at": createdAt,
		})
	}

	jsonOK(w, webhooks)
}

type createWebhookReq struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret"`
}

// CreateWebhook handles POST /api/webhooks. Accepts {url, events, secret}.
func CreateWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r = limitBody(r, MaxBodySize)
	var req createWebhookReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.URL) == "" {
		jsonError(w, "url is required", http.StatusBadRequest)
		return
	}

	parsed, err := url.ParseRequestURI(req.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		jsonError(w, "invalid URL", http.StatusBadRequest)
		return
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		jsonError(w, "URL must use http or https", http.StatusBadRequest)
		return
	}

	if len(req.Events) == 0 {
		jsonError(w, "events is required and must not be empty", http.StatusBadRequest)
		return
	}

	eventsStr := strings.Join(req.Events, ",")
	createdAt := time.Now().UTC().Format("2006-01-02 15:04:05")

	res, err := appdb.DB.Exec(
		"INSERT INTO webhooks (url, events, secret, active, created_at) VALUES (?, ?, ?, 1, ?)",
		req.URL, eventsStr, req.Secret, createdAt,
	)
	if err != nil {
		jsonError(w, safeError(err), http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	jsonOK(w, map[string]any{
		"id":         id,
		"url":        req.URL,
		"events":     req.Events,
		"active":     true,
		"created_at": createdAt,
	})
}

// DeleteWebhook handles DELETE /api/webhooks/{id}.
func DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := parseID(r, 3)
	if err != nil {
		jsonError(w, "invalid webhook id", http.StatusBadRequest)
		return
	}

	res, err := appdb.DB.Exec("DELETE FROM webhooks WHERE id = ?", id)
	if err != nil {
		jsonError(w, safeError(err), http.StatusInternalServerError)
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		jsonError(w, "webhook not found", http.StatusNotFound)
		return
	}

	jsonOK(w, map[string]string{"status": "deleted"})
}

// ── Internal: FireWebhookEvent ─────────────────────────────────────────────────

// FireWebhookEvent spawns goroutines to POST the payload as JSON to all active
// webhooks that subscribe to the given event. If a webhook has a secret, the
// request includes an X-Webhook-Signature header with HMAC-SHA256 of the body.
// Each request has a 5-second timeout.
func FireWebhookEvent(event string, payload map[string]any) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	rows, err := appdb.DB.Query("SELECT id, url, events, secret FROM webhooks WHERE active = 1")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var webhookURL, events, secret string
		if err := rows.Scan(&id, &webhookURL, &events, &secret); err != nil {
			continue
		}

		eventList := strings.Split(events, ",")
		found := false
		for _, e := range eventList {
			if strings.TrimSpace(e) == event {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		go func(url, secret string, body []byte) {
			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")

			if secret != "" {
				mac := hmac.New(sha256.New, []byte(secret))
				mac.Write(body)
				req.Header.Set("X-Webhook-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
			}

			client := &http.Client{Timeout: 5 * time.Second}
			_, _ = client.Do(req)
		}(webhookURL, secret, body)
	}
}

package handlers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

var validChannelTypes = map[string]bool{
	"slack":   true,
	"discord": true,
	"teams":   true,
	"email":   true,
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// NotificationChannelsRoute dispatches GET (list) and POST (create) for /api/notifications.
func NotificationChannelsRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listNotificationChannels(w, r)
	case http.MethodPost:
		createNotificationChannel(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func listNotificationChannels(w http.ResponseWriter, r *http.Request) {
	rows, err := appdb.DB.Query("SELECT id, name, type, config, events, active, created_at FROM notification_channels ORDER BY id")
	if err != nil {
		jsonError(w, safeError(err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var channels []map[string]any
	for rows.Next() {
		var id int64
		var name, chType, config, events, createdAt string
		var active int
		if err := rows.Scan(&id, &name, &chType, &config, &events, &active, &createdAt); err != nil {
			jsonError(w, safeError(err), http.StatusInternalServerError)
			return
		}
		channels = append(channels, map[string]any{
			"id":         id,
			"name":       name,
			"type":       chType,
			"config":     json.RawMessage(config),
			"events":     strings.Split(events, ","),
			"active":     active == 1,
			"created_at": createdAt,
		})
	}

	jsonOK(w, channels)
}

type createChannelReq struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
	Events []string        `json:"events"`
}

func createNotificationChannel(w http.ResponseWriter, r *http.Request) {
	r = limitBody(r, MaxBodySize)
	var req createChannelReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}

	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	if !validChannelTypes[req.Type] {
		jsonError(w, "type must be one of: slack, discord, teams, email", http.StatusBadRequest)
		return
	}

	if len(req.Config) == 0 {
		jsonError(w, "config is required", http.StatusBadRequest)
		return
	}

	if len(req.Events) == 0 {
		jsonError(w, "events is required and must not be empty", http.StatusBadRequest)
		return
	}

	eventsStr := strings.Join(req.Events, ",")
	createdAt := time.Now().UTC().Format("2006-01-02 15:04:05")

	res, err := appdb.DB.Exec(
		"INSERT INTO notification_channels (name, type, config, events, active, created_at) VALUES (?, ?, ?, ?, 1, ?)",
		req.Name, req.Type, string(req.Config), eventsStr, createdAt,
	)
	if err != nil {
		jsonError(w, safeError(err), http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	jsonOK(w, map[string]any{
		"id":         id,
		"name":       req.Name,
		"type":       req.Type,
		"config":     req.Config,
		"events":     req.Events,
		"active":     true,
		"created_at": createdAt,
	})
}

// DeleteNotificationChannel handles DELETE /api/notifications/{id}.
func DeleteNotificationChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := parseID(r, 3)
	if err != nil {
		jsonError(w, "invalid notification channel id", http.StatusBadRequest)
		return
	}

	res, err := appdb.DB.Exec("DELETE FROM notification_channels WHERE id = ?", id)
	if err != nil {
		jsonError(w, safeError(err), http.StatusInternalServerError)
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		jsonError(w, "notification channel not found", http.StatusNotFound)
		return
	}

	jsonOK(w, map[string]any{"ok": true})
}

// ── Test ──────────────────────────────────────────────────────────────────────

type testChannelReq struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

type emailConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	To       string `json:"to"`
}

// TestNotificationChannel handles POST /api/notifications/test.
func TestNotificationChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r = limitBody(r, MaxBodySize)
	var req testChannelReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	if !validChannelTypes[req.Type] {
		jsonError(w, "type must be one of: slack, discord, teams, email", http.StatusBadRequest)
		return
	}

	const testMsg = "Anveesa Vestra test notification"

	switch req.Type {
	case "slack":
		if err := sendSlack(req.Config, testMsg); err != nil {
			jsonError(w, fmt.Sprintf("slack test failed: %v", err), http.StatusBadGateway)
			return
		}
	case "discord":
		if err := sendDiscord(req.Config, testMsg); err != nil {
			jsonError(w, fmt.Sprintf("discord test failed: %v", err), http.StatusBadGateway)
			return
		}
	case "teams":
		if err := sendTeams(req.Config, testMsg); err != nil {
			jsonError(w, fmt.Sprintf("teams test failed: %v", err), http.StatusBadGateway)
			return
		}
	case "email":
		if err := sendEmail(req.Config, "Anveesa Vestra Test", testMsg); err != nil {
			jsonError(w, fmt.Sprintf("email test failed: %v", err), http.StatusBadGateway)
			return
		}
	}

	jsonOK(w, map[string]any{"ok": true})
}

// ── Internal: FireNotification ────────────────────────────────────────────────

// FireNotification spawns goroutines to send the event payload to all active
// notification channels that subscribe to the given event.
// Each send has a 5-second timeout. Errors are silently ignored (fire and forget).
func FireNotification(event string, payload map[string]any) {
	rows, err := appdb.DB.Query("SELECT id, name, type, config, events FROM notification_channels WHERE active = 1")
	if err != nil {
		return
	}
	defer rows.Close()

	msg := formatNotificationMessage(event, payload)

	for rows.Next() {
		var id int64
		var name, chType, config, events string
		if err := rows.Scan(&id, &name, &chType, &config, &events); err != nil {
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

		go func(chType, config, msg, event string, payload map[string]any) {
			switch chType {
			case "slack":
				_ = sendSlackRich([]byte(config), event, msg, payload)
			case "discord":
				_ = sendDiscordRich([]byte(config), event, msg, payload)
			case "teams":
				_ = sendTeams([]byte(config), msg)
			case "email":
				subject := fmt.Sprintf("Anveesa Vestra — %s", event)
				_ = sendEmail([]byte(config), subject, msg)
			}
		}(chType, config, msg, event, payload)
	}
}

// ── Senders ──────────────────────────────────────────────────────────────────

func notifyClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

func webhookURL(raw json.RawMessage) (string, error) {
	var cfg struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return "", err
	}
	if cfg.URL == "" {
		return "", fmt.Errorf("url is required in config")
	}
	return cfg.URL, nil
}

func sendSlack(configJSON json.RawMessage, text string) error {
	u, err := webhookURL(configJSON)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"text": text})
	resp, err := notifyClient().Post(u, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}
	return nil
}

func sendSlackRich(configJSON json.RawMessage, event, text string, payload map[string]any) error {
	u, err := webhookURL(configJSON)
	if err != nil {
		return err
	}

	blocks := []map[string]any{
		{
			"type": "header",
			"text": map[string]string{"type": "plain_text", "text": fmt.Sprintf("Anveesa Vestra — %s", event)},
		},
		{
			"type": "section",
			"text": map[string]string{"type": "mrkdwn", "text": text},
		},
	}

	body, _ := json.Marshal(map[string]any{"text": text, "blocks": blocks})
	resp, err := notifyClient().Post(u, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func sendDiscord(configJSON json.RawMessage, text string) error {
	u, err := webhookURL(configJSON)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"content": text})
	resp, err := notifyClient().Post(u, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord returned status %d", resp.StatusCode)
	}
	return nil
}

func sendDiscordRich(configJSON json.RawMessage, event, text string, payload map[string]any) error {
	u, err := webhookURL(configJSON)
	if err != nil {
		return err
	}

	embeds := []map[string]any{
		{
			"title":       fmt.Sprintf("Anveesa Vestra — %s", event),
			"description": text,
			"color":       3447003, // blue
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		},
	}

	body, _ := json.Marshal(map[string]any{"content": text, "embeds": embeds})
	resp, err := notifyClient().Post(u, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func sendTeams(configJSON json.RawMessage, text string) error {
	u, err := webhookURL(configJSON)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"text": text})
	resp, err := notifyClient().Post(u, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("teams returned status %d", resp.StatusCode)
	}
	return nil
}

func sendEmail(configJSON json.RawMessage, subject, body string) error {
	var cfg emailConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return fmt.Errorf("invalid email config: %w", err)
	}

	if cfg.Host == "" || cfg.Port == "" || cfg.From == "" || cfg.To == "" {
		return fmt.Errorf("host, port, from, and to are required for email")
	}

	addr := cfg.Host + ":" + cfg.Port

	msg := "From: " + cfg.From + "\r\n" +
		"To: " + cfg.To + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=\"utf-8\"\r\n" +
		"\r\n" +
		body + "\r\n"

	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	// Try STARTTLS first for port 587, direct TLS for 465, plain for others.
	switch cfg.Port {
	case "465":
		return sendEmailTLS(addr, cfg, auth, msg)
	default:
		return smtp.SendMail(addr, auth, cfg.From, []string{cfg.To}, []byte(msg))
	}
}

func sendEmailTLS(addr string, cfg emailConfig, auth smtp.Auth, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host})
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP client failed: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}
	if err := client.Mail(cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(cfg.To); err != nil {
		return err
	}

	wc, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write([]byte(msg)); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}

	return client.Quit()
}

// ── Formatting ───────────────────────────────────────────────────────────────

func formatNotificationMessage(event string, payload map[string]any) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Event: %s\n", event))

	if file, ok := payload["file"].(string); ok && file != "" {
		b.WriteString(fmt.Sprintf("File: %s\n", file))
	}
	if bucket, ok := payload["bucket"].(string); ok && bucket != "" {
		b.WriteString(fmt.Sprintf("Bucket: %s\n", bucket))
	}
	if provider, ok := payload["provider"].(string); ok && provider != "" {
		b.WriteString(fmt.Sprintf("Provider: %s\n", provider))
	}
	if ts, ok := payload["timestamp"].(string); ok && ts != "" {
		b.WriteString(fmt.Sprintf("Time: %s\n", ts))
	}

	for k, v := range payload {
		switch k {
		case "file", "bucket", "provider", "timestamp", "event":
			continue
		default:
			b.WriteString(fmt.Sprintf("%s: %v\n", k, v))
		}
	}

	return strings.TrimSpace(b.String())
}

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	appdb "github.com/PandhuWibowo/oss-portable/db"
)

const exportVersion = 1

type exportConnection struct {
	Provider    string `json:"provider"`
	Name        string `json:"name"`
	Bucket      string `json:"bucket"`
	Credentials string `json:"credentials"`
}

type exportPayload struct {
	Version     int               `json:"version"`
	ExportedAt  string            `json:"exported_at"`
	Connections []exportConnection `json:"connections"`
}

type importConnection struct {
	Provider    string `json:"provider"`
	Name        string `json:"name"`
	Bucket      string `json:"bucket"`
	Credentials string `json:"credentials"`
}

type importPayload struct {
	Connections []importConnection `json:"connections"`
}

// ExportConnections exports all connections from all provider tables as JSON.
// GET method only. Credentials are decrypted before export.
func ExportConnections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var connections []exportConnection
	for provider, table := range providerTable {
		rows, err := appdb.DB.Query(fmt.Sprintf("SELECT name, bucket, credentials FROM %s", table))
		if err != nil {
			continue // skip tables that don't exist or fail
		}
		for rows.Next() {
			var name, bucket, credentials string
			if err := rows.Scan(&name, &bucket, &credentials); err != nil {
				rows.Close()
				continue
			}
			decrypted, err := decryptCredentials(credentials)
			if err != nil {
				decrypted = credentials // fallback to raw on decrypt failure
			}
			connections = append(connections, exportConnection{
				Provider:    provider,
				Name:        name,
				Bucket:      bucket,
				Credentials: decrypted,
			})
		}
		rows.Close()
	}

	payload := exportPayload{
		Version:     exportVersion,
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		Connections: connections,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="anveesa-connections.json"`)
	_ = json.NewEncoder(w).Encode(payload)
}

// ImportConnections imports connections from JSON. Credentials are encrypted before storage.
// POST method only. Skips connections for unknown providers.
func ImportConnections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r = limitBody(r, 10<<20) // 10MB limit
	var payload importPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	imported := 0

	for _, conn := range payload.Connections {
		table, ok := providerTable[conn.Provider]
		if !ok {
			continue // skip unknown providers
		}
		encrypted, err := encryptCredentials(conn.Credentials)
		if err != nil {
			continue
		}
		_, err = appdb.DB.Exec(
			fmt.Sprintf("INSERT INTO %s (name, bucket, credentials, created_at) VALUES (?, ?, ?, ?)", table),
			conn.Name, conn.Bucket, encrypted, now,
		)
		if err != nil {
			continue
		}
		imported++
	}

	jsonOK(w, map[string]int{"imported": imported})
}

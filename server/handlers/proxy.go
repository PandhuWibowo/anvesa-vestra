package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"time"
)

// ProxyDownload streams the actual file content through the backend, avoiding
// CORS issues that occur when the browser fetches presigned URLs directly.
func ProxyDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r = limitBody(r, MaxBodySize)
	var req struct {
		Provider    string `json:"provider"`
		Bucket      string `json:"bucket"`
		Credentials string `json:"credentials"`
		Object      string `json:"object"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Provider == "" || req.Bucket == "" || req.Object == "" {
		jsonError(w, "provider, bucket, and object are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	data, contentType, err := downloadObjectData(ctx, req.Provider, req.Bucket, req.Credentials, req.Object)
	if err != nil {
		jsonError(w, fmt.Sprintf("download failed: %v", err), http.StatusInternalServerError)
		return
	}

	filename := path.Base(req.Object)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Write(data)
}

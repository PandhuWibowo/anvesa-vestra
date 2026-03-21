package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"path"
	"strings"
	"time"
)

type bulkTransferRequest struct {
	SrcProvider     string   `json:"src_provider"`
	SrcConnectionID int64    `json:"src_connection_id"`
	DstProvider     string   `json:"dst_provider"`
	DstConnectionID int64    `json:"dst_connection_id"`
	DstPrefix       string   `json:"dst_prefix"`
	Objects         []string `json:"objects"`
}

type transferError struct {
	Object string `json:"object"`
	Error  string `json:"error"`
}

type bulkTransferResponse struct {
	Total   int            `json:"total"`
	Success int            `json:"success"`
	Failed  int            `json:"failed"`
	Errors  []transferError `json:"errors"`
}

// BulkTransferObjects transfers multiple objects from source to destination.
// POST method only.
func BulkTransferObjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r = limitBody(r, MaxBodySize)
	var req bulkTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Objects) == 0 {
		jsonError(w, "objects list is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	srcBucket, srcCreds, err := resolveProviderCreds(req.SrcProvider, req.SrcConnectionID, "", "")
	if err != nil {
		jsonError(w, "source connection error: "+err.Error(), http.StatusBadRequest)
		return
	}

	dstBucket, dstCreds, err := resolveProviderCreds(req.DstProvider, req.DstConnectionID, "", "")
	if err != nil {
		jsonError(w, "destination connection error: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp := bulkTransferResponse{
		Total:   len(req.Objects),
		Success: 0,
		Failed:  0,
		Errors:  []transferError{},
	}

	prefix := strings.TrimSuffix(req.DstPrefix, "/")
	for _, object := range req.Objects {
		data, contentType, err := downloadObjectData(ctx, req.SrcProvider, srcBucket, srcCreds, object)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, transferError{Object: object, Error: err.Error()})
			continue
		}

		var destKey string
		if prefix == "" {
			destKey = path.Base(object)
		} else {
			destKey = prefix + "/" + path.Base(object)
		}

		if err := uploadObjectData(ctx, req.DstProvider, dstBucket, dstCreds, destKey, data, contentType); err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, transferError{Object: object, Error: err.Error()})
			continue
		}

		resp.Success++
	}

	jsonOK(w, resp)
}

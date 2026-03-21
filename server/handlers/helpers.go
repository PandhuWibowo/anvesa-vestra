package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PandhuWibowo/oss-portable/crypto"
	appdb "github.com/PandhuWibowo/oss-portable/db"
)

const (
	MaxUploadSize   = 500 << 20 // 500 MB
	DefaultPageSize = 200
	MaxListResults  = 10000
	DefaultExpiry   = 900 // 15 minutes in seconds
	DefaultTimeout  = 30  // seconds
	MaxBodySize     = 2 << 20 // 2 MB for JSON request bodies
	MaxPresignExpiry = 7 * 24 * 60 * 60 // 7 days in seconds
)

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func requireFields(fields map[string]string) error {
	for name, val := range fields {
		if strings.TrimSpace(val) == "" {
			return fmt.Errorf("field %q is required", name)
		}
	}
	return nil
}

func parseID(r *http.Request, position int) (int64, error) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) <= position {
		return 0, fmt.Errorf("invalid path")
	}
	return strconv.ParseInt(parts[position], 10, 64)
}

func lookupConnection(table string, id int64) (bucket, credentials string, err error) {
	row := appdb.DB.QueryRow(
		fmt.Sprintf("SELECT bucket, credentials FROM %s WHERE id = ?", table), id,
	)
	err = row.Scan(&bucket, &credentials)
	if err != nil {
		return
	}
	credentials, err = decryptCredentials(credentials)
	return
}

func encryptCredentials(plaintext string) (string, error) {
	key := encryptionKey()
	if len(key) == 0 {
		return plaintext, nil
	}
	return crypto.Encrypt([]byte(plaintext), key)
}

func decryptCredentials(ciphertext string) (string, error) {
	key := encryptionKey()
	if len(key) == 0 {
		return ciphertext, nil
	}
	plain, err := crypto.Decrypt(ciphertext, key)
	if err != nil {
		// Fallback: assume plaintext (migration from unencrypted data)
		return ciphertext, nil
	}
	return string(plain), nil
}

func encryptionKey() []byte {
	k := os.Getenv("ENCRYPTION_KEY")
	if len(k) < 32 {
		return nil
	}
	return []byte(k[:32])
}

// safeError returns a generic message for internal errors to avoid leaking details.
func safeError(err error) string {
	_ = err
	return "an internal error occurred"
}

// limitBody wraps the request body with a size limit.
func limitBody(r *http.Request, maxBytes int64) *http.Request {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)
	return r
}

// sanitizeFilename strips path traversal characters from a filename.
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.ReplaceAll(name, "\x00", "")
	return strings.TrimSpace(name)
}

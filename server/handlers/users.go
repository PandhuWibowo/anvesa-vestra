package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/PandhuWibowo/oss-portable/crypto"
	appdb "github.com/PandhuWibowo/oss-portable/db"
)

// UsersRoute dispatches GET → list, POST → create.
func UsersRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListUsers(w, r)
	case http.MethodPost:
		CreateUser(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// UserByID handles /api/users/{id} (DELETE) and /api/users/{id}/role (PUT).
func UserByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/users/")
	if strings.HasSuffix(path, "/role") {
		UpdateUserRole(w, r)
		return
	}
	switch r.Method {
	case http.MethodDelete:
		DeleteUser(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := appdb.DB.Query("SELECT id, username, role, created_at FROM users ORDER BY id")
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type user struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Role      string `json:"role"`
		CreatedAt string `json:"created_at"`
	}

	users := []user{}
	for rows.Next() {
		var u user
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
			jsonError(w, "database error", http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}
	jsonOK(w, users)
}

func UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := parseID(r, 3)
	if err != nil {
		jsonError(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	validRoles := map[string]bool{"admin": true, "editor": true, "viewer": true}
	if !validRoles[req.Role] {
		jsonError(w, "role must be one of: admin, editor, viewer", http.StatusBadRequest)
		return
	}

	res, err := appdb.DB.Exec("UPDATE users SET role = ? WHERE id = ?", req.Role, id)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	jsonOK(w, map[string]bool{"ok": true})
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, 3)
	if err != nil {
		jsonError(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var role string
	if err := appdb.DB.QueryRow("SELECT role FROM users WHERE id = ?", id).Scan(&role); err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	if role == "admin" {
		var adminCount int
		appdb.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&adminCount)
		if adminCount <= 1 {
			jsonError(w, "cannot delete the last admin user", http.StatusForbidden)
			return
		}
	}

	res, err := appdb.DB.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	jsonOK(w, map[string]bool{"ok": true})
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := requireFields(map[string]string{
		"username": req.Username,
		"password": req.Password,
		"role":     req.Role,
	}); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	validRoles := map[string]bool{"admin": true, "editor": true, "viewer": true}
	if !validRoles[req.Role] {
		jsonError(w, "role must be one of: admin, editor, viewer", http.StatusBadRequest)
		return
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		jsonError(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	result, err := appdb.DB.Exec(
		"INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, ?)",
		req.Username, hash, req.Role, time.Now().UTC(),
	)
	if err != nil {
		jsonError(w, "failed to create user (username may already exist)", http.StatusConflict)
		return
	}

	id, _ := result.LastInsertId()
	jsonOK(w, map[string]any{
		"id":       id,
		"username": req.Username,
		"role":     req.Role,
	})
}

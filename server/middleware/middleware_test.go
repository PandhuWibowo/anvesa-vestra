package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PandhuWibowo/oss-portable/auth"
	"github.com/PandhuWibowo/oss-portable/middleware"
)

func TestRequireAuth(t *testing.T) {
	jwtSecret := "test-secret"
	var capturedClaims *auth.Claims

	next := func(w http.ResponseWriter, r *http.Request) {
		if claims, ok := r.Context().Value(middleware.ClaimsKey).(*auth.Claims); ok && claims != nil {
			capturedClaims = claims
		}
		w.WriteHeader(http.StatusOK)
	}

	wrapped := middleware.RequireAuth(jwtSecret)(next)

	// Request without Authorization header -> 401
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	wrapped(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth header, got %d: %s", rr.Code, rr.Body.String())
	}

	// Request with invalid token -> 401
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "Bearer invalid-token")
	rr2 := httptest.NewRecorder()
	wrapped(rr2, req2)

	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid token, got %d: %s", rr2.Code, rr2.Body.String())
	}

	// Request with valid token -> calls next handler, claims in context
	token, err := auth.GenerateToken(1, "testuser", "admin", jwtSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	capturedClaims = nil
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.Header.Set("Authorization", "Bearer "+token)
	rr3 := httptest.NewRecorder()
	wrapped(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d: %s", rr3.Code, rr3.Body.String())
	}
	if capturedClaims == nil {
		t.Fatal("expected claims in context")
	}
	if capturedClaims.UserID != 1 || capturedClaims.Username != "testuser" || capturedClaims.Role != "admin" {
		t.Errorf("unexpected claims: %+v", capturedClaims)
	}
}

func TestRequireRole(t *testing.T) {
	next := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Create fake claims with role "viewer" in context
	viewerClaims := &auth.Claims{UserID: 1, Username: "viewer", Role: "viewer"}
	ctx := context.WithValue(context.Background(), middleware.ClaimsKey, viewerClaims)

	// RequireRole("admin") -> 403
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	middleware.RequireRole("admin")(next)(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer with RequireRole(admin), got %d: %s", rr.Code, rr.Body.String())
	}

	// RequireRole("viewer", "admin") -> passes through
	req2 := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rr2 := httptest.NewRecorder()
	middleware.RequireRole("viewer", "admin")(next)(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 for viewer with RequireRole(viewer, admin), got %d: %s", rr2.Code, rr2.Body.String())
	}
}

func TestRateLimit(t *testing.T) {
	// RateLimit(1, 2) = 1 req/sec, burst 2
	limiter := middleware.RateLimit(1, 2)

	next := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	wrapped := limiter(next)

	// Fire 3 rapid requests -> first 2 succeed, third gets 429
	results := make([]int, 3)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rr := httptest.NewRecorder()
		wrapped(rr, req)
		results[i] = rr.Code
	}

	if results[0] != http.StatusOK || results[1] != http.StatusOK {
		t.Errorf("expected first two requests to succeed (200), got %v, %v", results[0], results[1])
	}
	if results[2] != http.StatusTooManyRequests {
		t.Errorf("expected third request to get 429, got %d", results[2])
	}
}

package middleware

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	lastTime time.Time
	mu       sync.Mutex
}

// RateLimit returns a middleware that enforces per-IP rate limiting using a
// token bucket algorithm. Each IP gets requestsPerSecond tokens refilled per
// second, up to a maximum of burst tokens.
func RateLimit(requestsPerSecond float64, burst int) func(http.HandlerFunc) http.HandlerFunc {
	var buckets sync.Map
	burstF := float64(burst)

	go cleanupBuckets(&buckets, 5*time.Minute)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			val, _ := buckets.LoadOrStore(ip, &bucket{
				tokens:   burstF,
				lastTime: time.Now(),
			})
			b := val.(*bucket)

			b.mu.Lock()
			now := time.Now()
			elapsed := now.Sub(b.lastTime).Seconds()
			b.tokens = math.Min(burstF, b.tokens+elapsed*requestsPerSecond)
			b.lastTime = now

			if b.tokens < 1 {
				b.mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
				return
			}

			b.tokens--
			b.mu.Unlock()

			next(w, r)
		}
	}
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := net.ParseIP(fwd)
		if parts != nil {
			return parts.String()
		}
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func cleanupBuckets(buckets *sync.Map, interval time.Duration) {
	for {
		time.Sleep(interval)
		cutoff := time.Now().Add(-interval)
		buckets.Range(func(key, value any) bool {
			b := value.(*bucket)
			b.mu.Lock()
			stale := b.lastTime.Before(cutoff)
			b.mu.Unlock()
			if stale {
				buckets.Delete(key)
			}
			return true
		})
	}
}

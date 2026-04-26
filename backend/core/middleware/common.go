package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-site")
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

var (
	requestTotal      uint64
	request2xxTotal   uint64
	request4xxTotal   uint64
	request5xxTotal   uint64
	requestDurationMs uint64
)

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
}

func AccessLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}

		entry := map[string]any{
			"ts":         time.Now().UTC().Format(time.RFC3339),
			"request_id": chiMiddleware.GetReqID(r.Context()),
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     rec.status,
			"bytes":      rec.bytes,
			"durationMs": time.Since(start).Milliseconds(),
			"ip":         clientIP(r),
			"userAgent":  r.UserAgent(),
		}
		_ = json.NewEncoder(os.Stdout).Encode(entry)

		atomic.AddUint64(&requestTotal, 1)
		atomic.AddUint64(&requestDurationMs, uint64(time.Since(start).Milliseconds()))
		switch {
		case rec.status >= 500:
			atomic.AddUint64(&request5xxTotal, 1)
		case rec.status >= 400:
			atomic.AddUint64(&request4xxTotal, 1)
		case rec.status >= 200:
			atomic.AddUint64(&request2xxTotal, 1)
		}
	})
}

func MetricsSnapshot() map[string]any {
	total := atomic.LoadUint64(&requestTotal)
	totalDuration := atomic.LoadUint64(&requestDurationMs)
	avgDuration := float64(0)
	if total > 0 {
		avgDuration = float64(totalDuration) / float64(total)
	}
	return map[string]any{
		"requests_total":          total,
		"requests_2xx_total":      atomic.LoadUint64(&request2xxTotal),
		"requests_4xx_total":      atomic.LoadUint64(&request4xxTotal),
		"requests_5xx_total":      atomic.LoadUint64(&request5xxTotal),
		"request_duration_ms_sum": totalDuration,
		"request_duration_ms_avg": avgDuration,
	}
}

func clientIP(r *http.Request) string {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return ip
	}
	return strings.TrimSpace(r.RemoteAddr)
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

func IPFixedWindowRateLimit(reqPerMinute int) func(http.Handler) http.Handler {
	if reqPerMinute <= 0 {
		reqPerMinute = 120
	}
	var (
		mu           sync.Mutex
		buckets      = map[string]*bucket{}
		capacity     = float64(reqPerMinute)
		refillPerSec = capacity / 60.0
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			now := time.Now()

			mu.Lock()
			b := buckets[ip]
			if b == nil {
				b = &bucket{tokens: capacity, lastRefill: now}
				buckets[ip] = b
			}
			elapsed := now.Sub(b.lastRefill).Seconds()
			b.tokens += elapsed * refillPerSec
			if b.tokens > capacity {
				b.tokens = capacity
			}
			b.lastRefill = now
			if b.tokens < 1 {
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}
			b.tokens -= 1
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

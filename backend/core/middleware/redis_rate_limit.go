package middleware

import (
	"fmt"
	"net/http"
	"time"

	coreRedis "github.com/Moeed-ul-Hassan/chatapp/core/redis"
)

// IPRateLimitPreferRedis uses Redis when available and falls back to in-memory limiter.
func IPRateLimitPreferRedis(reqPerMinute int) func(http.Handler) http.Handler {
	fallback := IPFixedWindowRateLimit(reqPerMinute)
	if reqPerMinute <= 0 {
		reqPerMinute = 120
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !coreRedis.IsReady() || coreRedis.Client == nil {
				fallback(next).ServeHTTP(w, r)
				return
			}
			ip := clientIP(r)
			now := time.Now().UTC()
			window := now.Format("200601021504") // per-minute bucket
			key := fmt.Sprintf("ratelimit:%s:%s", ip, window)

			count, err := coreRedis.Client.Incr(r.Context(), key).Result()
			if err != nil {
				fallback(next).ServeHTTP(w, r)
				return
			}
			_ = coreRedis.Client.Expire(r.Context(), key, time.Minute+5*time.Second).Err()
			if count > int64(reqPerMinute) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

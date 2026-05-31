package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
)

// RateLimiter defines the contract for our rate limiting mechanism,
// allowing us to easily swap Redis for an in-memory or different provider.
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, windowSec int) (bool, error)
}

// RateLimitMiddleware enforces rate limits on incoming proxy requests.
func RateLimitMiddleware(logger *slog.Logger, limiter RateLimiter, limit int, windowSec int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// In our architecture, rate limits are scoped per Project AND per Client IP.
			projectID := r.Header.Get("X-Aegis-Project-Id")
			if projectID == "" {
				projectID = "global"
			}

			// Extract the real client IP (ignoring port)
			// ip, _, err := net.SplitHostPort(r.RemoteAddr)
			// if err != nil {
			// 	ip = r.RemoteAddr
			// }
			ip := r.Header.Get("X-Forwarded-For")

			if ip != "" {
				// sometimes multiple IPs come: "client, proxy1, proxy2"
				ip = strings.Split(ip, ",")[0]
			} else {
				var err error
				ip, _, err = net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					ip = r.RemoteAddr
				}
			}

			// The unique bucket key: project_id:ip
			key := projectID + ":" + ip

			// Check against the rate limiter
			allowed, err := limiter.Allow(r.Context(), key, limit, windowSec)
			if err != nil {
				// We "fail open" on Redis errors so a cache outage doesn't take down the firewall.
				// But we log it as a severe error.
				logger.Error("Rate limiter backend failed (Failing Open)", "error", err, "key", key)
			} else if !allowed {
				logger.Warn("Rate limit exceeded", "ip", ip, "project_id", projectID)

				w.Header().Set("Retry-After", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Too Many Requests", "message": "Rate limit exceeded. Please try again later."}`))
				return
			}

			// Request allowed, pass to the next handler (the proxy)
			next.ServeHTTP(w, r)
		})
	}
}

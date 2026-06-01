package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aegis/firewall/internal/queue"
)

// statusRecorder is a lightweight wrapper that captures the HTTP status code
// returned by the downstream handlers, allowing us to log whether the request
// succeeded (200), was blocked by the WAF/AI (403), or hit a Rate Limit (429).
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Flush safely calls the underlying Flusher if it exists, fixing streaming compatibility.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// AnalyticsMiddleware intercepts every request, measures total latency, captures the
// response status, and asynchronously fires a log event to NATS.
func AnalyticsMiddleware(logger *slog.Logger, publisher queue.EventPublisher) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			recorder := &statusRecorder{
				ResponseWriter: w,
				status:         http.StatusOK, // Default to 200 if WriteHeader isn't called
			}

			// Execute the rest of the proxy pipeline
			next.ServeHTTP(recorder, r)

			// The request is now complete!
			latency := time.Since(start).Milliseconds()

			// Extract Project ID passed UP by RouteContextMiddleware via Response Header
			projectID := recorder.Header().Get("X-Aegis-Project-Id")
			if projectID == "" {
				projectID = "global" // Useful for distinguishing ping/health traffic or 401s
			} else {
				// Clean up the header so it doesn't leak to the external client
				recorder.Header().Del("X-Aegis-Project-Id")
			}

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

			// We can infer if the request was blocked by our security layers based on the status code
			blocked := recorder.status == http.StatusForbidden || recorder.status == http.StatusTooManyRequests

			reason := ""
			if blocked {
				if recorder.status == http.StatusTooManyRequests {
					reason = "Rate Limit Exceeded"
				} else {
					reason = "Security Policy Blocked (WAF / AI / DLP)"
				}
			}

			event := queue.LogEvent{
				Timestamp:   time.Now().UTC(),
				ProjectID:   projectID,
				ClientIP:    ip,
				Method:      r.Method,
				Path:        r.URL.Path,
				StatusCode:  recorder.status,
				LatencyMs:   latency,
				Blocked:     blocked,
				BlockReason: reason,
			}

			// Fire and forget: send the log to NATS asynchronously.
			// This adds practically zero latency to the user's web request.
			// err = publisher.Publish("analytics.logs", event)
			// if err != nil {
			// 	logger.Error("Failed to publish analytics event to NATS", "error", err)
			// }
			if err := publisher.Publish("analytics.logs", event); err != nil {
				logger.Error("Failed to publish analytics event to NATS", "error", err)
			}
		})
	}
}

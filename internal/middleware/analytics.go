package middleware

import (
	"log/slog"
	"net"
	"net/http"
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

			projectID := r.Header.Get("X-Aegis-Project-Id")
			if projectID == "" {
				projectID = "global" // Useful for distinguishing ping/health traffic
			}

			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
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
			err = publisher.Publish("analytics.logs", event)
			if err != nil {
				logger.Error("Failed to publish analytics event to NATS", "error", err)
			}
		})
	}
}

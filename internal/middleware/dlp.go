package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
)

var (
	// ccRegex matches generic 13-16 digit credit card formats
	ccRegex = regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`)
	
	// ssnRegex matches standard US Social Security Numbers
	ssnRegex = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
)

// responseRecorder implements http.ResponseWriter completely independently.
// We do NOT embed http.ResponseWriter because we want to prevent the ReverseProxy 
// from accessing the original socket's Header() or Flush() methods prematurely!
type responseRecorder struct {
	header http.Header
	status int
	body   *bytes.Buffer
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	// Buffer the response in memory
	return r.body.Write(b)
}

// DLPMiddleware intercepts the response from the upstream API before it reaches the client,
// scanning it for sensitive data leaks (PII/Credit Cards).
func DLPMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			
			// Setup interceptor
			recorder := &responseRecorder{
				header: make(http.Header),
				status: http.StatusOK, // default if WriteHeader isn't called
				body:   &bytes.Buffer{},
			}

			// Pass the recorder down the chain to the ReverseProxy
			next.ServeHTTP(recorder, r)

			// The proxy has finished writing. Read the buffered response.
			responseBody := recorder.body.Bytes()

			// Scan for Data Leaks
			leakDetected := false
			var leakReason string

			if ccRegex.Match(responseBody) {
				leakDetected = true
				leakReason = "Credit Card Data Exposure Detected"
			} else if ssnRegex.Match(responseBody) {
				leakDetected = true
				leakReason = "Social Security Number Exposure Detected"
			}

			// Block the leak!
			if leakDetected {
				logger.Warn("Data Leak Prevented (Egress Blocked)", "ip", r.RemoteAddr, "reason", leakReason)
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				
				resp := map[string]string{
					"error":   "Data Leak Prevention",
					"message": "Response blocked due to sensitive data exposure",
					"reason":  leakReason,
				}
				json.NewEncoder(w).Encode(resp)
				return
			}

			// Safe! Flush original headers and status to the real client
			for k, v := range recorder.Header() {
				w.Header()[k] = v
			}
			w.WriteHeader(recorder.status)
			w.Write(responseBody)
		})
	}
}

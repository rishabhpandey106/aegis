package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"regexp"
)

var (
	// sqliRegex is a basic pattern to detect SQL Injection attempts
	sqliRegex = regexp.MustCompile(`(?i)(UNION.*SELECT|SELECT.*FROM|INSERT.*INTO|UPDATE.*SET|DELETE.*FROM|DROP.*TABLE|--|\bOR\b.*=|\bAND\b.*=)`)
	
	// xssRegex is a basic pattern to detect Cross-Site Scripting (XSS)
	xssRegex  = regexp.MustCompile(`(?i)(<script>|javascript:|onerror=|onload=|eval\()`)
)

// WAFMiddleware intercepts HTTP requests to check for deterministic attacks
// like SQL Injection and XSS using regular expressions.
func WAFMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Check the URL Path and Query Parameters
			if sqliRegex.MatchString(r.URL.RawQuery) || sqliRegex.MatchString(r.URL.Path) {
				blockWAF(w, logger, r, "SQL Injection Detected in URL")
				return
			}
			if xssRegex.MatchString(r.URL.RawQuery) || xssRegex.MatchString(r.URL.Path) {
				blockWAF(w, logger, r, "XSS Detected in URL")
				return
			}

			// 2. Check the Request Body
			if r.Body != nil {
				bodyBytes, _ := io.ReadAll(r.Body)
				
				// Restore body for downstream middlewares
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				if len(bodyBytes) > 0 {
					if sqliRegex.MatchString(string(bodyBytes)) {
						blockWAF(w, logger, r, "SQL Injection Detected in Request Body")
						return
					}
					if xssRegex.MatchString(string(bodyBytes)) {
						blockWAF(w, logger, r, "XSS Detected in Request Body")
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func blockWAF(w http.ResponseWriter, logger *slog.Logger, r *http.Request, reason string) {
	logger.Warn("WAF Blocked Request", "ip", r.RemoteAddr, "reason", reason, "path", r.URL.Path)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	
	resp := map[string]string{
		"error":   "Forbidden",
		"message": "Blocked by Web Application Firewall (WAF)",
		"reason":  reason,
	}
	json.NewEncoder(w).Encode(resp)
}

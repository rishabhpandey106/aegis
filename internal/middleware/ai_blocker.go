package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	pb "github.com/aegis/firewall/internal/proto"
	"github.com/aegis/firewall/internal/proxy"
)

// AIAnalyzer defines the contract for sending requests to the AI engine.
// Using an interface here allows us to mock the gRPC client in unit tests!
type AIAnalyzer interface {
	AnalyzeRequest(ctx context.Context, req *pb.AnalyzeRequestMessage) (*pb.AnalyzeResponseMessage, error)
}

// AIBlockerMiddleware intercepts HTTP requests, reads their bodies, and sends them
// to the external AI Engine for real-time prompt injection detection.
func AIBlockerMiddleware(logger *slog.Logger, analyzer AIAnalyzer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Optimization: We only check for prompt injection on methods that carry payloads.
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// 1. Read the HTTP body into memory
			var bodyBytes []byte
			if r.Body != nil {
				bodyBytes, _ = io.ReadAll(r.Body)

				// CRITICAL: Once an HTTP body is read, it's consumed!
				// We MUST restore it with a NopCloser so the core reverse proxy can still read it
				// to forward it to the upstream API.
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			if len(bodyBytes) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			route, ok := r.Context().Value(proxy.RouteConfigKey).(*proxy.RouteConfig)
			if !ok || route == nil {
				http.Error(w, "Internal configuration error", http.StatusInternalServerError)
				return
			}
			projectID := route.ProjectID

			// AI Blocker is EXPENSIVE and must be strictly OPT-IN.
			// It only runs if the rule exists AND is explicitly enabled.
			aiEnabled := false
			if rawConfig, exists := route.SecurityRules["ai_blocker"]; exists {
				var customConf struct {
					Enabled *bool `json:"enabled"`
				}
				if err := json.Unmarshal(rawConfig, &customConf); err == nil && customConf.Enabled != nil && *customConf.Enabled {
					aiEnabled = true
				}
			}

			if !aiEnabled {
				next.ServeHTTP(w, r)
				return
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

			// Flatten HTTP headers to pass to gRPC
			headers := make(map[string]string)
			for k, v := range r.Header {
				if len(v) > 0 {
					headers[k] = v[0]
				}
			}

			reqMsg := &pb.AnalyzeRequestMessage{
				ProjectId: projectID,
				ClientIp:  ip,
				Method:    r.Method,
				Path:      r.URL.Path,
				Headers:   headers,
				Body:      string(bodyBytes),
			}

			// 2. Call the AI Engine Synchronously
			resp, err := analyzer.AnalyzeRequest(r.Context(), reqMsg)
			if err != nil {
				// FAIL OPEN DESIGN: If the AI Engine is offline, or timeouts after 2s,
				// we log an error but ALLOW the request to proceed.
				// An AI outage should never cause a total system API outage!
				logger.Error("AI Engine gRPC call failed (Failing Open)", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			// 3. Act on the AI Verdict
			if resp.BlockRecommended {
				logger.Warn("Request BLOCKED by AI Security Engine",
					"ip", ip,
					"project_id", projectID,
					"risk_score", resp.RiskScore,
					"reason", resp.Reason,
					"flags", resp.Flags,
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)

				// Return a safe, sanitized block message to the client
				respMap := map[string]interface{}{
					"error":   "Forbidden",
					"message": "Request blocked by Aegis AI Security Policy",
					"reason":  resp.Reason,
				}
				json.NewEncoder(w).Encode(respMap)
				return
			}

			// AI says it's safe! Pass it to the core proxy.
			next.ServeHTTP(w, r)
		})
	}
}

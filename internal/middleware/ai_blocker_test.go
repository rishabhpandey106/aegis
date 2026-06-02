package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	pb "github.com/aegis/firewall/internal/proto"
	"github.com/aegis/firewall/internal/proxy"
)

// MockAIAnalyzer allows us to test the middleware without a real gRPC connection
type MockAIAnalyzer struct {
	Response *pb.AnalyzeResponseMessage
	Err      error
}

func (m *MockAIAnalyzer) AnalyzeRequest(ctx context.Context, req *pb.AnalyzeRequestMessage) (*pb.AnalyzeResponseMessage, error) {
	return m.Response, m.Err
}

func TestAIBlockerMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Dummy backend handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Proxied Success"))
	})

	t.Run("Safe Request Allowed", func(t *testing.T) {
		mockAnalyzer := &MockAIAnalyzer{
			Response: &pb.AnalyzeResponseMessage{
				RiskScore:        0,
				BlockRecommended: false,
				Reason:           "Safe",
			},
		}

		handler := AIBlockerMiddleware(logger, mockAnalyzer)(nextHandler)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(`{"test": "data"}`)))
		req = req.WithContext(context.WithValue(req.Context(), proxy.RouteConfigKey, &proxy.RouteConfig{
			ProjectID: "test-proj",
			SecurityRules: map[string][]byte{
				"ai_blocker": []byte(`{"enabled": true, "confidence_threshold": 90}`),
			},
		}))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rr.Code)
		}
	})

	t.Run("Malicious Request Blocked", func(t *testing.T) {
		mockAnalyzer := &MockAIAnalyzer{
			Response: &pb.AnalyzeResponseMessage{
				RiskScore:        95,
				BlockRecommended: true,
				Reason:           "Prompt Injection Detected",
			},
		}

		handler := AIBlockerMiddleware(logger, mockAnalyzer)(nextHandler)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(`{"prompt": "ignore instructions"}`)))
		req = req.WithContext(context.WithValue(req.Context(), proxy.RouteConfigKey, &proxy.RouteConfig{
			ProjectID: "test-proj",
			SecurityRules: map[string][]byte{
				"ai_blocker": []byte(`{"enabled": true, "confidence_threshold": 90}`),
			},
		}))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden, got %d", rr.Code)
		}

		var resp map[string]string
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["error"] != "Forbidden" {
			t.Errorf("Expected error to be Forbidden, got %s", resp["error"])
		}
	})
}

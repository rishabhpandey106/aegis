package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"context"

	"github.com/aegis/firewall/internal/proxy"
	"github.com/aegis/firewall/internal/queue"
)

// MockPublisher allows us to test the analytics middleware without a real NATS server
type MockPublisher struct {
	Events []queue.LogEvent
}

func (m *MockPublisher) Publish(subject string, event queue.LogEvent) error {
	m.Events = append(m.Events, event)
	return nil
}

func (m *MockPublisher) Close() {}

func TestAnalyticsMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockPublisher := &MockPublisher{}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Aegis-Project-Id", "test-project-123")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Created OK"))
	})

	handler := AnalyticsMiddleware(logger, mockPublisher)(nextHandler)

	req := httptest.NewRequest("POST", "/api/data", nil)
	req = req.WithContext(context.WithValue(req.Context(), proxy.RouteConfigKey, &proxy.RouteConfig{ProjectID: "test-project-123"}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Validate the HTTP response
	if rr.Code != http.StatusCreated {
		t.Errorf("Expected 201 Created, got %d", rr.Code)
	}

	// Validate the NATS event was published
	if len(mockPublisher.Events) != 1 {
		t.Fatalf("Expected 1 event published, got %d", len(mockPublisher.Events))
	}

	event := mockPublisher.Events[0]
	if event.ProjectID != "test-project-123" {
		t.Errorf("Expected ProjectID 'test-project-123', got '%s'", event.ProjectID)
	}
	if event.StatusCode != http.StatusCreated {
		t.Errorf("Expected StatusCode 201, got %d", event.StatusCode)
	}
	if event.Blocked {
		t.Errorf("Expected Blocked to be false")
	}
}

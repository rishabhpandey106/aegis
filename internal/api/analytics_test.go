package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aegis/firewall/internal/db"
	"github.com/aegis/firewall/internal/queue"
)

// MockAnalyticsRepo implements db.LogsRepository for testing
type MockAnalyticsRepo struct {
	Stats db.ProjectStats
	Err   error
}

func (m *MockAnalyticsRepo) SaveLog(event queue.LogEvent) error {
	return nil
}

func (m *MockAnalyticsRepo) GetProjectAnalytics(projectID string) (db.ProjectStats, error) {
	return m.Stats, m.Err
}

func (m *MockAnalyticsRepo) GetOrgAnalytics(orgID string) (db.ProjectStats, error) {
	return m.Stats, m.Err
}

func TestAnalyticsHandler_GetProjectAnalytics(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	mockRepo := &MockAnalyticsRepo{
		Stats: db.ProjectStats{
			TotalRequests:   100,
			BlockedRequests: 5,
			AvgLatencyMs:    45.5,
			RecentBlocks: []db.BlockedEvent{
				{Timestamp: "2023-01-01T12:00:00Z", ClientIP: "1.1.1.1", BlockReason: "WAF"},
			},
		},
	}

	handler := NewAnalyticsHandler(logger, mockRepo)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/proj-123/analytics", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserRoleKey, "admin"))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rr.Code)
	}

	var stats db.ProjectStats
	if err := json.NewDecoder(rr.Body).Decode(&stats); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if stats.TotalRequests != 100 {
		t.Errorf("Expected 100 total requests, got %d", stats.TotalRequests)
	}
	if len(stats.RecentBlocks) != 1 {
		t.Errorf("Expected 1 recent block, got %d", len(stats.RecentBlocks))
	}
}

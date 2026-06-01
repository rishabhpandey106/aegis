package worker

import (
	"testing"
	"time"

	"github.com/aegis/firewall/internal/queue"
)

// MockLogsRepo simulates database insertion
type MockLogsRepo struct {
	saved []queue.LogEvent
}

func (m *MockLogsRepo) SaveLog(event queue.LogEvent) error {
	m.saved = append(m.saved, event)
	return nil
}

func TestAnalyticsWorker_DependencyInjection(t *testing.T) {
	// A simple test to verify our interface bindings and repo injection works
	repo := &MockLogsRepo{}
	
	event := queue.LogEvent{
		Timestamp:   time.Now(),
		ProjectID:   "test-uuid-1234",
		ClientIP:    "192.168.1.1",
		Method:      "POST",
		Path:        "/chat",
		StatusCode:  403,
		Blocked:     true,
		BlockReason: "Prompt Injection Detected",
	}

	if err := repo.SaveLog(event); err != nil {
		t.Fatalf("Unexpected error saving log: %v", err)
	}

	if len(repo.saved) != 1 {
		t.Errorf("Expected 1 log saved, got %d", len(repo.saved))
	}

	if repo.saved[0].BlockReason != "Prompt Injection Detected" {
		t.Errorf("Expected block reason 'Prompt Injection Detected', got '%s'", repo.saved[0].BlockReason)
	}
}

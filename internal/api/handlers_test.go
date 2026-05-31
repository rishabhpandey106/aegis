package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aegis/firewall/internal/models"
)

// MockRepo implements models.ProjectRepository for clean, isolated testing.
type MockRepo struct {
	projects map[string]*models.Project
}

func (m *MockRepo) Create(ctx context.Context, p *models.Project) error {
	p.ID = "test-uuid"
	m.projects[p.ID] = p
	return nil
}

func (m *MockRepo) GetByID(ctx context.Context, id string) (*models.Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return nil, errors.New("project not found")
	}
	return p, nil
}

func (m *MockRepo) ListByOrg(ctx context.Context, orgID string) ([]*models.Project, error) {
	return nil, nil // Not tested in this suite for brevity
}

func TestHandleCreateProject(t *testing.T) {
	repo := &MockRepo{projects: make(map[string]*models.Project)}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewProjectHandler(logger, repo)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	payload := []byte(`{"org_id":"00000000-0000-0000-0000-000000000001","name":"Test API","upstream_url":"https://api.test.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var p models.Project
	if err := json.NewDecoder(rr.Body).Decode(&p); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if p.ID != "test-uuid" || p.Name != "Test API" {
		t.Errorf("Unexpected project in response: %+v", p)
	}
}

func TestHandleGetProject(t *testing.T) {
	repo := &MockRepo{
		projects: map[string]*models.Project{
			"uuid-123": {
				ID:          "uuid-123",
				Name:        "Existing API",
				UpstreamURL: "https://existing.com",
			},
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewProjectHandler(logger, repo)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/uuid-123", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var p models.Project
	json.NewDecoder(rr.Body).Decode(&p)
	if p.Name != "Existing API" {
		t.Errorf("Expected name 'Existing API', got %s", p.Name)
	}
}

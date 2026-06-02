package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aegis/firewall/internal/models"
)

type MockOrgRepo struct {
	orgs map[string]*models.Organization
}

// Update implements [models.OrgRepository].
func (m *MockOrgRepo) Update(org *models.Organization) error {
	panic("unimplemented")
}

func (m *MockOrgRepo) Create(org *models.Organization) error {
	org.ID = "org-uuid"
	m.orgs[org.ID] = org
	return nil
}

func (m *MockOrgRepo) GetByID(id string) (*models.Organization, error) {
	org, ok := m.orgs[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return org, nil
}

func TestHandleCreateOrg(t *testing.T) {
	repo := &MockOrgRepo{orgs: make(map[string]*models.Organization)}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewOrgHandler(logger, repo)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	payload := []byte(`{"name":"Aegis Security Corp", "plan":"enterprise"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rr.Code)
	}

	var org models.Organization
	json.NewDecoder(rr.Body).Decode(&org)
	if org.ID != "org-uuid" || org.Name != "Aegis Security Corp" {
		t.Errorf("Unexpected response: %+v", org)
	}
}

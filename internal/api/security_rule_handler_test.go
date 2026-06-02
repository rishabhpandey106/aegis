package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aegis/firewall/internal/models"
)

type MockRuleRepo struct {
	rules map[string]*models.SecurityRule
}

func (m *MockRuleRepo) Create(rule *models.SecurityRule) error {
	rule.ID = "rule-uuid"
	m.rules[rule.ID] = rule
	return nil
}

func (m *MockRuleRepo) GetByProjectID(projectID string) ([]*models.SecurityRule, error) {
	var list []*models.SecurityRule
	for _, r := range m.rules {
		if r.ProjectID == projectID {
			list = append(list, r)
		}
	}
	return list, nil
}

func (m *MockRuleRepo) Delete(id string) error {
	delete(m.rules, id)
	return nil
}

func TestHandleCreateSecurityRule(t *testing.T) {
	repo := &MockRuleRepo{rules: make(map[string]*models.SecurityRule)}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewSecurityRuleHandler(logger, repo)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	payload := []byte(`{"rule_type":"rate_limit", "configuration":{"limit":5}, "action":"block"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-1/rules", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), UserRoleKey, "admin"))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rr.Code)
	}

	var r models.SecurityRule
	json.NewDecoder(rr.Body).Decode(&r)
	if r.ProjectID != "proj-1" || r.RuleType != "rate_limit" {
		t.Errorf("Unexpected rule: %+v", r)
	}
}

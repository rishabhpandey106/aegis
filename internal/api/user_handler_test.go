package api

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aegis/firewall/internal/models"
)

type MockUserRepo struct {
	users map[string]*models.User
}

func (m *MockUserRepo) Create(user *models.User) error {
	user.ID = "user-uuid"
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepo) GetByID(id string) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *MockUserRepo) ListByOrg(orgID string) ([]*models.User, error) {
	var list []*models.User
	for _, u := range m.users {
		if u.OrgID == orgID {
			list = append(list, u)
		}
	}
	return list, nil
}

func TestHandleCreateUser(t *testing.T) {
	repo := &MockUserRepo{users: make(map[string]*models.User)}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewUserHandler(logger, repo)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	payload := []byte(`{"org_id":"org-1", "email":"admin@test.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(payload))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rr.Code)
	}
}

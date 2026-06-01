package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aegis/firewall/internal/models"
)

type UserHandler struct {
	repo   models.UserRepository
	logger *slog.Logger
}

func NewUserHandler(logger *slog.Logger, repo models.UserRepository) *UserHandler {
	return &UserHandler{repo: repo, logger: logger}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/users", h.handleCreateUser)
	mux.HandleFunc("GET /api/v1/organizations/{org_id}/users", h.handleListUsers)
}

func (h *UserHandler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if user.Email == "" || user.OrgID == "" {
		http.Error(w, "Email and OrgID are required", http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(&user); err != nil {
		h.logger.Error("Failed to create user", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("org_id")
	users, err := h.repo.ListByOrg(orgID)
	if err != nil {
		h.logger.Error("Failed to list users", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if users == nil {
		users = make([]*models.User, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

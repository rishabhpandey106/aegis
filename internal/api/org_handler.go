package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aegis/firewall/internal/models"
)

type OrgHandler struct {
	repo   models.OrgRepository
	logger *slog.Logger
}

func NewOrgHandler(logger *slog.Logger, repo models.OrgRepository) *OrgHandler {
	return &OrgHandler{repo: repo, logger: logger}
}

func (h *OrgHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/organizations", h.handleCreateOrg)
	mux.HandleFunc("GET /api/v1/organizations/{id}", h.handleGetOrg)
}

func (h *OrgHandler) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	var org models.Organization
	if err := json.NewDecoder(r.Body).Decode(&org); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if org.Name == "" {
		http.Error(w, "Organization name is required", http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(&org); err != nil {
		h.logger.Error("Failed to create organization", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(org)
}

func (h *OrgHandler) handleGetOrg(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	org, err := h.repo.GetByID(id)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(org)
}

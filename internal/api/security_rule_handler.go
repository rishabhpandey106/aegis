package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aegis/firewall/internal/models"
)

type SecurityRuleHandler struct {
	repo   models.SecurityRuleRepository
	logger *slog.Logger
}

func NewSecurityRuleHandler(logger *slog.Logger, repo models.SecurityRuleRepository) *SecurityRuleHandler {
	return &SecurityRuleHandler{repo: repo, logger: logger}
}

func (h *SecurityRuleHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/projects/{id}/rules", h.handleCreateRule)
	mux.HandleFunc("GET /api/v1/projects/{id}/rules", h.handleGetRules)
}

func (h *SecurityRuleHandler) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, "Project ID required", http.StatusBadRequest)
		return
	}

	var rule models.SecurityRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	rule.ProjectID = projectID

	if err := h.repo.Create(&rule); err != nil {
		h.logger.Error("Failed to create security rule", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

func (h *SecurityRuleHandler) handleGetRules(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	rules, err := h.repo.GetByProjectID(projectID)
	if err != nil {
		h.logger.Error("Failed to fetch security rules", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Always return an array, even if empty
	if rules == nil {
		rules = make([]*models.SecurityRule, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

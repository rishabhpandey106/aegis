package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aegis/firewall/internal/db"
)

// AnalyticsHandler handles REST API requests for dashboard analytics.
type AnalyticsHandler struct {
	logger *slog.Logger
	repo   db.LogsRepository
}

// NewAnalyticsHandler uses dependency injection to set up the handler.
func NewAnalyticsHandler(logger *slog.Logger, repo db.LogsRepository) *AnalyticsHandler {
	return &AnalyticsHandler{
		logger: logger,
		repo:   repo,
	}
}

// RegisterRoutes registers the analytics routes onto the provided mux.
func (h *AnalyticsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/projects/{id}/analytics", RequireRole("admin", "viewer")(http.HandlerFunc(h.handleGetProjectAnalytics)))
	mux.Handle("GET /api/v1/organizations/{id}/analytics", RequireRole("admin", "viewer")(http.HandlerFunc(h.handleGetOrgAnalytics)))
}

// handleGetProjectAnalytics fetches aggregated time-series and security events for a specific project.
func (h *AnalyticsHandler) handleGetProjectAnalytics(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		http.Error(w, "Project ID is required", http.StatusBadRequest)
		return
	}

	stats, err := h.repo.GetProjectAnalytics(projectID)
	if err != nil {
		h.logger.Error("Failed to fetch project analytics", "project_id", projectID, "error", err)
		http.Error(w, "Failed to retrieve analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		h.logger.Error("Failed to encode analytics response", "error", err)
	}
}

// handleGetOrgAnalytics fetches aggregated analytics across all projects in an organization.
func (h *AnalyticsHandler) handleGetOrgAnalytics(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		http.Error(w, "Organization ID is required", http.StatusBadRequest)
		return
	}

	stats, err := h.repo.GetOrgAnalytics(orgID)
	if err != nil {
		h.logger.Error("Failed to fetch organization analytics", "org_id", orgID, "error", err)
		http.Error(w, "Failed to retrieve analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		h.logger.Error("Failed to encode org analytics response", "error", err)
	}
}

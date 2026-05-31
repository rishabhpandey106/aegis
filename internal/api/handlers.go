package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aegis/firewall/internal/models"
)

// ProjectHandler handles HTTP requests for Project resources.
type ProjectHandler struct {
	repo   models.ProjectRepository
	logger *slog.Logger
}

// NewProjectHandler uses Dependency Injection to wire up the handler 
// with its required repository and logger.
func NewProjectHandler(logger *slog.Logger, repo models.ProjectRepository) *ProjectHandler {
	return &ProjectHandler{
		repo:   repo,
		logger: logger,
	}
}

// RegisterRoutes binds the handler methods to the given mux router
// utilizing Go 1.22's native enhanced routing features.
func (h *ProjectHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/projects", h.handleCreateProject)
	mux.HandleFunc("GET /api/v1/projects/{id}", h.handleGetProject)
}

func (h *ProjectHandler) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req models.Project
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode project creation request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.OrgID == "" || req.Name == "" || req.UpstreamURL == "" {
		http.Error(w, "Missing required fields (org_id, name, upstream_url)", http.StatusBadRequest)
		return
	}

	// Default values
	req.IsActive = true

	if err := h.repo.Create(r.Context(), &req); err != nil {
		h.logger.Error("Failed to save project to database", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

func (h *ProjectHandler) handleGetProject(w http.ResponseWriter, r *http.Request) {
	// PathValue is a new feature in Go 1.22 routing
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing project ID", http.StatusBadRequest)
		return
	}

	project, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if err.Error() == "project not found" {
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to fetch project", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

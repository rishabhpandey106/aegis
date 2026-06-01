package models

import (
	"context"
	"time"
)

// Project represents a proxy configuration endpoint in the firewall.
// It maps directly to the 'projects' table in the database.
type Project struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	Name        string    `json:"name"`
	UpstreamURL string    `json:"upstream_url"`
	APIKeyHash  *string   `json:"-"`                   // Omitted from JSON for security
	RawAPIKey   string    `json:"api_key,omitempty"`   // Only populated once upon creation
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectRepository defines the interface for project data access.
// This enforces clean architecture, allowing the API layer to be completely
// decoupled from the specific database implementation (Postgres, Mock, etc.).
type ProjectRepository interface {
	Create(ctx context.Context, p *Project) error
	GetByID(ctx context.Context, id string) (*Project, error)
	GetByAPIKeyHash(ctx context.Context, hash string) (*Project, error)
	ListByOrg(ctx context.Context, orgID string) ([]*Project, error)
}

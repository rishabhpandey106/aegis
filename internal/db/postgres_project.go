package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/aegis/firewall/internal/models"
)

// PostgresProjectRepo implements the models.ProjectRepository interface
// using the standard database/sql library and PostgreSQL.
type PostgresProjectRepo struct {
	db *sql.DB
}

// NewPostgresProjectRepo injects the database connection into the repository.
func NewPostgresProjectRepo(db *sql.DB) *PostgresProjectRepo {
	return &PostgresProjectRepo{db: db}
}

// Create inserts a new project into the database and populates the generated fields.
func (r *PostgresProjectRepo) Create(ctx context.Context, p *models.Project) error {
	query := `
		INSERT INTO projects (org_id, name, upstream_url, api_key_hash, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query, p.OrgID, p.Name, p.UpstreamURL, p.APIKeyHash, p.IsActive).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	
	if err != nil {
		return err
	}
	return nil
}

// GetByID fetches a single project by its UUID.
func (r *PostgresProjectRepo) GetByID(ctx context.Context, id string) (*models.Project, error) {
	query := `
		SELECT id, org_id, name, upstream_url, is_active, created_at, updated_at
		FROM projects
		WHERE id = $1
	`
	p := &models.Project{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.OrgID, &p.Name, &p.UpstreamURL, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("project not found")
	}
	return p, err
}

// GetByAPIKeyHash fetches a project using its hashed API key.
func (r *PostgresProjectRepo) GetByAPIKeyHash(ctx context.Context, hash string) (*models.Project, error) {
	query := `
		SELECT id, org_id, name, upstream_url, api_key_hash, is_active, created_at, updated_at
		FROM projects
		WHERE api_key_hash = $1
	`
	p := &models.Project{}
	err := r.db.QueryRowContext(ctx, query, hash).Scan(
		&p.ID, &p.OrgID, &p.Name, &p.UpstreamURL, &p.APIKeyHash, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("project not found")
	}
	return p, err
}

// ListByOrg fetches all projects associated with a specific organization UUID.
func (r *PostgresProjectRepo) ListByOrg(ctx context.Context, orgID string) ([]*models.Project, error) {
	query := `
		SELECT id, org_id, name, upstream_url, is_active, created_at, updated_at
		FROM projects
		WHERE org_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		p := &models.Project{}
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.UpstreamURL, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

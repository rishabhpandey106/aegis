package db

import (
	"database/sql"

	"github.com/aegis/firewall/internal/models"
)

type PostgresOrgRepo struct {
	db *sql.DB
}

func NewPostgresOrgRepo(db *sql.DB) *PostgresOrgRepo {
	return &PostgresOrgRepo{db: db}
}

func (r *PostgresOrgRepo) Create(org *models.Organization) error {
	query := `
		INSERT INTO organizations (name, plan)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`
	plan := org.Plan
	if plan == "" {
		plan = "free"
	}
	return r.db.QueryRow(query, org.Name, plan).Scan(&org.ID, &org.CreatedAt, &org.UpdatedAt)
}

func (r *PostgresOrgRepo) GetByID(id string) (*models.Organization, error) {
	query := `SELECT id, name, plan, created_at, updated_at FROM organizations WHERE id = $1`
	var org models.Organization
	err := r.db.QueryRow(query, id).Scan(&org.ID, &org.Name, &org.Plan, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *PostgresOrgRepo) Update(org *models.Organization) error {
	query := `
		UPDATE organizations 
		SET name = $1, plan = $2, updated_at = CURRENT_TIMESTAMP 
		WHERE id = $3
		RETURNING updated_at
	`
	return r.db.QueryRow(query, org.Name, org.Plan, org.ID).Scan(&org.UpdatedAt)
}

package db

import (
	"database/sql"

	"github.com/aegis/firewall/internal/models"
)

type PostgresUserRepo struct {
	db *sql.DB
}

func NewPostgresUserRepo(db *sql.DB) *PostgresUserRepo {
	return &PostgresUserRepo{db: db}
}

func (r *PostgresUserRepo) Create(user *models.User) error {
	query := `
		INSERT INTO users (org_id, email, role)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	role := user.Role
	if role == "" {
		role = "viewer"
	}
	return r.db.QueryRow(query, user.OrgID, user.Email, role).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *PostgresUserRepo) GetByID(id string) (*models.User, error) {
	query := `SELECT id, org_id, email, role, created_at, updated_at FROM users WHERE id = $1`
	var user models.User
	err := r.db.QueryRow(query, id).Scan(&user.ID, &user.OrgID, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepo) ListByOrg(orgID string) ([]*models.User, error) {
	query := `SELECT id, org_id, email, role, created_at, updated_at FROM users WHERE org_id = $1`
	rows, err := r.db.Query(query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.OrgID, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}

package db

import (
	"database/sql"
	"encoding/json"

	"github.com/aegis/firewall/internal/models"
)

type PostgresSecurityRuleRepo struct {
	db *sql.DB
}

func NewPostgresSecurityRuleRepo(db *sql.DB) *PostgresSecurityRuleRepo {
	return &PostgresSecurityRuleRepo{db: db}
}

func (r *PostgresSecurityRuleRepo) Create(rule *models.SecurityRule) error {
	query := `
		INSERT INTO security_rules (project_id, rule_type, configuration, action)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(query, rule.ProjectID, rule.RuleType, string(rule.Configuration), rule.Action).
		Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

func (r *PostgresSecurityRuleRepo) GetByProjectID(projectID string) ([]*models.SecurityRule, error) {
	query := `
		SELECT id, project_id, rule_type, configuration, action, created_at, updated_at
		FROM security_rules
		WHERE project_id = $1
	`
	rows, err := r.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.SecurityRule
	for rows.Next() {
		var rule models.SecurityRule
		var configStr string
		if err := rows.Scan(&rule.ID, &rule.ProjectID, &rule.RuleType, &configStr, &rule.Action, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rule.Configuration = json.RawMessage(configStr)
		rules = append(rules, &rule)
	}
	return rules, nil
}

func (r *PostgresSecurityRuleRepo) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM security_rules WHERE id = $1", id)
	return err
}

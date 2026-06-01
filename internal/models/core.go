package models

import (
	"encoding/json"
	"time"
)

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SecurityRule struct {
	ID            string          `json:"id"`
	ProjectID     string          `json:"project_id"`
	RuleType      string          `json:"rule_type"`
	Configuration json.RawMessage `json:"configuration"`
	Action        string          `json:"action"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// SecurityRuleRepository defines CRUD for security rules
type SecurityRuleRepository interface {
	Create(rule *SecurityRule) error
	GetByProjectID(projectID string) ([]*SecurityRule, error)
	Delete(id string) error
}

// OrgRepository defines CRUD for organizations
type OrgRepository interface {
	Create(org *Organization) error
	GetByID(id string) (*Organization, error)
}

// UserRepository defines CRUD for users
type UserRepository interface {
	Create(user *User) error
	GetByID(id string) (*User, error)
	ListByOrg(orgID string) ([]*User, error)
}

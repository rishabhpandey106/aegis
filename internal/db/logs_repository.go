package db

import (
	"database/sql"
	"log/slog"

	"github.com/aegis/firewall/internal/queue"
)

// ProjectStats holds aggregated security and traffic analytics for a project
type ProjectStats struct {
	TotalRequests   int            `json:"total_requests"`
	BlockedRequests int            `json:"blocked_requests"`
	AvgLatencyMs    float64        `json:"avg_latency_ms"`
	RecentBlocks    []BlockedEvent `json:"recent_blocks"`
}

// BlockedEvent represents a recent blocked request
type BlockedEvent struct {
	Timestamp   string `json:"timestamp"`
	ClientIP    string `json:"client_ip"`
	BlockReason string `json:"block_reason"`
}

// LogsRepository handles persistence for analytics events.
type LogsRepository interface {
	SaveLog(event queue.LogEvent) error
	GetProjectAnalytics(projectID string) (ProjectStats, error)
	GetOrgAnalytics(orgID string) (ProjectStats, error)
}

type PostgresLogsRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewPostgresLogsRepository(db *sql.DB, logger *slog.Logger) *PostgresLogsRepository {
	return &PostgresLogsRepository{db: db, logger: logger}
}

func (r *PostgresLogsRepository) SaveLog(event queue.LogEvent) error {
	query := `
		INSERT INTO request_logs (timestamp, project_id, client_ip, method, path, status_code, latency_ms, blocked, block_reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(query,
		event.Timestamp,
		event.ProjectID,
		event.ClientIP,
		event.Method,
		event.Path,
		event.StatusCode,
		event.LatencyMs,
		event.Blocked,
		event.BlockReason,
	)
	return err
}

// GetProjectAnalytics aggregates traffic and security statistics for the dashboard
func (r *PostgresLogsRepository) GetProjectAnalytics(projectID string) (ProjectStats, error) {
	var stats ProjectStats

	// Calculate aggregates
	queryStats := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE blocked = true) as blocked,
			COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM request_logs
		WHERE project_id = $1
	`
	err := r.db.QueryRow(queryStats, projectID).Scan(&stats.TotalRequests, &stats.BlockedRequests, &stats.AvgLatencyMs)
	if err != nil {
		return stats, err
	}

	// Fetch recent blocked events
	queryBlocks := `
		SELECT timestamp, client_ip, block_reason 
		FROM request_logs 
		WHERE project_id = $1 AND blocked = true 
		ORDER BY timestamp DESC 
		LIMIT 10
	`
	rows, err := r.db.Query(queryBlocks, projectID)
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	stats.RecentBlocks = []BlockedEvent{}
	for rows.Next() {
		var event BlockedEvent
		var ts string
		if err := rows.Scan(&ts, &event.ClientIP, &event.BlockReason); err == nil {
			event.Timestamp = ts
			stats.RecentBlocks = append(stats.RecentBlocks, event)
		}
	}

	return stats, nil
}

// GetOrgAnalytics aggregates traffic and security statistics for all projects belonging to an organization.
func (r *PostgresLogsRepository) GetOrgAnalytics(orgID string) (ProjectStats, error) {
	var stats ProjectStats

	queryStats := `
		SELECT 
			COUNT(l.*) as total,
			COUNT(l.*) FILTER (WHERE l.blocked = true) as blocked,
			COALESCE(AVG(l.latency_ms), 0) as avg_latency
		FROM request_logs l
		JOIN projects p ON l.project_id ~* '^[0-9a-f-]{36}$' AND l.project_id::uuid = p.id
		WHERE p.org_id = $1::uuid
	`
	err := r.db.QueryRow(queryStats, orgID).Scan(&stats.TotalRequests, &stats.BlockedRequests, &stats.AvgLatencyMs)
	if err != nil {
		return stats, err
	}

	queryBlocks := `
		SELECT l.timestamp, l.client_ip, l.block_reason 
		FROM request_logs l
		JOIN projects p ON l.project_id ~* '^[0-9a-f-]{36}$' AND l.project_id::uuid = p.id
		WHERE p.org_id = $1::uuid AND l.blocked = true 
		ORDER BY l.timestamp DESC 
		LIMIT 10
	`
	rows, err := r.db.Query(queryBlocks, orgID)
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	stats.RecentBlocks = []BlockedEvent{}
	for rows.Next() {
		var event BlockedEvent
		var ts string
		if err := rows.Scan(&ts, &event.ClientIP, &event.BlockReason); err == nil {
			event.Timestamp = ts
			stats.RecentBlocks = append(stats.RecentBlocks, event)
		}
	}

	return stats, nil
}

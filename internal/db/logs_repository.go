package db

import (
	"database/sql"
	"log/slog"

	"github.com/aegis/firewall/internal/queue"
)

// LogsRepository handles persistence for analytics events.
type LogsRepository interface {
	SaveLog(event queue.LogEvent) error
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

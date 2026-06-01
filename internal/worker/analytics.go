package worker

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/aegis/firewall/internal/db"
	"github.com/aegis/firewall/internal/queue"
	"github.com/nats-io/nats.go"
)

// AnalyticsWorker consumes async logs from NATS and writes them to the DB.
type AnalyticsWorker struct {
	nc     *nats.Conn
	repo   db.LogsRepository
	logger *slog.Logger
	wg     sync.WaitGroup
	sub    *nats.Subscription
}

// NewAnalyticsWorker injects the required dependencies.
func NewAnalyticsWorker(nc *nats.Conn, repo db.LogsRepository, logger *slog.Logger) *AnalyticsWorker {
	return &AnalyticsWorker{
		nc:     nc,
		repo:   repo,
		logger: logger,
	}
}

// Start subscribes to the NATS queue and begins listening for logs in a background thread.
func (w *AnalyticsWorker) Start() error {
	w.logger.Info("Starting Analytics Worker... Listening on subject: analytics.logs")

	sub, err := w.nc.Subscribe("analytics.logs", func(m *nats.Msg) {
		w.wg.Add(1)
		defer w.wg.Done()

		var event queue.LogEvent
		if err := json.Unmarshal(m.Data, &event); err != nil {
			w.logger.Error("Failed to parse log event", "error", err)
			return
		}

		if err := w.repo.SaveLog(event); err != nil {
			w.logger.Error("Failed to save log to database", "error", err)
		} else {
			// Keeping as debug level to prevent log spam in production.
			w.logger.Debug("Log successfully saved to DB", "project_id", event.ProjectID)
		}
	})

	if err != nil {
		return err
	}

	w.sub = sub
	return nil
}

// Stop unsubscribes from the queue and waits for all active DB writes to complete safely.
func (w *AnalyticsWorker) Stop() {
	w.logger.Info("Gracefully stopping Analytics Worker...")
	if w.sub != nil {
		w.sub.Unsubscribe()
	}
	// Wait for any inflight database inserts to complete
	w.wg.Wait()
	w.logger.Info("Analytics Worker stopped cleanly.")
}

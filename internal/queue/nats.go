package queue

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// LogEvent represents an analytics log payload that gets published to NATS.
// It contains metadata about the request, the final status, and any security actions taken.
type LogEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	ProjectID   string    `json:"project_id"`
	ClientIP    string    `json:"client_ip"`
	Method      string    `json:"method"`
	Path        string    `json:"path"`
	StatusCode  int       `json:"status_code"`
	LatencyMs   int64     `json:"latency_ms"`
	Blocked     bool      `json:"blocked"`
	BlockReason string    `json:"block_reason"`
}

// EventPublisher defines the contract for asynchronously publishing logs.
// This interface allows us to easily swap NATS out for Kafka or mock it in tests.
type EventPublisher interface {
	Publish(subject string, event LogEvent) error
	Close()
}

type NatsPublisher struct {
	conn   *nats.Conn
	logger *slog.Logger
}

// NewNatsPublisher connects to the NATS cluster.
func NewNatsPublisher(url string, logger *slog.Logger) (*NatsPublisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &NatsPublisher{conn: nc, logger: logger}, nil
}

// Publish converts the event to JSON and sends it non-blocking over the NATS connection.
func (n *NatsPublisher) Publish(subject string, event LogEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	// NATS handles buffering and asynchronous delivery under the hood,
	// ensuring this call returns almost instantly (microseconds).
	return n.conn.Publish(subject, data)
}

// Close gracefully closes the NATS connection.
func (n *NatsPublisher) Close() {
	if n.conn != nil {
		n.conn.Close()
	}
}

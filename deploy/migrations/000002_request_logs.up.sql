CREATE TABLE IF NOT EXISTS request_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL,
    project_id VARCHAR(50) NOT NULL,
    client_ip VARCHAR(45) NOT NULL,
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    status_code INT NOT NULL,
    latency_ms INT NOT NULL,
    blocked BOOLEAN NOT NULL DEFAULT FALSE,
    block_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast time-series queries by project for the dashboard
CREATE INDEX idx_request_logs_project_id ON request_logs(project_id, timestamp DESC);

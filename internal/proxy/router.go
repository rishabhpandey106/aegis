package proxy

import "context"

// RouteConfig represents the routing information necessary to proxy a request.
type RouteConfig struct {
	ProjectID   string
	UpstreamURL string
	IsActive    bool
}

// ConfigProvider defines the interface for dynamically fetching routing rules.
// By abstracting this, we can easily swap between a database lookup, 
// an in-memory cache, or a Redis-backed configuration without altering the proxy logic.
type ConfigProvider interface {
	GetRoute(ctx context.Context, projectID string) (*RouteConfig, error)
}

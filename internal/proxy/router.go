package proxy

import "context"

type contextKey string

// RouteConfigKey is the key used to store the RouteConfig in the context
const RouteConfigKey contextKey = "route_config"

// RouteConfig represents the routing information necessary to proxy a request.
type RouteConfig struct {
	ProjectID     string
	UpstreamURL   string
	IsActive      bool
	SecurityRules map[string][]byte // Maps rule_type -> raw configuration JSON
}

// ConfigProvider defines the interface for dynamically fetching routing rules.
// By abstracting this, we can easily swap between a database lookup,
// an in-memory cache, or a Redis-backed configuration without altering the proxy logic.
type ConfigProvider interface {
	GetRoute(ctx context.Context, apiKey string) (*RouteConfig, error)
}

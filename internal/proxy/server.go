package proxy

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/sony/gobreaker"
)

// Server represents the core reverse proxy server with dynamic routing.
type Server struct {
	logger         *slog.Logger
	configProvider ConfigProvider
	cbRegistry     *CircuitBreakerRegistry
}

// NewServer initializes the proxy with a dynamic configuration provider.
func NewServer(logger *slog.Logger, provider ConfigProvider) *Server {
	return &Server{
		logger:         logger,
		configProvider: provider,
		cbRegistry:     NewCircuitBreakerRegistry(),
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Fetch routing config dynamically from context (injected by RouteContextMiddleware)
	route, ok := r.Context().Value(RouteConfigKey).(*RouteConfig)
	if !ok || route == nil {
		s.logger.Error("RouteConfig missing from context! Check middleware chain.")
		http.Error(w, "Internal configuration error", http.StatusInternalServerError)
		return
	}
	projectID := route.ProjectID

	parsedURL, err := url.Parse(route.UpstreamURL)
	if err != nil {
		s.logger.Error("Invalid upstream URL configured", "project_id", projectID, "url", route.UpstreamURL)
		http.Error(w, "Internal configuration error", http.StatusInternalServerError)
		return
	}

	// Instantiate the reverse proxy for this specific request
	rp := httputil.NewSingleHostReverseProxy(parsedURL)

	// Attach Circuit Breaker Transport
	rp.Transport = &CircuitBreakerTransport{
		BaseTransport: http.DefaultTransport,
		Registry:      s.cbRegistry,
	}

	// Customize ErrorHandler to catch Circuit Breaker Open State
	rp.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		if errors.Is(err, gobreaker.ErrOpenState) {
			s.logger.Warn("Circuit Breaker OPEN - Blocking traffic to protect upstream", "project_id", projectID, "upstream", parsedURL.Host)
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusServiceUnavailable)
			rw.Write([]byte(`{"error": "Upstream service temporarily unavailable", "code": 503}`))
			return
		}

		// Default bad gateway behavior
		s.logger.Error("Upstream proxy error", "error", err, "project_id", projectID)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusBadGateway)
		rw.Write([]byte(`{"error": "Bad Gateway", "code": 502}`))
	}

	// Override Director to rewrite headers before dispatching to upstream
	originalDirector := rp.Director
	rp.Director = func(req *http.Request) {
		originalDirector(req)

		// Map Host header to upstream's expectations
		req.Host = parsedURL.Host

		// Inject Aegis footprint header
		req.Header.Set("X-Aegis-Proxy", "true")

		// Strip the internal routing header for security so the backend doesn't see it
		req.Header.Del("X-API-Key")
	}

	s.logger.Info("Proxying request via Dynamic Router",
		"project_id", projectID,
		"method", r.Method,
		"path", r.URL.Path,
		"upstream", parsedURL.Host,
	)

	// Execute the reverse proxy
	rp.ServeHTTP(w, r)
}

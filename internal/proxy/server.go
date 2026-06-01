package proxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Server represents the core reverse proxy server with dynamic routing.
type Server struct {
	logger         *slog.Logger
	configProvider ConfigProvider
}

// NewServer initializes the proxy with a dynamic configuration provider.
func NewServer(logger *slog.Logger, provider ConfigProvider) *Server {
	return &Server{
		logger:         logger,
		configProvider: provider,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	projectID := r.Header.Get("X-Aegis-Project-Id")

	// Fetch routing config dynamically from context (injected by RouteContextMiddleware)
	route, ok := r.Context().Value(RouteConfigKey).(*RouteConfig)
	if !ok || route == nil {
		s.logger.Error("RouteConfig missing from context! Check middleware chain.", "project_id", projectID)
		http.Error(w, "Internal configuration error", http.StatusInternalServerError)
		return
	}

	parsedURL, err := url.Parse(route.UpstreamURL)
	if err != nil {
		s.logger.Error("Invalid upstream URL configured", "project_id", projectID, "url", route.UpstreamURL)
		http.Error(w, "Internal configuration error", http.StatusInternalServerError)
		return
	}

	// Instantiate the reverse proxy for this specific request
	rp := httputil.NewSingleHostReverseProxy(parsedURL)

	// Override Director to rewrite headers before dispatching to upstream
	originalDirector := rp.Director
	rp.Director = func(req *http.Request) {
		originalDirector(req)

		// Map Host header to upstream's expectations
		req.Host = parsedURL.Host

		// Inject Aegis footprint header
		req.Header.Set("X-Aegis-Proxy", "true")

		// Strip the internal routing header for security so the backend doesn't see it
		req.Header.Del("X-Aegis-Project-Id")
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

package middleware

import (
	"context"
	"net/http"

	"github.com/aegis/firewall/internal/proxy"
)

// RouteContextMiddleware fetches the RouteConfig from the cache and injects it into the request context.
// This allows all downstream middlewares (RateLimit, WAF, etc.) to access dynamic security rules
// without having to fetch from the cache/DB repeatedly.
func RouteContextMiddleware(provider proxy.ConfigProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			projectID := r.Header.Get("X-Aegis-Project-Id")
			if projectID == "" {
				http.Error(w, "Missing X-Aegis-Project-Id header for dynamic routing", http.StatusBadRequest)
				return
			}

			route, err := provider.GetRoute(r.Context(), projectID)
			if err != nil {
				http.Error(w, "Project not found or invalid", http.StatusNotFound)
				return
			}

			if !route.IsActive {
				http.Error(w, "Project is temporarily inactive", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), proxy.RouteConfigKey, route)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

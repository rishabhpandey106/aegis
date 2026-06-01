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
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				http.Error(w, "Missing X-API-Key header", http.StatusUnauthorized)
				return
			}

			route, err := provider.GetRoute(r.Context(), apiKey)
			if err != nil {
				http.Error(w, "Unauthorized - Invalid API Key", http.StatusUnauthorized)
				return
			}

			if !route.IsActive {
				http.Error(w, "Project is temporarily inactive", http.StatusForbidden)
				return
			}

			// Pass the Project ID "up" the middleware chain to the Analytics logger
			w.Header().Set("X-Aegis-Project-Id", route.ProjectID)

			ctx := context.WithValue(r.Context(), proxy.RouteConfigKey, route)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

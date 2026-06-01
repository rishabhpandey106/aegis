package api

import (
	"log/slog"
	"net/http"
	"strings"
)

// AuthMiddleware protects the Control Plane REST APIs.
// For the MVP, we use a basic static token check. In Phase 4, this would be upgraded 
// to parse a real JWT and extract the user's OrgID to enforce strict RBAC.
func AuthMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				logger.Warn("Missing or invalid Authorization header", "path", r.URL.Path)
				http.Error(w, "Unauthorized - Missing Bearer Token", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			// MVP: Hardcoded secret token for Control Plane access until UI Auth0/Clerk is integrated.
			if token != "super-secret-aegis-token" {
				logger.Warn("Invalid access token provided", "path", r.URL.Path)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Token is valid. In a real system, we would inject the UserContext here.
			next.ServeHTTP(w, r)
		})
	}
}

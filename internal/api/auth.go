package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/aegis/firewall/internal/models"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UserRoleKey contextKey = "user_role"
)

// AuthMiddleware validates Clerk JWT tokens and injects the user details into the context.
func AuthMiddleware(logger *slog.Logger, userRepo models.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				logger.Warn("Missing or invalid Authorization header", "path", r.URL.Path)
				http.Error(w, "Unauthorized - Missing Bearer Token", http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// Verify the JWT signature automatically using Clerk's JWKS
			claims, err := jwt.Verify(r.Context(), &jwt.VerifyParams{
				Token: tokenStr,
			})
			if err != nil {
				logger.Warn("Invalid Clerk JWT provided", "error", err, "path", r.URL.Path)
				http.Error(w, "Unauthorized - Invalid Token", http.StatusUnauthorized)
				return
			}

			// Token is valid!
			userID := claims.Subject

			// Special exception for /api/v1/auth/me which provisions the user
			if r.URL.Path == "/api/v1/auth/me" {
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				// Put a dummy role to allow the request to pass to the handler
				ctx = context.WithValue(ctx, UserRoleKey, "admin")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			dbUser, err := userRepo.GetByClerkID(userID)
			if err != nil {
				logger.Warn("Valid token, but user not found in database", "clerk_id", userID, "path", r.URL.Path)
				http.Error(w, "Unauthorized - User Not Provisioned", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, UserRoleKey, dbUser.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole enforces Role-Based Access Control (RBAC) on specific routes.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, ok := r.Context().Value(UserRoleKey).(string)
			if !ok {
				http.Error(w, "Forbidden - No Role Found", http.StatusForbidden)
				return
			}

			for _, role := range allowedRoles {
				if userRole == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "Forbidden - Insufficient Permissions", http.StatusForbidden)
		})
	}
}

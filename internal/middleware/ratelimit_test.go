package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// MockRateLimiter simulates a rate limiter for unit testing without needing Redis.
type MockRateLimiter struct {
	allowed bool
	err     error
}

func (m *MockRateLimiter) Allow(ctx context.Context, key string, limit int, windowSec int) (bool, error) {
	return m.allowed, m.err
}

func TestRateLimitMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// A dummy handler that represents our core proxy success path
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Proxy Success"))
	})

	t.Run("Request Allowed", func(t *testing.T) {
		limiter := &MockRateLimiter{allowed: true, err: nil}
		handler := RateLimitMiddleware(logger, limiter, 100, 60)(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rr.Code)
		}
	})

	t.Run("Request Rate Limited", func(t *testing.T) {
		limiter := &MockRateLimiter{allowed: false, err: nil}
		handler := RateLimitMiddleware(logger, limiter, 100, 60)(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("Expected 429 Too Many Requests, got %d", rr.Code)
		}
	})
}

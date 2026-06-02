package proxy

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// MockConfigProvider mocks the DB/Cache lookup for clean testing.
type MockConfigProvider struct {
	routes map[string]*RouteConfig
}

func (m *MockConfigProvider) GetRoute(ctx context.Context, projectID string) (*RouteConfig, error) {
	route, ok := m.routes[projectID]
	if !ok {
		return nil, errors.New("project not found")
	}
	return route, nil
}

func TestProxyServerDynamicRouting(t *testing.T) {
	// 1. Setup a dummy upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ensure the routing header was safely stripped before reaching upstream
		if r.Header.Get("X-API-Key") != "" {
			t.Errorf("Expected X-API-Key to be stripped by the proxy")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Dynamic Upstream Response"))
	}))
	defer upstream.Close()

	// 2. Provide mock routes
	provider := &MockConfigProvider{
		routes: map[string]*RouteConfig{
			"valid-project": {
				ProjectID:   "valid-project",
				UpstreamURL: upstream.URL,
				IsActive:    true,
			},
			"inactive-project": {
				ProjectID:   "inactive-project",
				UpstreamURL: upstream.URL,
				IsActive:    false,
			},
		},
	}

	// 3. Initialize proxy
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	proxySrv := NewServer(logger, provider)

	// Wrap in a test middleware to inject context since httptest.NewServer makes real network calls
	contextInjector := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		route, err := provider.GetRoute(r.Context(), apiKey)
		if err != nil {
			if err.Error() == "project not found" {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		if !route.IsActive {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), RouteConfigKey, route)
		proxySrv.ServeHTTP(w, r.WithContext(ctx))
	})

	proxyTestSrv := httptest.NewServer(contextInjector)
	defer proxyTestSrv.Close()

	// --- Test Cases ---

	t.Run("Valid Route", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, proxyTestSrv.URL+"/test", nil)
		req.Header.Set("X-API-Key", "valid-project")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "Dynamic Upstream Response" {
			t.Errorf("Unexpected body: %s", string(body))
		}
	})

	t.Run("Missing Routing Header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, proxyTestSrv.URL+"/test", nil)
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for missing header, got %d", resp.StatusCode)
		}
	})

	t.Run("Inactive Project", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, proxyTestSrv.URL+"/test", nil)
		req.Header.Set("X-API-Key", "inactive-project")
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for inactive project, got %d", resp.StatusCode)
		}
	})

	t.Run("Unknown Project", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, proxyTestSrv.URL+"/test", nil)
		req.Header.Set("X-API-Key", "ghost-project")
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found for unknown project, got %d", resp.StatusCode)
		}
	})
}

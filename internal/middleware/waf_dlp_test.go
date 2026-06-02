package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func TestWAFMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Safe"))
	})

	handler := WAFMiddleware(logger)(nextHandler)

	t.Run("Safe Request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rr.Code)
		}
	})

	t.Run("SQLi in URL", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users?id="+url.QueryEscape("1' OR 1=1 --"), nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", rr.Code)
		}
	})

	t.Run("XSS in Body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer([]byte(`<script>alert(1)</script>`)))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", rr.Code)
		}
	})
}

func TestDLPMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("Safe Response", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"user": "John Doe", "id": 123}`))
		})
		handler := DLPMiddleware(logger)(nextHandler)
		req := httptest.NewRequest("GET", "/api/users/1", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rr.Code)
		}
	})

	t.Run("Credit Card Leak Blocked", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Upstream tries to return a CC number
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"user": "John Doe", "cc": "1234-5678-9012-3456"}`))
		})
		handler := DLPMiddleware(logger)(nextHandler)
		req := httptest.NewRequest("GET", "/api/users/1", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden due to DLP, got %d", rr.Code)
		}

		var resp map[string]string
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["error"] != "Data Leak Prevention" {
			t.Errorf("Expected DLP error, got %s", resp["error"])
		}
	})
}

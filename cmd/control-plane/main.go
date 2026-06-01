package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aegis/firewall/internal/api"
	"github.com/aegis/firewall/internal/db"
	_ "github.com/lib/pq"
)

func main() {
	var port string
	var dbURL string

	flag.StringVar(&port, "port", "8081", "Port for the control plane API to listen on")
	flag.StringVar(&dbURL, "db", "postgres://aegis_user:aegis_password@localhost:5432/aegis?sslmode=disable", "Database Connection URL")
	flag.Parse()

	// Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("Starting Aegis Control Plane", "port", port)

	// Database Connection setup
	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Error("Failed to open database connection", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	if err := dbConn.Ping(); err != nil {
		logger.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("Successfully connected to the database")

	// Dependency Injection: Setup Repositories and Handlers
	projectRepo := db.NewPostgresProjectRepo(dbConn)
	projectHandler := api.NewProjectHandler(logger, projectRepo)

	logsRepo := db.NewPostgresLogsRepository(dbConn, logger)
	analyticsHandler := api.NewAnalyticsHandler(logger, logsRepo)

	// Setup Router
	mux := http.NewServeMux()
	projectHandler.RegisterRoutes(mux)
	analyticsHandler.RegisterRoutes(mux)

	// Add simple logging middleware
	loggedMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Control Plane Request", "method", r.Method, "path", r.URL.Path)
		mux.ServeHTTP(w, r)
	})

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: loggedMux,
	}

	// Start Server
	go func() {
		logger.Info("Control Plane listening", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Control plane server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down control plane...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}
}

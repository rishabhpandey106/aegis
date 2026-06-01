package main

import (
	"database/sql"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/aegis/firewall/internal/db"
	"github.com/aegis/firewall/internal/worker"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
)

func main() {
	var dbURL string
	var natsURL string

	flag.StringVar(&dbURL, "db", "postgres://aegis_user:aegis_password@localhost:5432/aegis?sslmode=disable", "Database Connection URL")
	flag.StringVar(&natsURL, "nats-url", "nats://localhost:4222", "NATS server URL")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("Starting Aegis Analytics Worker...", "nats_target", natsURL)

	// 1. Connect to Database (Storage Layer)
	database, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Error("Failed to connect to Postgres", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	logger.Info("Connected to PostgreSQL successfully")

	repo := db.NewPostgresLogsRepository(database, logger)

	// 2. Connect to NATS (Queue Layer)
	nc, err := nats.Connect(natsURL)
	if err != nil {
		logger.Error("Failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer nc.Close()
	logger.Info("Connected to NATS successfully")

	// 3. Initialize and Start Worker via Dependency Injection
	analyticsWorker := worker.NewAnalyticsWorker(nc, repo, logger)
	if err := analyticsWorker.Start(); err != nil {
		logger.Error("Failed to start Analytics Worker", "error", err)
		os.Exit(1)
	}

	// 4. Graceful Shutdown Handler
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutdown signal received...")
	analyticsWorker.Stop()
}

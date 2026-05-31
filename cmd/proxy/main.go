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

	"github.com/aegis/firewall/internal/cache"
	"github.com/aegis/firewall/internal/db"
	"github.com/aegis/firewall/internal/middleware"
	"github.com/aegis/firewall/internal/proxy"
	"github.com/aegis/firewall/internal/redis"
	_ "github.com/lib/pq"
)

func main() {
	var port string
	var dbURL string
	var redisAddr string
	var redisPass string

	flag.StringVar(&port, "port", "8080", "Port for the proxy to listen on")
	flag.StringVar(&dbURL, "db", "postgres://aegis_user:aegis_password@localhost:5432/aegis?sslmode=disable", "Database Connection URL")
	flag.StringVar(&redisAddr, "redis-addr", "localhost:6379", "Redis address")
	flag.StringVar(&redisPass, "redis-pass", "aegis_redis_pass", "Redis password")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("Initializing Aegis Proxy (Dynamic Routing + Rate Limiting)", "port", port)

	// 1. Connect to Database
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

	// 2. Connect to Redis
	redisClient, err := redis.NewRedisClient(redisAddr, redisPass)
	if err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	logger.Info("Successfully connected to Redis")

	// 3. Setup Repositories and the In-Memory Route Cache
	projectRepo := db.NewPostgresProjectRepo(dbConn)
	routeCache := cache.NewInMemoryRouteCache(projectRepo, 60*time.Second)

	// 4. Initialize Core Proxy
	srv := proxy.NewServer(logger, routeCache)

	// 5. Wrap Proxy in Rate Limiting Middleware (e.g. 10 requests per 60 seconds per IP for MVP testing)
	rateLimiter := redis.NewRedisRateLimiter(redisClient)
	handlerWithMiddleware := middleware.RateLimitMiddleware(logger, rateLimiter, 10, 60)(srv)

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      handlerWithMiddleware,
		// For a reverse proxy, these timeouts must be high enough to allow 
		// the upstream server to process complex requests. 
		// 10s was too short for slow external APIs like httpbin.
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("Proxy server listening", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Proxy server encountered a fatal error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down proxy server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}
}

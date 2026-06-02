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
	grpc_client "github.com/aegis/firewall/internal/grpc"
	"github.com/aegis/firewall/internal/middleware"
	"github.com/aegis/firewall/internal/proxy"
	"github.com/aegis/firewall/internal/queue"
	"github.com/aegis/firewall/internal/redis"
	_ "github.com/lib/pq"
)

func main() {
	var port string
	var dbURL string
	var redisAddr string
	var redisPass string
	var natsURL string

	flag.StringVar(&port, "port", "8080", "Port for the proxy to listen on")
	flag.StringVar(&dbURL, "db", "postgres://aegis_user:aegis_password@localhost:5432/aegis?sslmode=disable", "Database Connection URL")
	flag.StringVar(&redisAddr, "redis-addr", "localhost:6379", "Redis address")
	flag.StringVar(&redisPass, "redis-pass", "aegis_redis_pass", "Redis password")
	flag.StringVar(&natsURL, "nats-url", "nats://localhost:4222", "NATS server URL")
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
	ruleRepo := db.NewPostgresSecurityRuleRepo(dbConn)
	routeCache := cache.NewInMemoryRouteCache(projectRepo, ruleRepo, 5*time.Minute)

	// 4. Initialize Core Proxy
	srv := proxy.NewServer(logger, routeCache)

	// 5. Initialize AI Client via gRPC
	aiClient, err := grpc_client.NewAIClient("localhost:50051")
	if err != nil {
		// We log the error but don't os.Exit(1).
		// If the AI Engine is offline, the middleware automatically "Fails Open" safely.
		logger.Error("Failed to connect to AI Engine on startup", "error", err)
	} else {
		defer aiClient.Close()
		logger.Info("Successfully initialized gRPC connection to AI Engine")
	}

	// 6. Connect to NATS (For asynchronous analytics logging)
	natsPublisher, err := queue.NewNatsPublisher(natsURL, logger)
	if err != nil {
		// FAIL OPEN: We do not want to crash the proxy if the analytics queue is down!
		logger.Error("Failed to connect to NATS. Analytics logging disabled.", "error", err)
	} else {
		defer natsPublisher.Close()
		logger.Info("Successfully connected to NATS")
	}

	// 7. Wrap Proxy in Middlewares
	// ORDER MATTERS EXTREMELY HERE:
	// 1. RateLimiter: Drop fast DOS traffic.
	// 2. WAF: Drop cheap known-patterns (SQLi/XSS).
	// 3. AIBlocker: Expensive LLM check (Prompt Injection).
	// 4. DLP: Egress scanner (Response interception).
	// 5. Proxy: The core upstream router.
	rateLimiter := redis.NewRedisRateLimiter(redisClient)
	pipeline := middleware.RouteContextMiddleware(routeCache)(
		middleware.RateLimitMiddleware(logger, rateLimiter, 10, 60)(
			middleware.WAFMiddleware(logger)(
				middleware.AIBlockerMiddleware(logger, aiClient)(
					middleware.DLPMiddleware(logger)(srv),
				),
			),
		),
	)

	// Finally, wrap the entire pipeline in the Analytics middleware (so it logs total latency and final status)
	var finalHandler http.Handler = pipeline
	if natsPublisher != nil {
		finalHandler = middleware.AnalyticsMiddleware(logger, natsPublisher)(pipeline)
	}

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: finalHandler,
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

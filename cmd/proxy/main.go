package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/s3-azure-proxy/internal/config"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/logging"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/server"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/telemetry"
) // main is the entry point for the S3 to Azure Blob Storage proxy application.
// It performs the following initialization steps:
//  1. Loads configuration from environment variables
//  2. Initializes the logging system (file, console, or both)
//  3. Creates the HTTP router and S3 API server
//  4. Starts the HTTP server (HTTP or HTTPS based on config)
//  5. Handles graceful shutdown on SIGINT/SIGTERM signals
func main() {
	// Load configuration from environment variables first.
	// This must happen before logging setup to get logging configuration.
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize custom logger with configured output mode and level.
	// Supports console-only, file-only, or both console and file logging.
	logger, err := logging.NewLogger(cfg.LogFile, logging.LogLevel(cfg.LogLevel), cfg.LogMode)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	// Initialize telemetry manager for metrics and logs export.
	ctx := context.Background()
	telMgr, err := telemetry.NewManager(ctx)
	if err != nil {
		logger.Error("failed to initialize telemetry manager", zap.Error(err))
		// Continue without telemetry rather than failing startup
	}
	if telMgr != nil && telMgr.IsEnabled() {
		logger.Info("telemetry enabled", zap.String("config", "see TELEMETRY.md for details"))
	}

	// Create the Chi HTTP router for request routing and middleware.
	router := chi.NewRouter()

	// Initialize the S3 proxy server with the router, configuration, logger, and telemetry manager.
	// This sets up all routes, middleware, and connects the Azure backend.
	proxyServer, err := server.NewS3ProxyServer(router, cfg, logger.GetZapLogger(), telMgr)
	if err != nil {
		logger.Crit("failed to create S3 proxy server", zap.Error(err))
	}

	// Create the HTTP server with configured address and request timeouts.
	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: router,
		// Set read/write timeouts to prevent slow client attacks
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
	}

	// Start the server in a background goroutine to allow signal handling.
	// Server will listen on configured address with optional TLS.
	go func() {
		logger.Info("starting S3 proxy server",
			zap.String("addr", cfg.ListenAddr),
			zap.Bool("tls", cfg.EnableTLS),
			zap.String("log_level", cfg.LogLevel),
			zap.String("log_mode", cfg.LogMode))

		var err error
		// Start server with HTTPS if enabled, otherwise use HTTP
		if cfg.EnableTLS {
			err = httpServer.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
		} else {
			err = httpServer.ListenAndServe()
		}

		// Log any startup errors (except ErrServerClosed which is expected during shutdown)
		if err != nil && err != http.ErrServerClosed {
			logger.Crit("server error", zap.Error(err))
		}
	}()

	// Set up signal handling for graceful shutdown.
	// Listen for SIGINT (Ctrl+C) and SIGTERM signals.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Perform graceful shutdown with timeout.
	logger.Info("shutting down server")

	// Close the proxy server to clean up resources (especially cache manager)
	if err := proxyServer.Close(); err != nil {
		logger.Error("error closing proxy server", zap.Error(err))
	}

	// Shutdown telemetry manager to flush any pending metrics
	if telMgr != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := telMgr.Shutdown(shutdownCtx); err != nil {
			logger.Error("error shutting down telemetry", zap.Error(err))
		}
		cancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the HTTP server, allowing in-flight requests to complete.
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	logger.Info("server stopped")
}

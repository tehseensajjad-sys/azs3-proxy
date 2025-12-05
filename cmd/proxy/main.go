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
)

func main() {
	// Load config first to get logging configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize custom logger with file and/or console output
	logger, err := logging.NewLogger(cfg.LogFile, logging.LogLevel(cfg.LogLevel), cfg.LogMode)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Create router
	router := chi.NewRouter()

	// Initialize S3 proxy server
	_, err = server.NewS3ProxyServer(router, cfg, logger.GetZapLogger())
	if err != nil {
		logger.Crit("failed to create S3 proxy server", zap.Error(err))
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: router,
		// Add timeouts
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("starting S3 proxy server",
			zap.String("addr", cfg.ListenAddr),
			zap.Bool("tls", cfg.EnableTLS),
			zap.String("log_level", cfg.LogLevel),
			zap.String("log_mode", cfg.LogMode))
		var err error
		if cfg.EnableTLS {
			err = httpServer.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
		} else {
			err = httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Crit("server error", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	logger.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	logger.Info("server stopped")
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"github.com/vibhansa-msft/azs3-proxy/internal/logging"
	"github.com/vibhansa-msft/azs3-proxy/internal/server"
	"github.com/vibhansa-msft/azs3-proxy/internal/telemetry"
	"github.com/vibhansa-msft/azs3-proxy/internal/version"
)

var (
	pidFileName    string
	defaultLogFile string
)

func init() {
	// Determine binary name for dynamic file naming
	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}
	baseName := filepath.Base(exe)

	pidFileName = baseName + ".pid"
	defaultLogFile = baseName + ".log"
}

// main is the entry point for the S3 to Azure Blob Storage proxy application.
// It performs the following initialization steps:
//  1. Loads configuration from environment variables and CLI flags
//  2. Initializes the logging system (file, console, or both)
//  3. Creates the HTTP router and S3 API server
//  4. Starts the HTTP server (HTTP or HTTPS based on config)
//  5. Handles graceful shutdown on SIGINT/SIGTERM signals
func main() {
	// Parse CLI flags
	listenAddr := flag.String("addr", "", "Address and port to listen on (e.g., :8080 or 127.0.0.1:9000)")
	logFileFlag := flag.String("log-file", "", "Path to log file")
	logLevelFlag := flag.String("log-level", "", "Log level (debug, info, warn, error, crit)")
	foreground := flag.Bool("foreground", false, "Run in foreground")
	stop := flag.Bool("stop", false, "Stop the running daemon")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	pprof := flag.Bool("pprof", false, "Enable pprof profiling server on localhost:6060")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("azs3-proxy version %s\n", version.Version)
		return
	}

	if *stop {
		stopDaemon()
		return
	}

	if !*foreground {
		startDaemon()
		return
	}

	// Setup crash handling (core dumps and panic logs)
	setupCrashHandler()

	// Load configuration from environment variables first.
	// This must happen before logging setup to get logging configuration.
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Override configuration with CLI flags if provided
	if *listenAddr != "" {
		cfg.ListenAddr = *listenAddr
	}
	if *logFileFlag != "" {
		cfg.LogFile = *logFileFlag
	}
	if *logLevelFlag != "" {
		cfg.LogLevel = *logLevelFlag
	}

	// Set default log file if not configured
	if cfg.LogFile == "" {
		cfg.LogFile = defaultLogFile
	}

	// Default to file logging if not explicitly set by environment
	if os.Getenv("LOG_MODE") == "" {
		cfg.LogMode = "file"
	}

	// Initialize custom logger with configured output mode and level.
	// Supports console-only, file-only, or both console and file logging.
	logger, err := logging.NewLogger(cfg.LogFile, logging.LogLevel(cfg.LogLevel), cfg.LogMode)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("starting azs3-proxy", zap.String("version", version.Version))

	// Start pprof server for profiling if enabled
	if *pprof {
		go func() {
			logger.Info("starting pprof server on localhost:6060")
			if err := http.ListenAndServe("localhost:6060", nil); err != nil {
				logger.Error("pprof server error", zap.Error(err))
			}
		}()
	}

	// Initialize telemetry manager for metrics and logs export.
	ctx := context.Background()
	telMgr, err := telemetry.NewManager(ctx)
	if err != nil {
		logger.Error("failed to initialize telemetry manager", zap.Error(err))
		// Continue without telemetry rather than failing startup
	}
	if telMgr != nil && telMgr.IsEnabled() {
		logger.Info("telemetry enabled", zap.String("config", "see TELEMETRY.md for details"))

		// Attach OTel Zap Core for log export
		otelCore := telemetry.NewOtelZapCore(logger.GetZapLogger().Core())
		logger = logger.WithCore(otelCore)
	} else {
		logger.Info("telemetry disabled")
	}

	// Create the Chi HTTP router for request routing and middleware.
	router := chi.NewRouter()

	// Initialize the S3 proxy server with the router, configuration, logger, and telemetry manager.
	// This sets up all routes, middleware, and connects the Azure backend.
	proxyServer, err := server.NewS3ProxyServer(router, cfg, logger.GetZapLogger(), telMgr)
	if err != nil {
		logger.Crit("failed to create S3 proxy server", zap.Error(err))
	}

	// Wrap router with OpenTelemetry handler if tracing is enabled
	var handler http.Handler = router
	if telMgr != nil && telMgr.IsEnabled() {
		handler = otelhttp.NewHandler(router, "s3-proxy")
	}

	// Configure the HTTP server with timeouts and handler.
	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: handler,
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

func startDaemon() {
	// Check if already running
	if _, err := os.Stat(pidFileName); err == nil {
		data, err := os.ReadFile(pidFileName)
		if err == nil {
			pidStr := strings.TrimSpace(string(data))
			if pid, err := strconv.Atoi(pidStr); err == nil {
				if proc, err := os.FindProcess(pid); err == nil {
					// Check if process is actually running by sending signal 0
					if err := proc.Signal(syscall.Signal(0)); err == nil {
						fmt.Printf("Daemon is already running (PID: %d). Please stop it first.\n", pid)
						os.Exit(1)
					}
				}
			}
		}
		// PID file exists but process is dead, clean it up
		_ = os.Remove(pidFileName)
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("Failed to determine executable path: %v\n", err)
		os.Exit(1)
	}

	// Reconstruct arguments, appending --foreground to prevent infinite recursion
	var args []string
	args = append(args, os.Args[1:]...)
	args = append(args, "--foreground")

	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(), "GOTRACEBACK=crash")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	// Only enable debug logging if the level is explicitly set to debug
	logLevel := os.Getenv("LOG_LEVEL")
	if strings.ToLower(logLevel) == "debug" {
		// Redirect stdout/stderr to a debug file to capture any startup errors or panics
		debugLog, err := os.OpenFile("daemon-debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			cmd.Stdout = debugLog
			cmd.Stderr = debugLog
		}
	} else {
		// Discard stdout/stderr if not in debug mode
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start daemon: %v\n", err)
		os.Exit(1)
	}

	pid := cmd.Process.Pid
	if err := os.WriteFile(pidFileName, []byte(strconv.Itoa(pid)), 0644); err != nil {
		fmt.Printf("Warning: Failed to write PID file: %v\n", err)
	}

	fmt.Printf("S3 Proxy started in background (PID: %d)\n", pid)
	fmt.Printf("Logs are being written to %s\n", defaultLogFile)
}

func stopDaemon() {
	data, err := os.ReadFile(pidFileName)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("PID file not found. Is the server running?")
		} else {
			fmt.Printf("Error reading PID file: %v\n", err)
		}
		return
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fmt.Printf("Invalid PID in file: %v\n", err)
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("Process %d not found: %v\n", pid, err)
		return
	}

	// Send SIGTERM
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("Failed to stop process %d: %v\n", pid, err)
		return
	}

	fmt.Printf("Stopped process %d\n", pid)
	_ = os.Remove(pidFileName)
}

// setupCrashHandler configures the process to generate core dumps and crash logs
func setupCrashHandler() {
	// 1. Enable core dumps in Go runtime
	debug.SetTraceback("crash")

	// 2. Try to set ulimit -c unlimited to allow OS to write core files
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_CORE, &rLimit)
	if err == nil {
		rLimit.Max = ^uint64(0)
		rLimit.Cur = ^uint64(0)
		_ = syscall.Setrlimit(syscall.RLIMIT_CORE, &rLimit)
	}

	// 3. Redirect stderr to a crash log file if we are not running in a terminal
	// This ensures panic stack traces are captured in a file named <binary>-crash-<pid>.log
	if !isTerminal(int(os.Stderr.Fd())) {
		exe, _ := os.Executable()
		baseName := filepath.Base(exe)
		pid := os.Getpid()
		crashFile := fmt.Sprintf("%s-crash-%d.log", baseName, pid)

		// Open crash log file
		f, err := os.OpenFile(crashFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err == nil {
			// Redirect stderr (fd 2) to the file
			// We ignore the error here as we can't do much if it fails
			_ = syscall.Dup2(int(f.Fd()), 2)
		}
	}
}

// isTerminal checks if the file descriptor is a terminal
func isTerminal(fd int) bool {
	fileInfo, err := os.Stat("/dev/stderr")
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"github.com/vibhansa-msft/azs3-proxy/internal/version"
)

// setBaseEnv configures required auth/env vars and clears cap vars.
func setBaseEnv(t *testing.T) func() {
	t.Helper()
	_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
	_ = os.Setenv("AZURE_STORAGE_KEY", "testkey")
	_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
	_ = os.Setenv("S3_SECRET_KEY", "testsecret")
	_ = os.Unsetenv("CAP_MBPS")
	_ = os.Unsetenv("CAP_MBPS_READ")
	_ = os.Unsetenv("CAP_MBPS_WRITE")

	return func() {
		_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
		_ = os.Unsetenv("AZURE_STORAGE_KEY")
		_ = os.Unsetenv("S3_ACCESS_KEY")
		_ = os.Unsetenv("S3_SECRET_KEY")
		_ = os.Unsetenv("CAP_MBPS")
		_ = os.Unsetenv("CAP_MBPS_READ")
		_ = os.Unsetenv("CAP_MBPS_WRITE")
	}
}

func TestApplyCLIOverridesBandwidthCaps(t *testing.T) {
	tests := []struct {
		name           string
		cap            string
		read           string
		write          string
		expectErr      bool
		expectCombined float64
		expectRead     float64
		expectWrite    float64
	}{
		{name: "combined only", cap: "10", expectCombined: 10},
		{name: "read only", read: "5", expectRead: 5},
		{name: "write only", write: "6", expectWrite: 6},
		{name: "read and write", read: "5", write: "7", expectRead: 5, expectWrite: 7},
		{name: "combined with read fails", cap: "10", read: "5", expectErr: true},
		{name: "combined with write fails", cap: "10", write: "5", expectErr: true},
		{name: "combined with both fails", cap: "10", read: "5", write: "6", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setBaseEnv(t)
			defer cleanup()

			cfg, err := config.LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			overrides := cliOverrides{capCombo: tt.cap, capRead: tt.read, capWrite: tt.write}
			if err := applyCLIOverrides(cfg, overrides); err != nil {
				t.Fatalf("applyCLIOverrides failed: %v", err)
			}

			err = cfg.Validate()
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected validation error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}

			if cfg.CapMbpsCombined != tt.expectCombined {
				t.Fatalf("combined cap = %v, want %v", cfg.CapMbpsCombined, tt.expectCombined)
			}
			if cfg.CapMbpsRead != tt.expectRead {
				t.Fatalf("read cap = %v, want %v", cfg.CapMbpsRead, tt.expectRead)
			}
			if cfg.CapMbpsWrite != tt.expectWrite {
				t.Fatalf("write cap = %v, want %v", cfg.CapMbpsWrite, tt.expectWrite)
			}
		})
	}
}

// captureOutput runs f while capturing stdout/stderr output, returning what was printed.
func captureOutput(f func()) string {
	oldOut := os.Stdout
	oldErr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	outCh := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outCh <- buf.String()
	}()

	f()
	_ = w.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	return <-outCh
}

func TestStartDaemonSkipSpawnCreatesPidFile(t *testing.T) {
	t.Setenv("SKIP_DAEMON_SPAWN", "true")
	wd := t.TempDir()
	oldWD, _ := os.Getwd()
	_ = os.Chdir(wd)
	defer os.Chdir(oldWD)
	pidFileName = filepath.Join(wd, "azs3-proxy.pid")

	out := captureOutput(startDaemon)
	if !strings.Contains(out, "Skipping daemon spawn") {
		t.Fatalf("expected skip message, got %q", out)
	}
	data, err := os.ReadFile(pidFileName)
	if err != nil {
		t.Fatalf("pid file missing: %v", err)
	}
	if strings.TrimSpace(string(data)) != "0" {
		t.Fatalf("unexpected pid file contents: %s", string(data))
	}
}

func TestStopDaemonMissingFile(t *testing.T) {
	wd := t.TempDir()
	oldWD, _ := os.Getwd()
	_ = os.Chdir(wd)
	defer os.Chdir(oldWD)
	pidFileName = filepath.Join(wd, "azs3-proxy.pid")

	_ = captureOutput(stopDaemon) // should handle missing file gracefully
}

func TestStopDaemonInvalidPID(t *testing.T) {
	wd := t.TempDir()
	oldWD, _ := os.Getwd()
	_ = os.Chdir(wd)
	defer os.Chdir(oldWD)
	pidFileName = filepath.Join(wd, "azs3-proxy.pid")

	if err := os.WriteFile(pidFileName, []byte("not-a-number"), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	_ = captureOutput(stopDaemon) // should report invalid PID and return
}

func TestStopDaemonProcessNotFound(t *testing.T) {
	wd := t.TempDir()
	oldWD, _ := os.Getwd()
	_ = os.Chdir(wd)
	defer os.Chdir(oldWD)
	pidFileName = filepath.Join(wd, "azs3-proxy.pid")

	if err := os.WriteFile(pidFileName, []byte("999999"), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	_ = captureOutput(stopDaemon) // attempts SIGTERM to non-existent pid; should not panic
}

func TestMainVersionFlag(t *testing.T) {
	origArgs := os.Args
	origFlagSet := flag.CommandLine
	origStdout := os.Stdout
	defer func() {
		os.Args = origArgs
		flag.CommandLine = origFlagSet
		os.Stdout = origStdout
	}()

	os.Args = []string{"azs3-proxy", "-version"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	r, w, _ := os.Pipe()
	os.Stdout = w

	main()

	_ = w.Close()
	out, _ := io.ReadAll(r)

	if !strings.Contains(string(out), version.Version) {
		t.Fatalf("version output missing version string: %q", string(out))
	}
}

func TestMainStopFlagMissingPID(t *testing.T) {
	origArgs := os.Args
	origFlagSet := flag.CommandLine
	origStdout := os.Stdout
	defer func() {
		os.Args = origArgs
		flag.CommandLine = origFlagSet
		os.Stdout = origStdout
	}()

	tmpDir := t.TempDir()
	pidFileName = filepath.Join(tmpDir, "no-pid.pid")
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"azs3-proxy", "--stop"}

	r, w, _ := os.Pipe()
	os.Stdout = w

	main()

	_ = w.Close()
	out, _ := io.ReadAll(r)

	if !strings.Contains(string(out), "PID file not found") {
		t.Fatalf("expected missing pid message, got %q", string(out))
	}
}

func TestIsTerminal(t *testing.T) {
	_ = isTerminal(int(os.Stdout.Fd()))
}

func TestSetupCrashHandlerNoPanic(t *testing.T) {
	setupCrashHandler()
}

func TestApplyCLIOverridesPrecedence(t *testing.T) {
	cfg := &config.Config{
		ListenAddr:      "env-addr",
		LogFile:         "env.log",
		LogLevel:        "info",
		CapMbpsCombined: 1,
		CapMbpsRead:     2,
		CapMbpsWrite:    3,
	}

	overrides := cliOverrides{
		listenAddr: "cli-addr",
		logFile:    "cli.log",
		logLevel:   "debug",
		capCombo:   "9",
		capRead:    "8",
		capWrite:   "7",
	}

	if err := applyCLIOverrides(cfg, overrides); err != nil {
		t.Fatalf("applyCLIOverrides returned error: %v", err)
	}

	if cfg.ListenAddr != "cli-addr" || cfg.LogFile != "cli.log" || cfg.LogLevel != "debug" {
		t.Fatalf("string flags not overridden: %+v", cfg)
	}
	if cfg.CapMbpsCombined != 9 || cfg.CapMbpsRead != 8 || cfg.CapMbpsWrite != 7 {
		t.Fatalf("bandwidth flags not overridden: %+v", cfg)
	}

	// Blank override leaves existing value
	if err := applyCLIOverrides(cfg, cliOverrides{listenAddr: ""}); err != nil {
		t.Fatalf("unexpected error for blank override: %v", err)
	}
	if cfg.ListenAddr != "cli-addr" {
		t.Fatalf("blank override should not change value: %s", cfg.ListenAddr)
	}
}

func TestApplyCLIOverridesInvalidBandwidth(t *testing.T) {
	cfg := &config.Config{}
	err := applyCLIOverrides(cfg, cliOverrides{capCombo: "not-a-number"})
	if err == nil {
		t.Fatalf("expected error for invalid bandwidth override")
	}
}

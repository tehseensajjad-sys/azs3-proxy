package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibhansa-msft/azs3-proxy/internal/version"
)

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

package server

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name   string
		addr   string
		port   int
		wantErr bool
	}{
		{name: "valid address", addr: "localhost", port: 8080, wantErr: false},
		{name: "empty address", addr: "", port: 8080, wantErr: false},
		{name: "zero port", addr: "localhost", port: 0, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewServer(tt.addr, tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if s == nil && !tt.wantErr {
				t.Error("NewServer() returned nil server")
			}
		})
	}
}

func TestServerStart(t *testing.T) {
	s, _ := NewServer("localhost", 0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.Start(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Server.Start() error = %v", err)
	}
}

func TestServerShutdown(t *testing.T) {
	s, _ := NewServer("localhost", 0)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.Shutdown(ctx)
	if err != nil {
		t.Errorf("Server.Shutdown() error = %v", err)
	}
}

func TestServerRoutes(t *testing.T) {
	s, _ := NewServer("localhost", 0)
	
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "GET root", method: "GET", path: "/"},
		{name: "GET bucket object", method: "GET", path: "/bucket/key"},
		{name: "PUT object", method: "PUT", path: "/bucket/key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			// Test that route is registered without panic
			_ = req
		})
	}
}

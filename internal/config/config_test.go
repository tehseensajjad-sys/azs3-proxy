package config

import (
	"os"
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		setup     func()
		cleanup   func()
		expectErr bool
	}{
		{
			name: "config from environment",
			setup: func() {
				os.Setenv("LISTEN_ADDR", "0.0.0.0:8080")
				os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				os.Setenv("AZURE_STORAGE_KEY", "testkey")
				os.Setenv("S3_ACCESS_KEY", "testaccess")
				os.Setenv("S3_SECRET_KEY", "testsecret")
			},
			cleanup: func() {
				os.Unsetenv("LISTEN_ADDR")
				os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				os.Unsetenv("AZURE_STORAGE_KEY")
				os.Unsetenv("S3_ACCESS_KEY")
				os.Unsetenv("S3_SECRET_KEY")
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			defer func() {
				if tt.cleanup != nil {
					tt.cleanup()
				}
			}()
		})
	}
}

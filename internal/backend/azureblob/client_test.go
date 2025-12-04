package azureblob

import "testing"

func TestNewAzureBlobBackend(t *testing.T) {
	tests := []struct {
		name    string
		connStr string
		wantErr bool
	}{
		{name: "invalid connection string", connStr: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAzureBlobBackend(tt.connStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want %v", err, tt.wantErr)
			}
		})
	}
}

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleListBuckets(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		wantStatusCode int
	}{
		{name: "list buckets GET", method: "GET", path: "/", wantStatusCode: http.StatusOK},
		{name: "list buckets invalid method", method: "POST", path: "/", wantStatusCode: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			HandleListBuckets(w, req)
			if w.Code != tt.wantStatusCode {
				t.Errorf("HandleListBuckets() got status %d, want %d", w.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestHandleGetObject(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		wantStatusCode int
	}{
		{name: "get object", method: "GET", path: "/bucket/key", wantStatusCode: http.StatusOK},
		{name: "get missing object", method: "GET", path: "/bucket/missing", wantStatusCode: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			HandleGetObject(w, req)
			if w.Code != tt.wantStatusCode {
				t.Errorf("HandleGetObject() got status %d, want %d", w.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestHandleDeleteObject(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		wantStatusCode int
	}{
		{name: "delete object", method: "DELETE", path: "/bucket/key", wantStatusCode: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			HandleDeleteObject(w, req)
			if w.Code != tt.wantStatusCode {
				t.Errorf("HandleDeleteObject() got status %d, want %d", w.Code, tt.wantStatusCode)
			}
		})
	}
}

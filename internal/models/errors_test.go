package models

import (
	"errors"
	"testing"
)

func TestAWSErrorToS3Error(t *testing.T) {
	tests := []struct {
		name    string
		awsErr  error
		wantErr bool
		wantCode string
	}{
		{name: "nil error", awsErr: nil, wantErr: false},
		{name: "not found error", awsErr: errors.New("NotFound"), wantErr: false, wantCode: "404"},
		{name: "access denied error", awsErr: errors.New("AccessDenied"), wantErr: false, wantCode: "403"},
		{name: "unknown error", awsErr: errors.New("UnknownError"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3Err := AWSErrorToS3Error(tt.awsErr)
			if (s3Err != nil) != tt.wantErr {
				t.Errorf("AWSErrorToS3Error() error = %v, wantErr %v", s3Err, tt.wantErr)
			}
		})
	}
}

func TestS3ErrorStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   int
	}{
		{name: "nil error", err: nil, wantCode: 200},
		{name: "not found error", err: &S3Error{Code: "NoSuchKey", HTTPCode: 404}, wantCode: 404},
		{name: "access denied error", err: &S3Error{Code: "AccessDenied", HTTPCode: 403}, wantCode: 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := S3ErrorStatusCode(tt.err)
			if code != tt.wantCode {
				t.Errorf("S3ErrorStatusCode() got %v, want %v", code, tt.wantCode)
			}
		})
	}
}

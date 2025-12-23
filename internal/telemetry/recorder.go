package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RecordS3Request records S3 API request metrics
func (m *Manager) RecordS3Request(ctx context.Context, operation string, success bool, errorType string) {
	if !m.IsEnabled() || m.metricsProvider == nil {
		return
	}

	method := inferMethod(operation)
	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.String("method", method),
	}

	// Record total requests
	m.metricsProvider.S3RequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))

	if success {
		m.metricsProvider.S3RequestsSuccess.Add(ctx, 1, metric.WithAttributes(attrs...))
	} else {
		m.metricsProvider.S3RequestsErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
		if errorType != "" {
			m.metricsProvider.S3ErrorsByType.Add(ctx, 1, metric.WithAttributes(
				append(attrs, attribute.String("error_type", errorType))...,
			))
		}
	}
}

func inferMethod(op string) string {
	switch op {
	case "GetObject", "ListBuckets", "ListObjects", "ListObjectsV2", "ListParts", "ListMultipartUploads", "GetBucketVersioning", "GetObjectVersion", "ListObjectVersions":
		return "GET"
	case "PutObject", "CreateBucket", "UploadPart", "CopyObject", "PutBucketVersioning":
		return "PUT"
	case "DeleteObject", "DeleteBucket", "AbortMultipartUpload", "DeleteObjectVersion":
		return "DELETE"
	case "HeadObject", "HeadBucket":
		return "HEAD"
	case "InitiateMultipartUpload", "CompleteMultipartUpload":
		return "POST"
	default:
		return "UNKNOWN"
	}
}

// RecordCacheHit records a cache hit
func (m *Manager) RecordCacheHit(ctx context.Context, key string) {
	if !m.IsEnabled() || m.metricsProvider == nil {
		return
	}

	m.metricsProvider.CacheHitsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("key", key),
	))
}

// RecordCacheMiss records a cache miss
func (m *Manager) RecordCacheMiss(ctx context.Context, key string) {
	if !m.IsEnabled() || m.metricsProvider == nil {
		return
	}

	m.metricsProvider.CacheMissesTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("key", key),
	))
}

// RecordCacheEviction records a cache eviction
func (m *Manager) RecordCacheEviction(ctx context.Context, reason string) {
	if !m.IsEnabled() || m.metricsProvider == nil {
		return
	}

	m.metricsProvider.CacheEvictionsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("reason", reason),
	))
}

// RecordCacheExpiration records a cache expiration
func (m *Manager) RecordCacheExpiration(ctx context.Context) {
	if !m.IsEnabled() || m.metricsProvider == nil {
		return
	}

	m.metricsProvider.CacheExpirationsTotal.Add(ctx, 1)
}

// RecordCacheOperation records a cache operation (put, get, delete)
func (m *Manager) RecordCacheOperation(ctx context.Context, opType string) {
	if !m.IsEnabled() || m.metricsProvider == nil {
		return
	}

	m.metricsProvider.CacheOperationsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("type", opType),
	))
}

// RecordAzureRequest records Azure Blob Storage request metrics
func (m *Manager) RecordAzureRequest(ctx context.Context, operation string, success bool, errorType string) {
	if !m.IsEnabled() || m.metricsProvider == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
	}

	// Record total requests
	m.metricsProvider.AzureRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))

	if !success {
		m.metricsProvider.AzureRequestsErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

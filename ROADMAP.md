# Project Roadmap & Pending Items

This document outlines the pending tasks, improvements, and future features for the `s3-azure-proxy` project.

## Priority Levels
- **P0**: Critical / Immediate Blocker
- **P1**: High Priority / Core Feature
- **P2**: Medium Priority / Improvement
- **P3**: Low Priority / Nice to have

## 1. Telemetry & Observability
- [ ] **[P2] Distributed Tracing**: Implement OpenTelemetry Tracing (Spans) for HTTP requests, backend calls, and cache operations. Currently, only Metrics are supported.
- [ ] **[P2] Structured Logging Export**: Implement OTLP log exporter to send logs to collectors/backends instead of just writing to stdout via Zap.
- [ ] **[P3] Telemetry Integration Tests**: Add tests to verify metrics are actually being emitted to the configured exporters.

## 2. S3 Compatibility & Features
- [ ] **[P2] S3 ACL / Canned ACL Support**: Basic mapping of S3 ACLs (private, public-read) to Azure container/blob access levels.
- [ ] **[P2] Presigned URLs**: Implement generation of presigned URLs.
- [ ] **[P3] Lifecycle Policies**: Mapping S3 lifecycle rules to Azure Blob Lifecycle management.
- [ ] **[P3] ListObjectsV2 Pagination**: Verify and robustify the pagination token mapping between S3 (ContinuationToken) and Azure (Marker).

## 3. Infrastructure & Deployment
- [ ] **[P2] Helm Chart**: Create Kubernetes deployment charts for easy deployment.

## 4. Testing & QA
- [x] **[P1] S3 Compliance Tests**: Implemented a custom Go-based compliance test suite (`test/compliance`) covering CRUD, Multipart, Versioning, and Metadata operations.
- [ ] **[P2] Load/Performance Testing**: Benchmarks for throughput and latency, especially measuring the impact of the caching layer.

## 5. Documentation
- [ ] **[P2] Configuration Guide**: A comprehensive guide on all environment variables (expanding on `TELEMETRY.md` to include Auth, Server, and Cache configs).

## Recently Completed
- **S3 Compliance Tests**: Added a comprehensive integration test suite for S3 compatibility verification.
- **Azure Monitor Exporter**: Added support for exporting metrics to Azure Monitor.
- **API Compatibility Matrix**: Documented supported S3 operations.
- **CopyObject Support**: Implemented `CopyObject` using download-upload strategy.



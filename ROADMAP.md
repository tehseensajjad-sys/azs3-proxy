# Project Roadmap & Pending Items

This document outlines the pending tasks, improvements, and future features for the `azs3-proxy` project.

## Priority Levels
- **P0**: Critical / Immediate Blocker
- **P1**: High Priority / Core Feature
- **P2**: Medium Priority / Improvement
- **P3**: Low Priority / Nice to have

## 0. Test and stability
- **[P0] WARP Test**: Test proxy with WARP tests for read/write.
- **[P0] LMCache Test**: Test proxy with LM Cache offload to S3.
- **[P1] Perf Test**: Compare WARP results with Rabata.io reulsts (https://rabata.io/s3-comparison).

## 1. Telemetry & Observability
- **[P3] Telemetry Integration Tests**: Add tests to verify metrics are actually being emitted to the configured exporters.

## 2. S3 Compatibility & Features
- **[P2] S3 ACL / Canned ACL Support**: Basic mapping of S3 ACLs (private, public-read) to Azure container/blob access levels.
- **[P2] Presigned URLs**: Implement generation of presigned URLs.
- **[P3] Lifecycle Policies**: Mapping S3 lifecycle rules to Azure Blob Lifecycle management.
- **[P3] ListObjectsV2 Pagination**: Verify and robustify the pagination token mapping between S3 (ContinuationToken) and Azure (Marker).

## 3. Infrastructure & Deployment
- **[P2] Helm Chart**: Create Kubernetes deployment charts for easy deployment.

## 4. Testing & QA
- **[P2] Load/Performance Testing**: Benchmarks for throughput and latency, especially measuring the impact of the caching layer.

## 5. Documentation
- **[P2] Configuration Guide**: A comprehensive guide on all environment variables (expanding on `TELEMETRY.md` to include Auth, Server, and Cache configs).



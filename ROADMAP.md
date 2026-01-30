# Project Roadmap (Pending Only)

Completed items have been cleared; this list reflects current pending work for `azs3-proxy`.

## Priority Levels
- **P0**: Critical / Immediate Blocker
- **P1**: High Priority / Core Feature
- **P2**: Medium Priority / Improvement
- **P3**: Low Priority / Nice to have

## 0. Test and stability (pending)
- **[P0] LMCache Test**: Test proxy with LM Cache offload to S3.

## 1. Telemetry & Observability (pending)
- **[P3] Telemetry Integration Tests**: Add tests to verify metrics are actually being emitted to the configured exporters.

## 2. S3 Compatibility & Features (pending)
- **[P2] S3 ACL / Canned ACL Support**: Basic mapping of S3 ACLs (private, public-read) to Azure container/blob access levels.
- **[P2] Presigned URLs**: Implement generation of presigned URLs.
- **[P3] Lifecycle Policies**: Mapping S3 lifecycle rules to Azure Blob Lifecycle management.

## 3. Testing & QA (pending)
- **[P2] Load/Performance Testing**: Benchmarks for throughput and latency, especially measuring the impact of the caching layer.



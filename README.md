# azs3-proxy

[![Status: Alpha](https://img.shields.io/badge/status-alpha-orange)]()
[![Language: Go](https://img.shields.io/badge/language-Go-blue)]()
[![Tests](https://github.com/vibhansa-msft/azs3-proxy/actions/workflows/tests.yml/badge.svg?branch=main)](https://github.com/vibhansa-msft/azs3-proxy/actions/workflows/tests.yml)
[![Coverage](https://img.shields.io/badge/coverage-76.0%25-brightgreen)]()
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](https://github.com/vibhansa-msft/azs3-proxy/blob/main/LICENSE)

S3-compatible API gateway that translates S3 REST calls to **Azure Blob Storage** or **Azure Files**. Drop-in replacement — migrate existing S3 applications to Azure with zero code changes.

## What is azs3-proxy?

A lightweight Go proxy that exposes an S3-compatible REST API (with SigV4 auth) and translates requests to Azure storage operations. Any application that speaks S3 — AWS CLI, boto3, Spark, MinIO client, VAST — can use Azure storage without code changes.

**Key capabilities:** Streaming I/O (no buffering), multipart uploads, object versioning, connection pooling, TLS, local object caching, OpenTelemetry observability.

## Supported S3 APIs

| Category | Operations |
|---|---|
| **Bucket** | ListBuckets, CreateBucket, DeleteBucket, HeadBucket, GetBucketLocation |
| **Object** | PutObject, GetObject, HeadObject, DeleteObject, CopyObject, ListObjects, ListObjectsV2 |
| **Multipart** | CreateMultipartUpload, UploadPart, CompleteMultipartUpload, AbortMultipartUpload, ListParts, ListMultipartUploads |
| **Versioning** | PutBucketVersioning, GetBucketVersioning, ListObjectVersions, GetObjectVersion, DeleteObjectVersion |

For the full compatibility matrix, see [COMPATIBILITY.md](COMPATIBILITY.md).

## Storage Backend Mapping

| S3 Concept | Azure Blob (`AZURE_BACKEND_TYPE=blob`) | Azure Files (`AZURE_BACKEND_TYPE=file`) |
|---|---|---|
| Bucket | Blob Container | File Share |
| Object | Block Blob | File (auto-creates directories) |
| Multipart Upload | Staged blocks | Buffered file upload |
| Versioning | Native blob versioning | Not supported |

## Quick Start

### Docker (recommended)

Pull from Docker Hub:
```bash
docker pull bhansalivikas/azs3-proxy:latest
```

Or from GitHub Container Registry:
```bash
docker pull ghcr.io/vibhansa-msft/azs3-proxy:latest
```

Run the container:
```bash
docker run -d -p 8080:8080 \
  -e AZURE_STORAGE_ACCOUNT=youraccount \
  -e AZURE_STORAGE_KEY=yourkey \
  -e S3_ACCESS_KEY=your-s3-access-key \
  -e S3_SECRET_KEY=your-s3-secret-key \
  bhansalivikas/azs3-proxy:latest
```

### From source

```bash
git clone https://github.com/vibhansa-msft/azs3-proxy.git
cd azs3-proxy
make build
./azs3-proxy
```

The proxy listens on `http://localhost:8080` by default. See [Configuration Guide](CONFIGURATION.md) for all environment variables, CLI flags, and auth methods.

## Migrating Existing S3 Applications

The key advantage of azs3-proxy is that **existing S3 applications can migrate to Azure storage without any code changes**. There are multiple ways to point your application at the proxy:

### Option 1: Environment Variable (zero code changes)

AWS SDKs (boto3 >= 1.28.0, AWS CLI v2, AWS SDK for Go v2, etc.) support endpoint override via environment variables:

```bash
export AWS_ENDPOINT_URL_S3=http://localhost:8080
export AWS_ACCESS_KEY_ID=your-s3-access-key
export AWS_SECRET_ACCESS_KEY=your-s3-secret-key
export AWS_DEFAULT_REGION=us-east-1

# Run your existing application as-is
python your_existing_app.py
```

### Option 2: AWS Config File (`~/.aws/config`)

```ini
[profile default]
region = us-east-1
services = azs3-proxy

[services azs3-proxy]
s3 =
  endpoint_url = http://localhost:8080
```

### Option 3: Explicit endpoint in code

**AWS CLI:**
```bash
aws s3 ls --endpoint-url http://localhost:8080
aws s3 cp myfile.txt s3://mybucket/ --endpoint-url http://localhost:8080
```

**Boto3 (Python):**
```python
import boto3
s3 = boto3.client('s3',
    endpoint_url='http://localhost:8080',
    aws_access_key_id='your-s3-access-key',
    aws_secret_access_key='your-s3-secret-key',
    region_name='us-east-1')

s3.list_buckets()
s3.put_object(Bucket='mybucket', Key='file.txt', Body=b'hello')
```

### Path-Style Addressing

azs3-proxy uses **path-style** addressing (`http://host/bucket/key`). If your application defaults to virtual-hosted-style, configure path-style:

```ini
# ~/.aws/config
[profile default]
s3 =
  addressing_style = path
```

### Endpoint Resolution Precedence

1. Explicit `endpoint_url` in code
2. Service-specific env var (`AWS_ENDPOINT_URL_S3`)
3. Global env var (`AWS_ENDPOINT_URL`)
4. Service-specific config file setting
5. Global config file setting
6. Default AWS endpoint

## Configuration

Create a `.env` file or set environment variables. CLI flags (e.g., `--addr`, `--log-level`, `--cap-mbps`) override env vars.

**Minimal setup:**
```bash
export AZURE_STORAGE_ACCOUNT=youraccount
export AZURE_STORAGE_KEY=yourkey
export S3_ACCESS_KEY=your-s3-access-key
export S3_SECRET_KEY=your-s3-secret-key
./azs3-proxy
```

**For Azure Blob Storage** (default): set `AZURE_BACKEND_TYPE=blob` (or omit — blob is the default). S3 buckets map to Blob Containers, objects to Block Blobs. All six Azure auth modes supported.

**For Azure Files**: set `AZURE_BACKEND_TYPE=file`. S3 buckets map to File Shares, objects to Files. Only Account Key and SAS Token auth are supported.

See **[Configuration Guide](CONFIGURATION.md)** for the complete reference — all CLI flags, environment variables, auth modes, cache, bandwidth caps, adaptive concurrency, and telemetry settings.

## Build & Test

```bash
make build                 # Build binary
make test                  # Run all unit tests (with race detector)
make coverage              # Run tests with coverage report
make lint                  # Run linter
make docker-build          # Build Docker image
```

### Integration & Compliance Tests

```bash
# Requires Azure credentials
go test -tags=integration -v ./test/integration/...
go test -tags=integration -v ./test/compliance/...
```

## Benchmarking with WARP

[MinIO WARP](https://github.com/minio/warp) is used for performance benchmarking. See [BENCHMARK.md](BENCHMARK.md) for detailed results.

### Run the full benchmark suite

```bash
make benchmark
```

This starts the proxy, runs all WARP tests across multiple concurrencies, and generates a report in `warp_report.md`.

### Run individual WARP benchmarks

```bash
# Install warp
go install github.com/minio/warp@latest

# GET benchmark (1 MiB objects, 16 concurrent)
warp get --host localhost:8080 \
  --access-key $S3_ACCESS_KEY --secret-key $S3_SECRET_KEY \
  --obj.size 1MiB --concurrent 16 --duration 180s

# PUT benchmark (10 MiB objects)
warp put --host localhost:8080 \
  --access-key $S3_ACCESS_KEY --secret-key $S3_SECRET_KEY \
  --obj.size 10MiB --concurrent 16 --duration 180s

# Mixed workload (45% GET, 45% PUT, 10% DELETE)
warp mixed --host localhost:8080 \
  --access-key $S3_ACCESS_KEY --secret-key $S3_SECRET_KEY \
  --obj.size 1MiB --concurrent 16 --duration 180s

# Multipart upload benchmark
warp multipart-put --host localhost:8080 \
  --access-key $S3_ACCESS_KEY --secret-key $S3_SECRET_KEY \
  --obj.size 100MiB --part.size 10MiB --concurrent 8

# Analyze results
warp analyze put.csv.zst
```

### WARP YAML configs

Pre-built WARP configs are in `test/benchmarking/`:

```bash
# Run a specific config
warp run test/benchmarking/get-1MiB.yml \
  -var Host="localhost:8080" \
  -var AccessKey="$S3_ACCESS_KEY" \
  -var SecretKey="$S3_SECRET_KEY" \
  -var Region="us-east-1" \
  -var Bucket="warp-bench" \
  -var TLS="false" \
  -var Concurrent="16" \
  -var BenchData="get-1MiB.csv.zst"
```

Available configs: `get-1MiB.yml`, `get-10MiB.yml`, `get-100MiB.yml`, `get-2GiB.yml`, `put-1MiB.yml`, `put-10MiB.yml`, `put-100MiB.yml`, `put-2GiB.yml`, `mixed-1MiB.yml`, `small-128KiB.yml`.

## Documentation

| Document | Description |
|---|---|
| [CONFIGURATION.md](CONFIGURATION.md) | Full config reference — env vars, CLI flags, auth methods, cache, telemetry |
| [BENCHMARK.md](BENCHMARK.md) | Performance results and benchmark methodology |
| [TELEMETRY.md](TELEMETRY.md) | OpenTelemetry metrics, tracing, and logging setup |
| [COMPATIBILITY.md](COMPATIBILITY.md) | S3 API compatibility matrix |

## License

MIT — see [LICENSE](LICENSE).

## Support

- [Issues](https://github.com/vibhansa-msft/azs3-proxy/issues)
- [Discussions](https://github.com/vibhansa-msft/azs3-proxy/discussions)

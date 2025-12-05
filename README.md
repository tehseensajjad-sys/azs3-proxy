# s3-azure-proxy

[![Status: Alpha](https://img.shields.io/badge/status-alpha-orange)]()
[![Language: Go](https://img.shields.io/badge/language-Go-blue)]()
[![Tests](https://img.shields.io/github/actions/workflow/status/vibhansa-msft/s3-azure-proxy/tests.yml?branch=main&label=tests)](https://github.com/vibhansa-msft/s3-azure-proxy/actions)
[![Coverage](https://img.shields.io/badge/coverage-61.4%25-yellow)]()
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](https://github.com/vibhansa-msft/s3-azure-proxy/blob/main/LICENSE)

S3-compatible API gateway that translates S3 REST calls to Azure Blob Storage using Azure Go SDK. Enables S3 clients to work with Azure Blob as backend.

## Overview

**s3-azure-proxy** is a lightweight, production-ready proxy server that:

- Exposes an S3-compatible REST API (with SigV4 auth)
- Translates S3 requests to Azure Blob Storage operations
- Allows S3 clients and applications to seamlessly work with Azure Blob Storage
- Supports core S3 bucket and object operations (CRUD, listing, multipart upload)
- Uses the latest Azure SDK for Go with optimized performance

### Key Features

- ✅ Core S3 API support (bucket & object operations, multipart, versioning)
- ✅ AWS Signature Version 4 authentication
- ✅ Multipart uploads for large objects
- ✅ Object versioning support
- ✅ Streaming support (no full object buffering)
- ✅ Connection pooling & concurrency optimization
- ✅ Structured logging with zap
- ✅ Easy Docker deployment

## Use Cases

- Migrate S3-dependent applications to Azure
- Multi-cloud storage abstraction for S3 clients
- Enable VAST, MinIO, or other S3-compatible tools to use Azure Blob Storage
- Development & testing without AWS S3

## Quick Start

### Prerequisites

- Go 1.21 or later
- Azure storage account with connection string or account key
- Docker (optional, for containerized deployment)

### Installation

```bash
git clone https://github.com/vibhansa-msft/s3-azure-proxy.git
cd s3-azure-proxy
go mod download
go build -o bin/s3-proxy ./cmd/proxy
```

### Configuration

Create a `.env` file or set environment variables:

```bash
# HTTP Server
LISTEN_ADDR=:8080

# Azure Storage
AZURE_STORAGE_ACCOUNT=youraccount
AZURE_STORAGE_KEY=yourkey

# S3 Authentication
S3_ACCESS_KEY=AKIA1234567890ABCDEF
S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY

# Logging
LOG_LEVEL=info
```

### Running

```bash
./bin/s3-proxy
```

The proxy listens on `http://localhost:8080` by default.

## Architecture

### Project Structure

```
s3-azure-proxy/
├── cmd/
│   └── proxy/
│       └── main.go              # Entry point
├── internal/
│   ├── config/                  # Configuration loading
│   ├── server/                  # S3 API server setup
│   ├── handler/                 # S3 HTTP handlers (bucket, object ops)
│   ├── backend/                 # Storage backend abstraction
│   │   └── azureblob/           # Azure Blob implementation
│   ├── auth/                    # SigV4 verification
│   └── models/                  # Data models & types
├── .github/workflows/           # CI/CD pipelines
├── go.mod                       # Dependencies
└── README.md
```

### Key Components

1. **HTTP/S3 Frontend** (`internal/handler/`)
   - Chi router for request routing
   - S3 API handlers for buckets and objects
   - SigV4 middleware for authentication

2. **Backend Abstraction** (`internal/backend/`)
   - Clean interface for storage operations
   - Azure Blob implementation
   - Easy to extend for other backends

3. **Auth Layer** (`internal/auth/`)
   - AWS Signature Version 4 verification
   - Access key/secret mapping

4. **Models & Utilities** (`internal/models/`)
   - S3 error code mapping (Azure → S3)
   - S3 XML response serialization
   - Error handling

## Supported S3 Operations

### Bucket Operations

- `PUT /{bucket}` – Create bucket
- `DELETE /{bucket}` – Delete bucket
- `GET /` – List buckets
- `GET /{bucket}` – List objects (v1 & v2)
- `GET /{bucket}?location` – Bucket location

### Object Operations

- `PUT /{bucket}/{key}` – Upload object
- `GET /{bucket}/{key}` – Download object
- `HEAD /{bucket}/{key}` – Object metadata
- `DELETE /{bucket}/{key}` – Delete object
- `PUT /{bucket}/{key}?x-amz-copy-source=...` – Copy object

### Multipart Upload

- `POST /{bucket}/{key}?uploads` – Initiate
- `PUT /{bucket}/{key}?partNumber=X&uploadId=...` – Upload part
- `POST /{bucket}/{key}?uploadId=...` – Complete
- `DELETE /{bucket}/{key}?uploadId=...` – Abort

### Versioning

- `PUT /{bucket}?versioning` – Enable versioning on bucket
- `GET /{bucket}?versioning` – Get versioning status
- `GET /{bucket}?versions` – List object versions
- `GET /{bucket}/{key}?versionId=X` – Get specific version
- `DELETE /{bucket}/{key}?versionId=X` – Delete specific version

### Not Supported (v1)

- ACLs, tagging, object encryption (SSE-S3, SSE-KMS)
- Bucket policies, lifecycle rules
- S3 Select, requester pays

## Configuration Reference

### Server Configuration

| Env Variable | Required | Default | Description |
|---|---|---|---|
| `LISTEN_ADDR` | No | `:8080` | HTTP server listen address |
| `LOG_LEVEL` | No | `info` | Logging level (debug, info, warn, error, crit) |
| `LOG_FILE` | No | – | Path to log file (empty for console only) |
| `LOG_MODE` | No | `console` | Logging mode (console, file, both) |
| `ENABLE_TLS` | No | `false` | Enable HTTPS/TLS support (true/false) |
| `TLS_CERT_FILE` | Conditional | – | Path to TLS certificate file (required if ENABLE_TLS=true) |
| `TLS_KEY_FILE` | Conditional | – | Path to TLS private key file (required if ENABLE_TLS=true) |

### S3 Authentication

| Env Variable | Required | Default | Description |
|---|---|---|---|
| `S3_ACCESS_KEY` | Yes | – | S3 access key ID for SigV4 authentication |
| `S3_SECRET_KEY` | Yes | – | S3 secret access key for SigV4 authentication |

### Azure Storage Authentication

Azure storage supports multiple authentication methods. Choose **one** of the following:

#### Account Key Authentication (Recommended for simplicity)

| Env Variable | Required | Default | Description |
|---|---|---|---|
| `AZURE_STORAGE_ACCOUNT` | Yes | – | Azure storage account name |
| `AZURE_STORAGE_KEY` | Yes | – | Storage account access key (for account key auth) |

#### SAS Token Authentication

| Env Variable | Required | Default | Description |
|---|---|---|---|
| `AZURE_STORAGE_ACCOUNT` | Yes | – | Azure storage account name |
| `AZURE_STORAGE_SAS_TOKEN` | Yes | – | Shared Access Signature token |

#### Managed Identity (MSI) Authentication

| Env Variable | Required | Default | Description |
|---|---|---|---|
| `AZURE_STORAGE_ACCOUNT` | Yes | – | Azure storage account name |
| `AZURE_USE_MSI` | Yes | – | Set to `true` to enable MSI authentication |
| `AZURE_CLIENT_ID` | No | – | Client ID for user-assigned MSI (optional, uses system-assigned by default) |

#### Service Principal Authentication

| Env Variable | Required | Default | Description |
|---|---|---|---|
| `AZURE_STORAGE_ACCOUNT` | Yes | – | Azure storage account name |
| `AZURE_TENANT_ID` | Yes | – | Azure AD tenant ID |
| `AZURE_CLIENT_ID` | Yes | – | Service principal client ID |
| `AZURE_CLIENT_SECRET` | Yes | – | Service principal client secret |

#### Federated Token Authentication (OpenID Connect)

| Env Variable | Required | Default | Description |
|---|---|---|---|
| `AZURE_STORAGE_ACCOUNT` | Yes | – | Azure storage account name |
| `AZURE_CLIENT_ID` | Yes | – | Application client ID |
| `AZURE_TENANT_ID` | Yes | – | Azure AD tenant ID |
| `AZURE_FEDERATED_TOKEN_FILE` | Yes | – | Path to OIDC token file |

#### Azure CLI Authentication

| Env Variable | Required | Default | Description |
|---|---|---|---|
| `AZURE_STORAGE_ACCOUNT` | Yes | – | Azure storage account name |
| `AZURE_USE_CLI_AUTH` | Yes | – | Set to `true` to use Azure CLI cached credentials |

### Optional Azure Configuration

| Env Variable | Required | Default | Description |
|---|---|---|---|
| `AZURE_STORAGE_URL` | No | – | Custom Azure storage URL (auto-generated from account name if not provided) |
| `AZURE_SUBSCRIPTION_ID` | No | – | Azure subscription ID |
| `AZURE_OBJECT_ID` | No | – | Service principal object ID |

## Usage Examples

### Using AWS CLI

```bash
export AWS_ACCESS_KEY_ID=AKIA1234567890ABCDEF
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY

# List buckets
aws s3 ls \
  --endpoint-url http://localhost:8080 \
  --region us-east-1

# Upload file
aws s3 cp myfile.txt s3://mybucket/ \
  --endpoint-url http://localhost:8080 \
  --region us-east-1

# Download file
aws s3 cp s3://mybucket/myfile.txt ./downloaded.txt \
  --endpoint-url http://localhost:8080 \
  --region us-east-1
```

### Using Boto3 (Python)

```python
import boto3

s3 = boto3.client(
    's3',
    endpoint_url='http://localhost:8080',
    aws_access_key_id='AKIA1234567890ABCDEF',
    aws_secret_access_key='wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY',
    region_name='us-east-1'
)

# List buckets
response = s3.list_buckets()
print(response['Buckets'])

# Upload object
s3.put_object(
    Bucket='mybucket',
    Key='myfile.txt',
    Body=b'Hello, World!'
)

# Download object
obj = s3.get_object(Bucket='mybucket', Key='myfile.txt')
data = obj['Body'].read()
```

### HTTPS/TLS Support

To enable HTTPS, generate or provide SSL/TLS certificates and configure the proxy:

```bash
# Generate self-signed certificate (for testing)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Set environment variables
export ENABLE_TLS=true
export TLS_CERT_FILE=/path/to/cert.pem
export TLS_KEY_FILE=/path/to/key.pem

# Start proxy (now listening on HTTPS)
./bin/s3-proxy
```

Then use HTTPS endpoint with S3 clients:

```bash
aws s3 ls \
  --endpoint-url https://localhost:8080 \
  --ca-bundle /path/to/cert.pem \
  --region us-east-1
```

### Logging & Monitoring

The proxy supports flexible logging with multiple output modes and levels:

#### Logging Configuration

```bash
# Console logging only (default)
export LOG_MODE=console
export LOG_LEVEL=info

# File logging only
export LOG_MODE=file
export LOG_FILE=/var/log/s3-proxy.log
export LOG_LEVEL=debug

# Both console and file logging
export LOG_MODE=both
export LOG_FILE=/var/log/s3-proxy.log
export LOG_LEVEL=info
```

#### Log Levels

- **debug**: Detailed debugging information, request/response details
- **info**: General informational messages, successful operations
- **warn**: Warning messages for potentially problematic situations
- **error**: Error messages when operations fail
- **crit**: Critical failures that may require immediate attention

#### Log Output Format

Logs are output in JSON format with the following fields:
```json
{
  "[s3-proxy 1234] {
    "timestamp": "2025-12-05 14:30:45.123",
    "level": "INFO",
    "msg": "object uploaded successfully",
    "caller": "handler/handler.go:195",
    "bucket": "mybucket",
    "key": "myfile.txt",
    "content_length": 1024
  }"
}
```

Each log line includes:
- **Program name and PID**: `[s3-proxy 1234]` at the start of each line
- **Timestamp**: ISO8601 format with millisecond precision
- **Log level**: DEBUG, INFO, WARN, ERROR, FATAL
- **Message**: The primary log message
- **Caller**: File and line number where log originated
- **Contextual fields**: Structured data related to the operation (bucket, key, error details, etc.)

#### Dynamic Log Level Changes

To change the log level while the proxy is running without restarting, modify the `LOG_LEVEL` environment variable and restart only the logging component (this feature is planned for future releases).

## Development

### Building

```bash
go build -o bin/s3-proxy ./cmd/proxy
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -v -cover ./...

# Run specific package
go test -v ./internal/handler

# Run with race detector
go test -v -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Coverage

Current test coverage: **61.4%** across all packages

- `internal/auth`: 87.9% (SigV4 verification)
- `internal/models`: 83.3% (error mapping, XML serialization)
- `internal/config`: 72.0% (config loading & validation)
- `internal/handler`: 69.5% (HTTP handler routing)
- `internal/backend/azureblob`: 50.0% (client initialization)
- `internal/server`: 42.6% (server lifecycle)

**Automated Testing:** All tests run automatically via GitHub Actions on every commit to `main` branch.

### Code Structure

- **Config Layer**: `internal/config` – loads environment variables
- **Server Layer**: `internal/server` – HTTP server initialization
- **Handler Layer**: `internal/handler` – S3 request handling
- **Backend Layer**: `internal/backend` – storage abstraction
- **Auth Layer**: `internal/auth` – SigV4 validation
- **Model Layer**: `internal/models` – data structures & serialization

## Docker Deployment

### Build Docker Image

```bash
docker build -t s3-azure-proxy:latest .
```

### Run Container

```bash
docker run -p 8080:8080 \
  -e AZURE_STORAGE_ACCOUNT=youraccount \
  -e AZURE_STORAGE_KEY=yourkey \
  -e S3_ACCESS_KEY=AKIA1234567890ABCDEF \
  -e S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY \
  s3-azure-proxy:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  s3-proxy:
    image: s3-azure-proxy:latest
    ports:
      - "8080:8080"
    environment:
      LISTEN_ADDR: :8080
      AZURE_STORAGE_ACCOUNT: youraccount
      AZURE_STORAGE_KEY: yourkey
      S3_ACCESS_KEY: AKIA1234567890ABCDEF
      S3_SECRET_KEY: wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY
      LOG_LEVEL: info
```

## Performance & Optimization

- Streaming uploads/downloads (no full-object buffering)
- Connection pooling to Azure Blob Storage
- Configurable timeouts and concurrency
- Efficient SigV4 signature verification
- Parallel multipart upload handling
- Optional caching layer (future enhancement)

## Roadmap

- [x] Core bucket/object operations (v0.1)
- [x] SigV4 authentication (v0.1)
- [x] Multipart upload (v0.1)
- [x] Unit test coverage (v0.1)
- [ ] Versioning support (v0.2)
- [ ] Object tagging (v0.2)
- [ ] Server-side encryption emulation (v0.2)
- [ ] Presigned URLs (v0.3)
- [ ] Performance benchmarks & tuning (v0.3)
- [ ] Helm chart for Kubernetes (v1.0)
- [ ] Production hardening & security audit (v1.0)

## Contributing

Contributions are welcome! Please follow the process:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Write tests for your changes
4. Commit changes (`git commit -am 'Add my feature'`)
5. Push to branch (`git push origin feature/my-feature`)
6. Open a Pull Request

### Code Quality

- All PRs must pass unit tests (`go test ./...`)
- Maintain test coverage at 75%+
- Follow Go conventions and idioms
- Add documentation for new features

## License

MIT License – see [LICENSE](LICENSE) file for details.

## Related Projects

- [S3Proxy](https://github.com/gaul/s3proxy) – S3 gateway in Java (reference implementation)
- [MinIO](https://github.com/minio/minio) – S3-compatible object storage
- [AWS SDK for Go](https://github.com/aws/aws-sdk-go-v2) – AWS API client
- [Azure SDK for Go](https://github.com/Azure/azure-sdk-for-go) – Azure API client

## Support

For issues, questions, or suggestions:

- Open an [issue](https://github.com/vibhansa-msft/s3-azure-proxy/issues)
- Check existing [discussions](https://github.com/vibhansa-msft/s3-azure-proxy/discussions)
- Review [contributing guide](CONTRIBUTING.md)

**Status:** ⚠️ Alpha – API may change. Not production-ready until v1.0.

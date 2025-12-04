# s3-azure-proxy

![Status](https://img.shields.io/badge/status-alpha-orange) ![Language](https://img.shields.io/badge/language-Go-blue) ![License](https://img.shields.io/badge/license-MIT-green)

S3-compatible API gateway that translates S3 REST calls to Azure Blob Storage using Azure Go SDK. Enables S3 clients to work with Azure Blob as backend.

## Overview

**s3-azure-proxy** is a lightweight, production-ready proxy server that:
- Exposes a full S3-compatible REST API (with SigV4 auth)
- Translates S3 requests to Azure Blob Storage operations
- Allows S3 clients and applications to seamlessly work with Azure Blob Storage
- Supports core S3 bucket and object operations (CRUD, listing, multipart upload)

### Use Cases
- Migrate S3-dependent applications to Azure
- Multi-cloud storage abstraction for S3 clients

---

## Quick Start

### Prerequisites
- Go 1.21 or later
- Azure storage account with connection string or account key

### Installation

```bash
git clone https://github.com/vibhansa-msft/s3-azure-proxy.git
cd s3-azure-proxy
go mod download
go build -o bin/s3-proxy ./cmd/proxy
```

### Configuration

Create a `.env` or config file:
```bash
LISTEN_ADDR=:8080
AZURE_STORAGE_ACCOUNT=youraccount
AZURE_STORAGE_KEY=yourkey
S3_ACCESS_KEY=AKIA1234567890ABCDEF
S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY
```

### Running

```bash
./bin/s3-proxy
```

The proxy listens on `http://localhost:8080` by default.

---

## Architecture

### Project Structure

```
s3-azure-proxy/
├── cmd/
│   └── proxy/          # Main entry point
│       └── main.go
├── internal/
│   ├── config/         # Configuration loading
│   ├── server/         # S3 API server setup
│   ├── handler/        # S3 HTTP handlers (bucket, object ops)
│   ├── backend/        # Storage backend abstraction
│   │   └── azureblob/  # Azure Blob implementation
│   ├── auth/           # SigV4 verification
│   └── models/         # Data models & types
├── go.mod
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

---

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

### Not Supported (v1)
- ACLs, versioning, tagging
- Object encryption (SSE-S3, SSE-KMS)
- Bucket policies, lifecycle
- S3 Select, requester pays

---

## Configuration Reference

| Env Variable | Required | Default | Description |
|---|---|---|---|
| `LISTEN_ADDR` | No | `:8080` | HTTP server listen address |
| `AZURE_STORAGE_ACCOUNT` | Yes | – | Azure storage account name |
| `AZURE_STORAGE_KEY` | Yes | – | Storage account access key |
| `S3_ACCESS_KEY` | Yes | – | S3 access key ID for clients |
| `S3_SECRET_KEY` | Yes | – | S3 secret access key |
| `LOG_LEVEL` | No | `info` | Logging level (debug, info, warn, error) |

---

## Usage Example

### Using AWS CLI

```bash
export AWS_ACCESS_KEY_ID=AKIA1234567890ABCDEF
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY

aws s3 \
  --endpoint-url http://localhost:8080 \
  --region us-east-1 \
  ls

aws s3 \
  --endpoint-url http://localhost:8080 \
  --region us-east-1 \
  cp myfile.txt s3://mybucket/
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
s3.put_object(Bucket='mybucket', Key='myfile.txt', Body=b'Hello, World!')

# Download object
obj = s3.get_object(Bucket='mybucket', Key='myfile.txt')
data = obj['Body'].read()
```

---

## Development

### Building

```bash
go build -o bin/s3-proxy ./cmd/proxy
```

### Running Tests

```bash
go test ./...
```

### Code Structure

- **Config Layer**: `internal/config` – loads environment variables
- **Server Layer**: `internal/server` – HTTP server initialization
- **Handler Layer**: `internal/handler` – S3 request handling
- **Backend Layer**: `internal/backend` – storage abstraction
- **Auth Layer**: `internal/auth` – SigV4 validation

---

## Performance & Optimization

- Streaming uploads/downloads (no full-object buffering)
- Connection pooling to Azure Blob Storage
- Configurable timeouts and concurrency
- Optional caching layer (future enhancement)

---

## Roadmap

- [ ] v1.0: Core bucket/object operations
- [ ] Versioning support
- [ ] Object tagging
- [ ] Server-side encryption (SSE-S3 emulation)
- [ ] Presigned URLs
- [ ] Performance benchmarks & tuning
- [ ] Docker image
- [ ] Helm chart for Kubernetes deployment

---

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit changes (`git commit -am 'Add my feature'`)
4. Push to branch (`git push origin feature/my-feature`)
5. Open a Pull Request

---

## License

MIT License – see LICENSE file for details.

---

## References

- [S3Proxy](https://github.com/gaul/s3proxy) – Production S3→multi-cloud proxy
- [AWS Signature Version 4](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_aws-signing.html)
- [Azure Blob Storage SDK for Go](https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/storage/azblob)
- [MinIO Azure Gateway](https://github.com/minio/minio/tree/master/cmd/gateway-azure) (deprecated)

---

**Status**: ⚠️ Alpha – API may change. Not production-ready until v1.0.

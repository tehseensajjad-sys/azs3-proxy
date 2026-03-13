# Configuration Guide

Authoritative reference for configuring `azs3-proxy`. Configuration is loaded from environment variables (or a `.env` file). CLI flags override env vars where available.

## Quick Start

Minimal setup (account key auth, blob backend):

```bash
export AZURE_STORAGE_ACCOUNT=youraccount
export AZURE_STORAGE_KEY=yourkey
export S3_ACCESS_KEY=your-s3-access-key
export S3_SECRET_KEY=your-s3-secret-key
./azs3-proxy
```

## CLI Flags

CLI flags take the highest precedence, overriding both environment variables and `.env` file values.

| Flag | Description |
|---|---|
| `--addr` | Listen address, e.g. `:8080` or `127.0.0.1:9000` (overrides `LISTEN_ADDR`) |
| `--log-file` | Path to log file (overrides `LOG_FILE`) |
| `--log-level` | Log level: `debug`, `info`, `warn`, `error`, `crit` (overrides `LOG_LEVEL`) |
| `--cap-mbps` | Aggregate Azure bandwidth cap in Mbps (overrides `CAP_MBPS`) |
| `--cap-mbps-read` | Download bandwidth cap in Mbps (overrides `CAP_MBPS_READ`) |
| `--cap-mbps-write` | Upload bandwidth cap in Mbps (overrides `CAP_MBPS_WRITE`) |
| `--foreground` | Run in foreground instead of daemon mode |
| `--stop` | Stop the running daemon |
| `--version` | Print version and exit |
| `--pprof` | Enable pprof profiling server on `localhost:6060` |

## Environment Variables

### Server & Runtime

| Env Variable | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | HTTP listen address |
| `ENABLE_TLS` | `false` | Enable HTTPS. When `true`, `TLS_CERT_FILE` and `TLS_KEY_FILE` are required |
| `TLS_CERT_FILE` | – | Path to TLS certificate file |
| `TLS_KEY_FILE` | – | Path to TLS private key file |
| `LOG_MODE` | `console` | Logging output: `console`, `file`, or `both` |
| `LOG_FILE` | `azs3-proxy.log` | Log file path (used when `LOG_MODE` is `file` or `both`) |
| `LOG_LEVEL` | `warn` | Log level: `debug`, `info`, `warn`, `error`, `crit` |
| `DEBUG_RESPONSES` | `false` | When `true`, logs response headers for all routes (verbose) |
| `ADMIN_TOKEN` | – | Shared secret for admin endpoints (e.g., bandwidth caps update) |

### S3 Authentication

| Env Variable | Required | Description |
|---|---|---|
| `S3_ACCESS_KEY` | Yes | S3 access key ID for SigV4 signature verification |
| `S3_SECRET_KEY` | Yes | S3 secret access key for SigV4 signature verification |

### Azure Backend Type

| Env Variable | Default | Description |
|---|---|---|
| `AZURE_BACKEND_TYPE` | `blob` | `blob` for Azure Blob Storage, `file` for Azure Files |

### Azure Authentication

Set `AZURE_STORAGE_ACCOUNT` (required for all modes) and choose **one** authentication mode. The proxy auto-detects the mode based on which env vars are set, in this priority order:

| Mode | Required Env Vars | Optional Env Vars | Description |
|---|---|---|---|
| **Account Key** | `AZURE_STORAGE_KEY` | `AZURE_STORAGE_URL`, `AZURE_SUBSCRIPTION_ID` | Simple shared key auth |
| **SAS Token** | `AZURE_STORAGE_SAS_TOKEN` | `AZURE_STORAGE_URL` | Shared Access Signature token |
| **Federated Token (OIDC)** | `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_FEDERATED_TOKEN_FILE` | `AZURE_STORAGE_URL` | OIDC token file (must exist and be readable) |
| **Service Principal** | `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID` | `AZURE_OBJECT_ID`, `AZURE_STORAGE_URL`, `AZURE_SUBSCRIPTION_ID` | Client secret auth |
| **Managed Identity** | `AZURE_USE_MSI=true` | `AZURE_CLIENT_ID` (user-assigned), `AZURE_STORAGE_URL` | System or user-assigned identity |
| **Azure CLI** | `AZURE_USE_CLI_AUTH=true` | `AZURE_STORAGE_URL` | Uses cached `az login` credentials |

If `AZURE_STORAGE_URL` is omitted, it is auto-generated:
- Blob backend: `https://<account>.blob.core.windows.net`
- File backend: `https://<account>.file.core.windows.net`
- Devstore (`devstoreaccount1`): `http://127.0.0.1:10000/devstoreaccount1`

### Cache

| Env Variable | Default | Description |
|---|---|---|
| `CACHE_ENABLED` | `false` | Enable local object cache |
| `CACHE_PATH` | `/tmp/azs3-proxy-cache` | Directory for cached objects |
| `CACHE_MAX_SIZE` | `1073741824` (1 GiB) | Maximum cache size in bytes |
| `CACHE_TTL` | `3600` | Time-to-live per cache entry in seconds |

### Bandwidth Caps

Limit bandwidth between the proxy and Azure storage. Use either a combined cap **or** per-direction caps (not both).

| Env Variable | Default | Description |
|---|---|---|
| `CAP_MBPS` | `0` (disabled) | Aggregate bandwidth cap (Mbps) for both read and write |
| `CAP_MBPS_READ` | `0` (disabled) | Download bandwidth cap (Mbps) |
| `CAP_MBPS_WRITE` | `0` (disabled) | Upload bandwidth cap (Mbps) |

CLI flags `--cap-mbps`, `--cap-mbps-read`, and `--cap-mbps-write` override these env vars.

### Adaptive Concurrency

Dynamically adjusts the number of concurrent requests to Azure to maintain a target latency.

| Env Variable | Default | Description |
|---|---|---|
| `ADAPTIVE_CONCURRENCY_ENABLED` | `false` | Enable adaptive concurrency limiter |
| `ADAPTIVE_CONCURRENCY_MIN` | `4` | Minimum concurrent requests |
| `ADAPTIVE_CONCURRENCY_MAX` | `64` | Maximum concurrent requests |
| `ADAPTIVE_CONCURRENCY_TARGET_MS` | `200` | Target latency in milliseconds for adjustments |
| `ADAPTIVE_CONCURRENCY_ACQUIRE_TIMEOUT_MS` | `500` | Timeout in milliseconds waiting for a concurrency slot |

### Telemetry (OpenTelemetry)

| Env Variable | Default | Description |
|---|---|---|
| `TELEMETRY_ENABLED` | `true` | Enable OpenTelemetry metrics and tracing |
| `SERVICE_NAME` | `azs3-proxy` | OTEL service name resource attribute |
| `SERVICE_VERSION` | `1.0.0` | OTEL service version resource attribute |
| `ENVIRONMENT` | `development` | OTEL environment resource attribute |
| `TELEMETRY_EXPORT_TYPE` | `otlp` | Telemetry export type |
| `TELEMETRY_EXPORT_INTERVAL` | `30s` | Metric export interval |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OTLP gRPC endpoint |
| `OTEL_EXPORTER_OTLP_INSECURE` | `false` | Use plaintext gRPC (set `true` for non-TLS) |
| `TELEMETRY_LOG_LEVEL` | `warn` | Log level for OTEL internal logging |

See [TELEMETRY.md](TELEMETRY.md) for exporter examples and Azure Monitor bridging.

## Configuring for Azure Blob Storage

```bash
AZURE_BACKEND_TYPE=blob          # default, can be omitted
AZURE_STORAGE_ACCOUNT=youraccount
AZURE_STORAGE_KEY=yourkey
S3_ACCESS_KEY=your-s3-access-key
S3_SECRET_KEY=your-s3-secret-key
```

| S3 Concept | Azure Blob Mapping |
|---|---|
| Bucket | Blob Container |
| Object | Block Blob |
| Multipart Upload | Staged blocks |
| Versioning | Native blob versioning (when enabled on storage account) |

All six Azure auth modes are supported with the blob backend.

## Configuring for Azure Files

```bash
AZURE_BACKEND_TYPE=file
AZURE_STORAGE_ACCOUNT=youraccount
AZURE_STORAGE_KEY=yourkey
S3_ACCESS_KEY=your-s3-access-key
S3_SECRET_KEY=your-s3-secret-key
```

| S3 Concept | Azure Files Mapping |
|---|---|
| Bucket | File Share |
| Object | File (directories auto-created from path separators) |
| Multipart Upload | Buffered file upload |
| Versioning | Not supported |

**Auth limitation:** Azure Files supports only **Account Key** and **SAS Token** authentication. For MSI, Service Principal, Federated Token, or Azure CLI auth, use the blob backend.

## Request Tracing

The proxy generates GUID-form request IDs and propagates them via `X-Request-ID`, `x-amz-request-id`, and `x-ms-client-request-id` for Azure. No host information is embedded in the ID.

## Examples

### Blob backend with HTTPS and file logging

```bash
export AZURE_BACKEND_TYPE=blob
export AZURE_STORAGE_ACCOUNT=myaccount
export AZURE_STORAGE_KEY=mykey
export S3_ACCESS_KEY=myaccesskey
export S3_SECRET_KEY=mysecretkey
export ENABLE_TLS=true
export TLS_CERT_FILE=/etc/ssl/certs/azs3.crt
export TLS_KEY_FILE=/etc/ssl/private/azs3.key
export LOG_MODE=both
export LOG_FILE=/var/log/azs3-proxy.log
./azs3-proxy --foreground
```

### File backend with SAS auth and cache

```bash
export AZURE_BACKEND_TYPE=file
export AZURE_STORAGE_ACCOUNT=myaccount
export AZURE_STORAGE_SAS_TOKEN="?sv=..."
export S3_ACCESS_KEY=myaccesskey
export S3_SECRET_KEY=mysecretkey
export CACHE_ENABLED=true
export CACHE_PATH=/var/cache/azs3
export CACHE_MAX_SIZE=2147483648   # 2 GiB
export CACHE_TTL=7200              # 2 hours
./azs3-proxy --foreground
```

### Bandwidth-limited deployment

```bash
export AZURE_STORAGE_ACCOUNT=myaccount
export AZURE_STORAGE_KEY=mykey
export S3_ACCESS_KEY=myaccesskey
export S3_SECRET_KEY=mysecretkey
./azs3-proxy --cap-mbps-read 500 --cap-mbps-write 200 --foreground
```

### MSI auth on Azure VM (blob backend)

```bash
export AZURE_STORAGE_ACCOUNT=myaccount
export AZURE_USE_MSI=true
export S3_ACCESS_KEY=myaccesskey
export S3_SECRET_KEY=mysecretkey
./azs3-proxy --foreground
```

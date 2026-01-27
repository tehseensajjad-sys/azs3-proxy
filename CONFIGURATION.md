# Configuration Guide

Authoritative reference for configuring `azs3-proxy`. Use `.env` or environment variables; CLI flags can override select values at runtime.

## Quick Start

Minimal setup (account key auth):

```bash
export LISTEN_ADDR=:8080
export AZURE_STORAGE_ACCOUNT=<account>
export AZURE_STORAGE_KEY=<key>
export S3_ACCESS_KEY=<s3_access_key>
export S3_SECRET_KEY=<s3_secret_key>
export LOG_MODE=file
export LOG_LEVEL=info
```

Start in foreground with pprof:

```bash
./azs3-proxy --foreground --pprof
```

## Server & Runtime

| Setting | Env | Default | Notes |
| --- | --- | --- | --- |
| Listen address | `LISTEN_ADDR` | `:8080` | Override with `--addr` flag.
| TLS enable | `ENABLE_TLS` | `false` | When `true`, `TLS_CERT_FILE` and `TLS_KEY_FILE` required.
| TLS cert | `TLS_CERT_FILE` | – | Path to certificate.
| TLS key | `TLS_KEY_FILE` | – | Path to private key.
| Logging mode | `LOG_MODE` | `console` (defaults to `file` if unset) | `console`, `file`, or `both`; `--log-file` overrides file path.
| Log file | `LOG_FILE` | `azs3-proxy.log` | Auto-filled if not set.
| Log level | `LOG_LEVEL` | `warn` | `debug`, `info`, `warn`, `error`, `crit`; `--log-level` overrides.
| Response debug | `DEBUG_RESPONSES` | `false` | When `true`, logs response headers for all routes (verbose).
| Foreground/daemon | flags | daemon by default | `--foreground` to run attached; `--stop` to stop daemon.
| pprof | flag `--pprof` | disabled | Starts pprof on `localhost:6060`.

## S3 Authentication

| Setting | Env | Required | Notes |
| --- | --- | --- | --- |
| Access key | `S3_ACCESS_KEY` | Yes | Used for SigV4 verification.
| Secret key | `S3_SECRET_KEY` | Yes | Used for SigV4 verification.

## Azure Authentication Modes

Set `AZURE_STORAGE_ACCOUNT` and choose one mode:

| Mode | Required env | Optional env | Description |
| --- | --- | --- | --- |
| Account Key | `AZURE_STORAGE_KEY` | `AZURE_STORAGE_URL`, `AZURE_SUBSCRIPTION_ID` | Simple shared key auth.
| SAS | `AZURE_STORAGE_SAS_TOKEN` | `AZURE_STORAGE_URL` | SAS token auth.
| Managed Identity (MSI) | `AZURE_USE_MSI=true` | `AZURE_CLIENT_ID` (user-assigned), `AZURE_STORAGE_URL` | Uses system or user-assigned identity.
| Service Principal (SPN) | `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID` | `AZURE_OBJECT_ID`, `AZURE_STORAGE_URL`, `AZURE_SUBSCRIPTION_ID` | Client secret auth.
| Federated Token (OIDC) | `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_FEDERATED_TOKEN_FILE` | `AZURE_STORAGE_URL` | OIDC token file; file must exist/readable.
| Azure CLI | `AZURE_USE_CLI_AUTH=true` | `AZURE_STORAGE_URL` | Uses cached `az login` credentials.

If `AZURE_STORAGE_URL` is omitted, it is built as `https://<account>.blob.core.windows.net`; devstore uses Azurite URL automatically.

## Optional Azure Settings

| Setting | Env | Notes |
| --- | --- | --- |
| Subscription ID | `AZURE_SUBSCRIPTION_ID` | Included for logging/telemetry context.
| Tenant ID | `AZURE_TENANT_ID` | Required for SPN/OIDC.
| Object ID | `AZURE_OBJECT_ID` | Optional metadata for SPN.

## Cache

| Setting | Env | Default | Notes |
| --- | --- | --- | --- |
| Enable cache | `CACHE_ENABLED` | `false` | When `true`, local object cache is used.
| Cache path | `CACHE_PATH` | `/tmp/azs3-proxy-cache` | Directory for cached objects.
| Max size (bytes) | `CACHE_MAX_SIZE` | `1073741824` (1 GiB) | Parsed as int64.
| TTL (seconds) | `CACHE_TTL` | `3600` | Time-to-live per entry.

## Telemetry (OpenTelemetry)

| Setting | Env | Default | Notes |
| --- | --- | --- | --- |
| Enable telemetry | `TELEMETRY_ENABLED` | `true` | Disable to remove OTEL overhead.
| Service name | `SERVICE_NAME` | `azs3-proxy` | Resource attribute.
| Service version | `SERVICE_VERSION` | `1.0.0` | Resource attribute.
| Environment | `ENVIRONMENT` | `development` | Resource attribute.
| Export type | `TELEMETRY_EXPORT_TYPE` | `otlp` | Currently OTLP.
| Export interval | `TELEMETRY_EXPORT_INTERVAL` | `30s` | Metric export period.
| OTLP endpoint | `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | gRPC endpoint.
| OTLP insecure | `OTEL_EXPORTER_OTLP_INSECURE` | `false` | Set `true` for plaintext.
| Telemetry log level | `TELEMETRY_LOG_LEVEL` | `warn` | For OTEL internal logging.

See `TELEMETRY.md` for deeper exporter examples and Azure Monitor bridging.

## Request Tracing

The proxy generates GUID-form request IDs and propagates them via `X-Request-ID`, `x-amz-request-id`, and `x-ms-client-request-id` for Azure. No host information is embedded in the ID.

## Examples

### HTTPS with file logging

```bash
export ENABLE_TLS=true
export TLS_CERT_FILE=/etc/ssl/certs/azs3.crt
export TLS_KEY_FILE=/etc/ssl/private/azs3.key
export LOG_MODE=both
export LOG_FILE=/var/log/azs3-proxy.log
./azs3-proxy --foreground
```

### SAS auth with cache enabled

```bash
export AZURE_STORAGE_ACCOUNT=myaccount
export AZURE_STORAGE_SAS_TOKEN="?sv=..."
export S3_ACCESS_KEY=example
export S3_SECRET_KEY=example
export CACHE_ENABLED=true
export CACHE_PATH=/var/cache/azs3
export CACHE_MAX_SIZE=2147483648   # 2 GiB
export CACHE_TTL=7200              # 2 hours
./azs3-proxy --foreground
```

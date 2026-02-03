---
name: Bug report
description: Report a reproducible problem in azs3-proxy
labels: [bug]
assignees: []
---

## Summary
A clear and concise description of the bug.

## Version
- azs3-proxy version/commit: 
- Go version used to build/run: 

## Environment
- Deployment: docker | binary | source | k8s | other (specify)
- OS / distro and version: 
- Kernel version (if Linux): 
- Architecture: amd64 | arm64
- Network topology (optional): same-VM client/proxy | separate hosts | k8s | other

## Azure Storage Details
- Backend type: blob | file
- Storage account SKU/tier (e.g., Standard_LRS, Premium_LRS): 
- Auth method: account key | SAS | MSI | SPN | OIDC | CLI
- Region: 

## Configuration
- Key env vars (mask secrets):
  - AZURE_BACKEND_TYPE=
  - AZURE_STORAGE_ACCOUNT=
  - CACHE_ENABLED=
  - TELEMETRY_ENABLED=
- Flags or config files used:

## Steps to Reproduce
1. 
2. 
3. 

## Expected Behavior
What you expected to happen.

## Actual Behavior
What actually happened (include HTTP status codes or errors if applicable).

## Logs / Output
Paste relevant logs with timestamps. Mask sensitive data.

## Additional Context
Add any other context, links to failing workflow runs, or screenshots that help explain the issue.

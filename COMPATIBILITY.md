# API Compatibility Matrix

This document details the S3 API operations supported by `azs3-proxy` and their compatibility status with Azure Blob Storage.

## Bucket Operations

| S3 Operation | HTTP Method | URI | Supported | Notes |
|--------------|-------------|-----|-----------|-------|
| **ListBuckets** | `GET` | `/` | ✅ Yes | Maps to Azure `ListContainers`. |
| **CreateBucket** | `PUT` | `/{bucket}` | ✅ Yes | Maps to Azure `CreateContainer`. |
| **DeleteBucket** | `DELETE` | `/{bucket}` | ✅ Yes | Maps to Azure `DeleteContainer`. Must be empty. |
| **GetBucketVersioning** | `GET` | `/{bucket}?versioning` | ✅ Yes | Checks if Azure Blob Versioning is enabled. |
| **PutBucketVersioning** | `PUT` | `/{bucket}?versioning` | ✅ Yes | Enables/Disables Azure Blob Versioning. |
| **ListMultipartUploads** | `GET` | `/{bucket}?uploads` | ✅ Yes | Lists in-progress multipart uploads (tracked in memory/metadata). |

## Object Operations

| S3 Operation | HTTP Method | URI | Supported | Notes |
|--------------|-------------|-----|-----------|-------|
| **PutObject** | `PUT` | `/{bucket}/{key}` | ✅ Yes | Maps to Azure `UploadStream`. |
| **GetObject** | `GET` | `/{bucket}/{key}` | ✅ Yes | Maps to Azure `DownloadStream`. Supports Range requests. |
| **HeadObject** | `HEAD` | `/{bucket}/{key}` | ✅ Yes | Maps to Azure `GetProperties`. |
| **DeleteObject** | `DELETE` | `/{bucket}/{key}` | ✅ Yes | Maps to Azure `DeleteBlob`. |
| **ListObjectsV2** | `GET` | `/{bucket}?list-type=2` | ✅ Yes | Maps to Azure `ListBlobsFlat`. |
| **CopyObject** | `PUT` | `/{bucket}/{key}` | ✅ Yes | Supports `x-amz-copy-source` header. Implemented via download-upload (naive copy). |

## Multipart Upload Operations

| S3 Operation | HTTP Method | URI | Supported | Notes |
|--------------|-------------|-----|-----------|-------|
| **InitiateMultipartUpload** | `POST` | `/{bucket}/{key}?uploads` | ✅ Yes | Generates an Upload ID. |
| **UploadPart** | `PUT` | `/{bucket}/{key}?partNumber&uploadId` | ✅ Yes | Maps to Azure `StageBlock`. |
| **CompleteMultipartUpload** | `POST` | `/{bucket}/{key}?uploadId` | ✅ Yes | Maps to Azure `CommitBlockList`. |
| **AbortMultipartUpload** | `DELETE` | `/{bucket}/{key}?uploadId` | ✅ Yes | Cleans up staged blocks. |
| **ListParts** | `GET` | `/{bucket}/{key}?uploadId` | ✅ Yes | Lists staged blocks for an upload ID. |

## Object Versioning Operations

| S3 Operation | HTTP Method | URI | Supported | Notes |
|--------------|-------------|-----|-----------|-------|
| **ListObjectVersions** | `GET` | `/{bucket}?versions` | ✅ Yes | Lists all blob versions. |
| **GetObjectVersion** | `GET` | `/{bucket}/{key}?versionId` | ✅ Yes | Retrieves a specific snapshot/version. |
| **DeleteObjectVersion** | `DELETE` | `/{bucket}/{key}?versionId` | ✅ Yes | Deletes a specific snapshot/version. |

## Unsupported / Planned Features

- **ACLs**: `x-amz-acl` headers are currently ignored.
- **Lifecycle Policies**: No mapping to Azure Lifecycle Management yet.
- **Presigned URLs**: Not yet implemented.
- **Tagging**: Object tagging is not yet supported.
- **Metadata**: Custom user metadata (`x-amz-meta-*`) is partially supported but not fully validated.

## Error Handling

The proxy attempts to map Azure Blob Storage errors to their closest S3 equivalents:

- `ContainerNotFound` -> `NoSuchBucket`
- `BlobNotFound` -> `NoSuchKey`
- `ContainerAlreadyExists` -> `BucketAlreadyExists`
- `AuthorizationPermissionMismatch` -> `AccessDenied`

## Azure Authentication Support

The proxy supports multiple methods for authenticating with Azure Blob Storage:

| Method | Config Mode | Description |
|--------|-------------|-------------|
| **Account Key** | `account_key` | Uses Storage Account Name and Key. Simple but less secure. |
| **SAS Token** | `sas` | Uses Shared Access Signature token. Granular access control. |
| **Managed Identity** | `msi` | Uses Azure Managed Identity (System or User Assigned). Recommended for Azure deployments. |
| **Service Principal** | `spn` | Uses Client ID and Secret. Good for external applications. |
| **Workload Identity** | `federated_token` | Uses Federated Identity (OIDC). Ideal for Kubernetes (AKS). |
| **Azure CLI** | `az_cli` | Uses local Azure CLI credentials. Best for local development. |

## Storage Account Compatibility

| Account Type | Supported | Notes |
|--------------|-----------|-------|
| **Standard General Purpose v2** | ✅ Yes | Native support via Blob API. |
| **Premium Block Blob** | ✅ Yes | Native support via Blob API. |
| **Data Lake Storage Gen2 (HNS)** | ✅ Yes | Supported via **Blob API endpoint** (`blob.core.windows.net`). Multi-protocol access allows S3 operations to work on HNS accounts, but they are treated as a flat namespace. The `dfs` endpoint is not used. |
| **Blob Storage (Legacy)** | ✅ Yes | Supported. |



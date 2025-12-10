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

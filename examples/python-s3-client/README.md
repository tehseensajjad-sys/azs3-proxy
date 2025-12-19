# Python S3 Client Example

This example demonstrates how to use the standard Python `boto3` library to interact with the `azs3-proxy`.

## Prerequisites

- Python 3.x installed
- `azs3-proxy` running locally (default: `http://localhost:8080`)

## Setup

1. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```

## Configuration

The script uses the following default credentials, which match the examples in the main README. If you changed them when starting the proxy, export the corresponding environment variables:

```bash
export S3_ACCESS_KEY=AKIA1234567890ABCDEF
export S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY
export PROXY_URL=http://localhost:8080
export BUCKET_NAME=test-bucket-python # Optional: Defaults to test-bucket-python
```

### Azure Authentication (SAS Token)

If you want to use **Shared Access Signature (SAS)** authentication instead of an Account Key, you need to export the following environment variables before starting the proxy:

```bash
# Required for SAS Auth
export AZURE_STORAGE_ACCOUNT=youraccountname
export AZURE_STORAGE_SAS_TOKEN="sv=2022-11-02&ss=b&srt=sco&sp=rwdlacx&se=2025-01-01T00:00:00Z&st=2024-01-01T00:00:00Z&spr=https&sig=..."

# Optional: Override default Azure Blob URL
# export AZURE_STORAGE_URL=https://youraccountname.blob.core.windows.net
```

## Running the Tests

1. Ensure your proxy is running and connected to Azure (or Azurite).
   ```bash
   # In the root of the project
   make run
   ```

2. Run the comprehensive test script:
   ```bash
   python3 client.py
   ```

## Available Scripts

We provide individual scripts for specific S3 operations to make testing easier:

### Bucket Operations
- **`list_buckets.py`**: Lists all available buckets.
- **`check_bucket_exists.py`**: Checks if a specific bucket exists.
- **`create_bucket.py`**: Creates a new bucket (idempotent).
- **`delete_bucket.py`**: Deletes an empty bucket.

### Object Operations
- **`upload_file.py`**: Uploads a file to a bucket.
- **`read_file.py`**: Reads and prints the content of a file.
- **`delete_file.py`**: Deletes a file from a bucket.
- **`rename_file.py`**: Renames a file (Copy + Delete).
- **`multipart_upload.py`**: Demonstrates a multipart upload (15MB file).

### Directory Operations
- **`create_directory.py`**: Creates a directory (object with trailing slash).
- **`delete_directory.py`**: Deletes a directory and all its contents.
- **`rename_directory.py`**: Renames a directory (moves all objects inside).

### Common
- **`common.py`**: Shared configuration and S3 client initialization logic.

You can run any of these scripts individually:
```bash
python3 list_buckets.py
python3 upload_file.py
```
   python client.py
   ```

## Verification

If successful, the script will:
1. Generate 1MB of random data.
2. Upload it as `random_1mb.bin` to the specified bucket.
3. Read the file back and verify the content matches exactly.

You can then verify the file exists in your Azure Storage Account (container `test-bucket-python` or your custom name) via the Azure Portal.

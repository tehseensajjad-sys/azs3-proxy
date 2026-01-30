import argparse
import concurrent.futures
import os
import sys
import time
from io import BytesIO

import boto3
from botocore.config import Config
from botocore.exceptions import ClientError


def ensure_bucket(s3, bucket):
    try:
        s3.head_bucket(Bucket=bucket)
        return
    except ClientError as exc:  # bucket may not exist
        error_code = exc.response.get("Error", {}).get("Code", "")
        if error_code not in ("404", "NoSuchBucket"):
            raise
    s3.create_bucket(Bucket=bucket)


def upload_object(s3, bucket, key, payload):
    # Each call uses a new BytesIO to avoid shared cursor
    s3.put_object(Bucket=bucket, Key=key, Body=BytesIO(payload))


def list_all_v2(s3, bucket, prefix, page_size):
    total = 0
    token = None
    while True:
        params = {"Bucket": bucket, "Prefix": prefix, "MaxKeys": page_size}
        if token:
            params["ContinuationToken"] = token
        resp = s3.list_objects_v2(**params)
        total += resp.get("KeyCount", 0)
        token = resp.get("NextContinuationToken")
        if not resp.get("IsTruncated"):
            break
    return total


def main():
    parser = argparse.ArgumentParser(description="Load test ListObjectsV2 with many 1MB objects")
    parser.add_argument("--bucket", default=os.getenv("BUCKET_NAME", "listv2-loadtest"))
    parser.add_argument("--prefix", default=os.getenv("PREFIX", "loadtest/"))
    parser.add_argument("--count", type=int, default=int(os.getenv("COUNT", "25000")))
    parser.add_argument("--size-mb", type=int, default=int(os.getenv("SIZE_MB", "1")))
    parser.add_argument("--concurrency", type=int, default=int(os.getenv("CONCURRENCY", "32")))
    parser.add_argument("--page-size", type=int, default=int(os.getenv("PAGE_SIZE", "1000")))
    args = parser.parse_args()

    payload = os.urandom(args.size_mb * 1024 * 1024)

    endpoint = os.getenv("AWS_ENDPOINT_URL") or os.getenv("PROXY_URL") or "http://localhost:8080"
    region = os.getenv("AWS_REGION") or os.getenv("AWS_DEFAULT_REGION") or "us-east-1"

    s3 = boto3.client(
        "s3",
        endpoint_url=endpoint,
        region_name=region,
        aws_access_key_id=os.getenv("AWS_ACCESS_KEY_ID") or os.getenv("S3_ACCESS_KEY"),
        aws_secret_access_key=os.getenv("AWS_SECRET_ACCESS_KEY") or os.getenv("S3_SECRET_KEY"),
        config=Config(s3={"addressing_style": os.getenv("AWS_S3_ADDRESSING_STYLE", "path")}),
    )

    print(f"Endpoint: {endpoint}")
    print(f"Bucket:   {args.bucket}")
    print(f"Objects:  {args.count} @ {args.size_mb}MB each")
    print(f"Prefix:   {args.prefix}")
    sys.stdout.flush()

    ensure_bucket(s3, args.bucket)

    keys = [f"{args.prefix}obj-{i:05d}.bin" for i in range(args.count)]

    start = time.time()
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.concurrency) as pool:
        futures = [pool.submit(upload_object, s3, args.bucket, key, payload) for key in keys]
        for idx, fut in enumerate(concurrent.futures.as_completed(futures), 1):
            fut.result()  # raise if error
            if idx % 1000 == 0:
                elapsed = time.time() - start
                print(f"Uploaded {idx}/{args.count} in {elapsed:.1f}s")
                sys.stdout.flush()
    elapsed_upload = time.time() - start
    print(f"Upload complete in {elapsed_upload:.1f}s ({args.count} objects)")

    # Validate ListObjectsV2 pagination
    list_start = time.time()
    total = list_all_v2(s3, args.bucket, args.prefix, args.page_size)
    elapsed_list = time.time() - list_start
    print(f"ListObjectsV2 returned {total} keys in {elapsed_list:.1f}s (page size {args.page_size})")

    if total != args.count:
        print(f"Mismatch: expected {args.count}, got {total}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()

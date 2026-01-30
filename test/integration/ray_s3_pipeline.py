#!/usr/bin/env python3
"""
Ray-to-S3 sample that exercises the proxy with Azure storage behind it.

What it does
- Initializes Ray locally.
- Ensures a target S3 bucket exists via the proxy endpoint.
- Writes a Ray Dataset to S3 (parquet), reads it back, and validates counts.
- Runs a Ray remote task that simulates training and uploads metrics/checkpoint objects to S3.

Prereqs
- Proxy running locally on port 8080 and pointing at Azure.
- Python deps: pip install -r test/integration/requirements-ray.txt
- Environment vars (aligns with existing .env/code):
    export S3_ACCESS_KEY=test-access-key
    export S3_SECRET_KEY=test-secret-key
    export PROXY_URL=http://localhost:8080
    export BUCKET_NAME=ray-s3-proxy-demo   # optional; defaults to this value
    export AWS_REGION=us-east-1            # optional; defaults to us-east-1
  (AWS_* variables are derived automatically from the above.)

Run
    python test/integration/ray_s3_pipeline.py

If you prefer Ray Jobs:
    ray job submit -- python test/integration/ray_s3_pipeline.py
"""

import json
import os
from typing import Tuple
from urllib.parse import urlparse
import io

import boto3
from botocore.config import Config
import ray


def _env(key: str, default: str) -> str:
    return os.environ.get(key, default)


def build_s3_client(endpoint: str, region: str, access_key: str, secret_key: str):
    return boto3.client(
        "s3",
        endpoint_url=endpoint,
        region_name=region,
        aws_access_key_id=access_key,
        aws_secret_access_key=secret_key,
        config=Config(s3={"addressing_style": "path"}, retries={"max_attempts": 3, "mode": "standard"}),
    )


def ensure_bucket(client, bucket: str, region: str):
    try:
        client.head_bucket(Bucket=bucket)
        return
    except Exception:
        pass

    create_args = {"Bucket": bucket}
    if region != "us-east-1":
        create_args["CreateBucketConfiguration"] = {"LocationConstraint": region}
    client.create_bucket(**create_args)


def dataset_roundtrip(bucket: str, endpoint: str, region: str, access_key: str, secret_key: str) -> Tuple[int, int]:
    client = build_s3_client(endpoint, region, access_key, secret_key)
    key = "ray-data/demo/dataset.csv"

    ds = ray.data.range(200).materialize()
    pdf = ds.to_pandas()

    buffer = io.BytesIO()
    pdf.to_csv(buffer, index=False)
    buffer.seek(0)

    client.put_object(Bucket=bucket, Key=key, Body=buffer.getvalue())

    resp = client.get_object(Bucket=bucket, Key=key)
    with resp["Body"] as body_stream:
        pdf_back = None
        try:
            import pandas as pd
        except ImportError as exc:  # pragma: no cover
            raise SystemExit(f"pandas is required for dataset validation: {exc}")
        pdf_back = pd.read_csv(body_stream)

    return len(pdf), len(pdf_back)


@ray.remote
def train_and_checkpoint(bucket: str, endpoint: str, region: str, access_key: str, secret_key: str):
    client = build_s3_client(endpoint, region, access_key, secret_key)

    metrics = {"loss": 0.123, "accuracy": 0.987}
    client.put_object(
        Bucket=bucket,
        Key="ray-checkpoints/metrics.json",
        Body=json.dumps(metrics).encode("utf-8"),
    )
    client.put_object(
        Bucket=bucket,
        Key="ray-checkpoints/model.chkpt",
        Body=b"dummy checkpoint bytes",
    )

    return metrics


def main():
    endpoint = _env("AWS_ENDPOINT_URL", _env("PROXY_URL", "http://localhost:8080"))
    bucket = _env("BUCKET_NAME", "ray-s3-proxy-demo")
    region = _env("AWS_REGION", _env("AWS_DEFAULT_REGION", "us-east-1"))

    access_key = _env("AWS_ACCESS_KEY_ID", _env("S3_ACCESS_KEY", ""))
    secret_key = _env("AWS_SECRET_ACCESS_KEY", _env("S3_SECRET_KEY", ""))
    if not access_key or not secret_key:
        raise SystemExit("S3_ACCESS_KEY/S3_SECRET_KEY (or AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY) must be set to proxy credentials")

    # Ensure Ray and data libraries use the proxy endpoint and credentials
    os.environ["AWS_ENDPOINT_URL"] = endpoint
    os.environ["AWS_ACCESS_KEY_ID"] = access_key
    os.environ["AWS_SECRET_ACCESS_KEY"] = secret_key
    os.environ.setdefault("AWS_DEFAULT_REGION", region)
    os.environ.setdefault("AWS_REGION", region)

    ray.init()

    s3_client = build_s3_client(endpoint, region, access_key, secret_key)
    ensure_bucket(s3_client, bucket, region)

    written, read_back = dataset_roundtrip(bucket, endpoint, region, access_key, secret_key)
    if written != read_back:
        raise RuntimeError(f"Dataset count mismatch: wrote {written}, read {read_back}")

    metrics = ray.get(train_and_checkpoint.remote(bucket, endpoint, region, access_key, secret_key))

    print("Ray S3 proxy validation succeeded")
    print(f"Endpoint: {endpoint}")
    print(f"Bucket:   {bucket}")
    print(f"Dataset rows written/read: {written}/{read_back}")
    print(f"Checkpoint metrics: {metrics}")


if __name__ == "__main__":
    main()

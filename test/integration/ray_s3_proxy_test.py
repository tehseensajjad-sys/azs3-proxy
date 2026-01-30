import os
import shutil
import tempfile
import time
from urllib.parse import urlparse

import ray
import ray.cloudpickle as pickle
import boto3
from ray import tune, air
from ray.air import session
from ray.train import Checkpoint, SyncConfig


LOCAL_STORAGE_PATH = os.getenv("LOCAL_STORAGE_PATH", "file:///tmp/ray-results/")


BUCKET_NAME = os.getenv("BUCKET_NAME", "testcntgo9")
RAW_UPLOAD_DIR = os.getenv("RAY_UPLOAD_DIR", f"s3://{BUCKET_NAME}/ray-results/")
PROXY_URL = os.getenv("PROXY_URL")
AWS_REGION = os.getenv("AWS_REGION", os.getenv("AWS_DEFAULT_REGION", "us-east-1"))
S3_ACCESS_KEY = os.getenv("S3_ACCESS_KEY")
S3_SECRET_KEY = os.getenv("S3_SECRET_KEY")


def _build_storage_path():
    """Prefer explicit endpoint override so pyarrow uses the proxy."""
    if not PROXY_URL:
        return RAW_UPLOAD_DIR

    parsed = urlparse(PROXY_URL)
    host = parsed.hostname or "localhost"
    port = parsed.port or (443 if parsed.scheme == "https" else 80)
    endpoint_override = f"{host}:{port}"
    scheme = parsed.scheme or "http"

    # pyarrow S3 URI accepts query params for endpoint override and bucket creation
    return (
        f"s3://{BUCKET_NAME}/ray-results/"
        f"?endpoint_override={endpoint_override}"
        f"&scheme={scheme}"
        f"&allow_bucket_creation=true"
    )


STORAGE_PATH = _build_storage_path()


def _s3_client():
    if not PROXY_URL:
        return None

    return boto3.client(
        "s3",
        endpoint_url=PROXY_URL,
        aws_access_key_id=S3_ACCESS_KEY,
        aws_secret_access_key=S3_SECRET_KEY,
        region_name=AWS_REGION,
        config=boto3.session.Config(s3={"addressing_style": "path"}),
    )


def upload_checkpoint_dir(checkpoint_dir: str, prefix: str):
    s3 = _s3_client()
    if not s3:
        return

    try:
        s3.create_bucket(Bucket=BUCKET_NAME)
    except s3.exceptions.BucketAlreadyOwnedByYou:
        pass
    except s3.exceptions.BucketAlreadyExists:
        pass

    for root, _, files in os.walk(checkpoint_dir):
        for name in files:
            full_path = os.path.join(root, name)
            rel_path = os.path.relpath(full_path, checkpoint_dir)
            key = f"{prefix}/{rel_path}"
            s3.upload_file(full_path, BUCKET_NAME, key)

def train_fn(config):
    checkpoint_root = tempfile.mkdtemp(prefix="ray-checkpoint-")

    for i in range(5):
        time.sleep(1)

        # Small checkpoint (metadata + object PUT)
        checkpoint_dir = tempfile.mkdtemp(dir=checkpoint_root)

        payload = {
            "step": i,
            "payload": "checkpoint-data",
        }

        data_path = os.path.join(checkpoint_dir, "data.pkl")
        with open(data_path, "wb") as fp:
            pickle.dump(payload, fp)

        checkpoint = Checkpoint.from_directory(checkpoint_dir)

        trial_name = getattr(session, "get_trial_name", lambda: str(os.getpid()))()
        prefix = f"ray-results/s3-proxy-test/{trial_name}/checkpoint_{i:06d}"
        upload_checkpoint_dir(checkpoint_dir, prefix)

        session.report(
            metrics={"loss": 10 - i},
            checkpoint=checkpoint,
        )

    # Best-effort cleanup after checkpoints have been persisted
    shutil.rmtree(checkpoint_root, ignore_errors=True)

def run_tuner(storage_path: str):
    tuner = tune.Tuner(
        train_fn,
        run_config=air.RunConfig(
            name="s3-proxy-test",
            storage_path=storage_path,
            # Deprecated but required for S3 sync without pyarrow's zero-byte validation PUT
            sync_config=SyncConfig(
                upload_dir=STORAGE_PATH,
                sync_on_checkpoint=True,
            ),
        ),
        tune_config=tune.TuneConfig(num_samples=2),
    )

    tuner.fit()

def put_marker_object():
    if not PROXY_URL:
        return

    s3 = boto3.client(
        "s3",
        endpoint_url=PROXY_URL,
        aws_access_key_id=os.getenv("S3_ACCESS_KEY"),
        aws_secret_access_key=os.getenv("S3_SECRET_KEY"),
        region_name=AWS_REGION,
        config=boto3.session.Config(s3={"addressing_style": "path"}),
    )

    try:
        s3.create_bucket(Bucket=BUCKET_NAME)
    except s3.exceptions.BucketAlreadyOwnedByYou:
        pass
    except s3.exceptions.BucketAlreadyExists:
        pass

    s3.put_object(Bucket=BUCKET_NAME, Key="ray-results/proxy-marker.txt", Body=b"ok")


if __name__ == "__main__":
    ray.init(num_cpus=2)
    run_tuner(LOCAL_STORAGE_PATH)
    put_marker_object()
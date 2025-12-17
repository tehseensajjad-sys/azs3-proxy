import boto3
import os

def get_s3_client():
    access_key = os.environ.get('S3_ACCESS_KEY', 'AKIA1234567890ABCDEF')
    secret_key = os.environ.get('S3_SECRET_KEY', 'wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY')
    proxy_url = os.environ.get('PROXY_URL', 'http://localhost:8080')

    return boto3.client(
        's3',
        endpoint_url=proxy_url,
        aws_access_key_id=access_key,
        aws_secret_access_key=secret_key,
        region_name='us-east-1'
    )

def get_bucket_name():
    return os.environ.get('BUCKET_NAME', 'test-bucket-python')

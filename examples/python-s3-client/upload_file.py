import os
from common import get_s3_client, get_bucket_name

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    file_name = "test-file.txt"
    content = b"Hello, S3 Proxy!"
    
    print(f"Uploading file '{file_name}' to bucket '{bucket_name}'...")
    try:
        s3.put_object(Bucket=bucket_name, Key=file_name, Body=content)
        print(f"✅ File '{file_name}' uploaded successfully.")
    except Exception as e:
        print(f"❌ Error uploading file: {e}")

if __name__ == "__main__":
    main()

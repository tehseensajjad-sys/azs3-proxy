import boto3
import os
import sys
from botocore.exceptions import ClientError

def generate_random_data(size_in_bytes):
    """Generates random byte data of specified size."""
    return os.urandom(size_in_bytes)

def main():
    # Configuration
    access_key = os.environ.get('S3_ACCESS_KEY', 'AKIA1234567890ABCDEF')
    secret_key = os.environ.get('S3_SECRET_KEY', 'wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY')
    proxy_url = os.environ.get('PROXY_URL', 'http://localhost:8080')
    
    bucket_name = os.environ.get('BUCKET_NAME', 'test-bucket-python')
    file_name = "random_1mb.bin"
    data_size = 1024 * 1024 # 1MB

    print(f"Connecting to S3 Proxy at {proxy_url}...")
    
    s3 = boto3.client(
        's3',
        endpoint_url=proxy_url,
        aws_access_key_id=access_key,
        aws_secret_access_key=secret_key,
        region_name='us-east-1'
    )

    try:
        # 1. List all buckets
        print("\n--- Step 1: List all buckets ---")
        response = s3.list_buckets()
        buckets = [b['Name'] for b in response.get('Buckets', [])]
        print("Buckets found:")
        for b in buckets:
            print(f" - {b}")

        # 2. Check bucket exists or not
        print(f"\n--- Step 2: Check if bucket '{bucket_name}' exists ---")
        bucket_exists = bucket_name in buckets
        if bucket_exists:
            print(f"✅ Bucket '{bucket_name}' exists.")
        else:
            print(f"ℹ️  Bucket '{bucket_name}' does not exist.")

        # 3. If not create the bucket
        if not bucket_exists:
            print(f"\n--- Step 3: Creating bucket '{bucket_name}' ---")
            try:
                s3.create_bucket(Bucket=bucket_name)
                print(f"✅ Bucket '{bucket_name}' created successfully.")
            except ClientError as e:
                print(f"❌ Failed to create bucket: {e}")
                sys.exit(1)
        else:
            print(f"\n--- Step 3: Skipping bucket creation (already exists) ---")

        # 4 & 5. Create file and write 1MB data
        print(f"\n--- Step 4 & 5: Create file '{file_name}' and write 1MB data ---")
        data = generate_random_data(data_size)
        s3.put_object(Bucket=bucket_name, Key=file_name, Body=data)
        print(f"✅ File '{file_name}' uploaded successfully.")

        # 6. Close file (Implicit in put_object)

        # 7. Again open the file and read the file
        print(f"\n--- Step 7: Read file '{file_name}' back ---")
        response = s3.get_object(Bucket=bucket_name, Key=file_name)
        read_data = response['Body'].read()
        print(f"Read {len(read_data)} bytes.")
        
        if len(read_data) == data_size and read_data == data:
            print("✅ Data verification successful.")
        else:
            print("❌ Data verification failed.")

        # 8. List all objects in container
        print(f"\n--- Step 8: List all objects in bucket '{bucket_name}' ---")
        response = s3.list_objects_v2(Bucket=bucket_name)
        if 'Contents' in response:
            for obj in response['Contents']:
                print(f" - {obj['Key']} (Size: {obj['Size']} bytes)")
        else:
            print("No objects found.")

        print("\n🎉 Test completed successfully!")

    except Exception as e:
        print(f"\n❌ Error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()

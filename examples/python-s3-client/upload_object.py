import os
from common import get_s3_client, get_bucket_name

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    file_name = "random_1mb.bin"
    data_size = 1024 * 1024
    
    print(f"Generating {data_size} bytes...")
    data = os.urandom(data_size)
    
    print(f"Uploading to '{bucket_name}/{file_name}'...")
    try:
        s3.put_object(Bucket=bucket_name, Key=file_name, Body=data)
        print("Upload successful.")
    except Exception as e:
        print(f"Error uploading: {e}")

if __name__ == "__main__":
    main()

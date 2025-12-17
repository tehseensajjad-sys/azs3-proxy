from common import get_s3_client, get_bucket_name

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    dir_name = "test-directory/"
    
    print(f"Creating directory '{dir_name}' in bucket '{bucket_name}'...")
    try:
        s3.put_object(Bucket=bucket_name, Key=dir_name)
        print(f"✅ Directory '{dir_name}' created successfully.")
    except Exception as e:
        print(f"❌ Error creating directory: {e}")

if __name__ == "__main__":
    main()

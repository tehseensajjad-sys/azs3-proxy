from common import get_s3_client, get_bucket_name

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    file_name = "test-file.txt"
    
    print(f"Reading file '{file_name}' from bucket '{bucket_name}'...")
    try:
        response = s3.get_object(Bucket=bucket_name, Key=file_name)
        data = response['Body'].read()
        print(f"✅ Read content: {data.decode('utf-8')}")
    except Exception as e:
        print(f"❌ Error reading file: {e}")

if __name__ == "__main__":
    main()

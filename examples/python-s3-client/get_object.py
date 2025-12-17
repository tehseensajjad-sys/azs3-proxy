from common import get_s3_client, get_bucket_name

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    file_name = "random_1mb.bin"
    
    print(f"Downloading '{bucket_name}/{file_name}'...")
    try:
        response = s3.get_object(Bucket=bucket_name, Key=file_name)
        data = response['Body'].read()
        print(f"Downloaded {len(data)} bytes.")
    except Exception as e:
        print(f"Error downloading: {e}")

if __name__ == "__main__":
    main()

from common import get_s3_client, get_bucket_name

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    file_name = "test-file.txt"
    
    print(f"Deleting file '{file_name}' from bucket '{bucket_name}'...")
    try:
        s3.delete_object(Bucket=bucket_name, Key=file_name)
        print(f"✅ File '{file_name}' deleted successfully.")
    except Exception as e:
        print(f"❌ Error deleting file: {e}")

if __name__ == "__main__":
    main()

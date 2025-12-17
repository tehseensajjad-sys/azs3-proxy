from common import get_s3_client, get_bucket_name

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    dir_name = "test-directory/"
    
    print(f"Deleting directory '{dir_name}' and its contents in bucket '{bucket_name}'...")
    try:
        # List objects with prefix
        response = s3.list_objects_v2(Bucket=bucket_name, Prefix=dir_name)
        if 'Contents' in response:
            objects = [{'Key': obj['Key']} for obj in response['Contents']]
            print(f"Found {len(objects)} objects to delete.")
            
            # Delete objects
            s3.delete_objects(Bucket=bucket_name, Delete={'Objects': objects})
            print(f"✅ Directory '{dir_name}' deleted successfully.")
        else:
            print(f"ℹ️  Directory '{dir_name}' not found or empty.")
            
    except Exception as e:
        print(f"❌ Error deleting directory: {e}")

if __name__ == "__main__":
    main()

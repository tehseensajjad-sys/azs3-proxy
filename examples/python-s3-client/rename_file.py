from common import get_s3_client, get_bucket_name

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    old_name = "test-file.txt"
    new_name = "renamed-file.txt"
    
    print(f"Renaming '{old_name}' to '{new_name}' in bucket '{bucket_name}'...")
    try:
        # Copy object
        print(f"Copying '{old_name}' to '{new_name}'...")
        s3.copy_object(Bucket=bucket_name, CopySource={'Bucket': bucket_name, 'Key': old_name}, Key=new_name)
        
        # Delete old object
        print(f"Deleting original file '{old_name}'...")
        s3.delete_object(Bucket=bucket_name, Key=old_name)
        
        print(f"✅ Renamed successfully.")
    except Exception as e:
        print(f"❌ Error renaming file: {e}")

if __name__ == "__main__":
    main()

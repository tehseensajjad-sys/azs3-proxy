from common import get_s3_client, get_bucket_name

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    old_dir = "test-directory/"
    new_dir = "renamed-directory/"
    
    print(f"Renaming directory '{old_dir}' to '{new_dir}' in bucket '{bucket_name}'...")
    try:
        # List objects with prefix
        response = s3.list_objects_v2(Bucket=bucket_name, Prefix=old_dir)
        if 'Contents' in response:
            objects_to_delete = []
            for obj in response['Contents']:
                old_key = obj['Key']
                new_key = old_key.replace(old_dir, new_dir, 1)
                
                print(f"Copying '{old_key}' to '{new_key}'...")
                s3.copy_object(Bucket=bucket_name, CopySource={'Bucket': bucket_name, 'Key': old_key}, Key=new_key)
                objects_to_delete.append({'Key': old_key})
            
            # Delete old objects
            print(f"Deleting {len(objects_to_delete)} old objects...")
            s3.delete_objects(Bucket=bucket_name, Delete={'Objects': objects_to_delete})
            print(f"✅ Directory renamed successfully.")
        else:
            print(f"ℹ️  Directory '{old_dir}' not found or empty.")
            
    except Exception as e:
        print(f"❌ Error renaming directory: {e}")

if __name__ == "__main__":
    main()

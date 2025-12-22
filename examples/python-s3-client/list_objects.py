from common import get_s3_client, get_bucket_name

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    
    try:
        print(f"Listing objects in bucket '{bucket_name}'...")
        response = s3.list_objects_v2(Bucket=bucket_name)
        
        if 'Contents' in response:
            print("\nFiles:")
            for obj in response['Contents']:
                print(f"- {obj['Key']} (Size: {obj['Size']} bytes, Last Modified: {obj['LastModified']})")
        else:
            print("No files found.")
            
        # CommonPrefixes contains directories if a delimiter is used, but list_objects_v2 without delimiter lists everything.
        # If we want to mimic "directories", we might want to use a delimiter, but "list all files and directories" usually implies a flat list or a recursive walk.
        # Given the prompt "list all files and directories", a flat list of keys is the most direct interpretation for S3.
        # However, if the user wants to see "directories" explicitly, they might exist as 0-byte objects ending in /.
        
    except Exception as e:
        print(f"Error listing objects: {e}")

if __name__ == "__main__":
    main()

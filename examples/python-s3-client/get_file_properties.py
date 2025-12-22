from common import get_s3_client, get_bucket_name
import json

def default_serializer(obj):
    """JSON serializer for objects not serializable by default json code"""
    return str(obj)

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    
    key = input("Enter file key (e.g., folder/file.txt): ").strip()
    if not key:
        print("File key is required.")
        return

    try:
        print(f"Getting properties for file '{key}' in bucket '{bucket_name}'...")
        response = s3.head_object(Bucket=bucket_name, Key=key)
        
        # Remove ResponseMetadata for cleaner output, or keep it if detailed debugging is needed
        if 'ResponseMetadata' in response:
            del response['ResponseMetadata']
            
        print(json.dumps(response, indent=4, default=default_serializer))
        
    except Exception as e:
        print(f"Error getting file properties: {e}")

if __name__ == "__main__":
    main()

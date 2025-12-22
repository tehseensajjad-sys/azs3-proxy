from common import get_s3_client, get_bucket_name
import json

def default_serializer(obj):
    """JSON serializer for objects not serializable by default json code"""
    return str(obj)

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    
    key = input("Enter folder key (e.g., myfolder/): ").strip()
    if not key:
        print("Folder key is required.")
        return
        
    if not key.endswith('/'):
        print("Note: Folder keys usually end with '/'. Appending it.")
        key += '/'

    try:
        print(f"Getting properties for folder '{key}' in bucket '{bucket_name}'...")
        response = s3.head_object(Bucket=bucket_name, Key=key)
        
        if 'ResponseMetadata' in response:
            del response['ResponseMetadata']
            
        print(json.dumps(response, indent=4, default=default_serializer))
        
    except Exception as e:
        print(f"Error getting folder properties: {e}")
        print("Note: In S3, folders are 0-byte objects. If this fails, the folder object might not explicitly exist, even if files exist under this prefix.")

if __name__ == "__main__":
    main()

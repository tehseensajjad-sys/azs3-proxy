from common import get_s3_client, get_bucket_name
from botocore.exceptions import ClientError

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    
    print(f"Attempting to delete bucket '{bucket_name}'...")
    try:
        s3.delete_bucket(Bucket=bucket_name)
        print(f"✅ Bucket '{bucket_name}' deleted successfully.")
    except ClientError as e:
        if e.response['Error']['Code'] == 'NoSuchBucket':
            print(f"ℹ️  Bucket '{bucket_name}' does not exist.")
        elif e.response['Error']['Code'] == 'BucketNotEmpty':
            print(f"❌ Bucket '{bucket_name}' is not empty. Please empty it first.")
        else:
            print(f"❌ Error deleting bucket: {e}")

if __name__ == "__main__":
    main()

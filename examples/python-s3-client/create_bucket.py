from common import get_s3_client, get_bucket_name
from botocore.exceptions import ClientError

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    
    print(f"Attempting to create bucket '{bucket_name}' if it doesn't exist...")
    try:
        s3.create_bucket(Bucket=bucket_name)
        print(f"✅ Bucket '{bucket_name}' created successfully.")
    except ClientError as e:
        if e.response['Error']['Code'] == 'BucketAlreadyOwnedByYou':
            print(f"ℹ️  Bucket '{bucket_name}' already exists and is owned by you.")
        elif e.response['Error']['Code'] == 'BucketAlreadyExists':
             print(f"❌ Bucket '{bucket_name}' already exists (owned by someone else).")
        else:
            print(f"❌ Error creating bucket: {e}")

if __name__ == "__main__":
    main()

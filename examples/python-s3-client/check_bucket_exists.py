from common import get_s3_client, get_bucket_name
from botocore.exceptions import ClientError

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    
    print(f"Checking if bucket '{bucket_name}' exists...")
    try:
        s3.head_bucket(Bucket=bucket_name)
        print(f"✅ Bucket '{bucket_name}' exists.")
    except ClientError as e:
        error_code = int(e.response['Error']['Code'])
        if error_code == 404:
            print(f"❌ Bucket '{bucket_name}' does not exist.")
        else:
            print(f"⚠️ Error checking bucket: {e}")

if __name__ == "__main__":
    main()

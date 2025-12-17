import os
import math
from common import get_s3_client, get_bucket_name

def main():
    s3 = get_s3_client()
    bucket_name = get_bucket_name()
    file_name = "multipart-file.bin"
    total_size = 15 * 1024 * 1024 # 15MB
    part_size = 5 * 1024 * 1024 # 5MB
    
    print(f"Starting multipart upload for '{file_name}' ({total_size} bytes)...")
    try:
        # 1. Initialize Multipart Upload
        mpu = s3.create_multipart_upload(Bucket=bucket_name, Key=file_name)
        upload_id = mpu['UploadId']
        print(f"Upload ID: {upload_id}")
        
        parts = []
        num_parts = math.ceil(total_size / part_size)
        
        # 2. Upload Parts
        for i in range(num_parts):
            part_number = i + 1
            print(f"Uploading part {part_number}/{num_parts}...")
            
            # Generate random data for part
            current_part_size = min(part_size, total_size - (i * part_size))
            data = os.urandom(current_part_size)
            
            part = s3.upload_part(
                Bucket=bucket_name, 
                Key=file_name, 
                PartNumber=part_number, 
                UploadId=upload_id, 
                Body=data
            )
            
            parts.append({
                'PartNumber': part_number,
                'ETag': part['ETag']
            })
        
        # 3. Complete Multipart Upload
        print("Completing multipart upload...")
        s3.complete_multipart_upload(
            Bucket=bucket_name, 
            Key=file_name, 
            UploadId=upload_id, 
            MultipartUpload={'Parts': parts}
        )
        print(f"✅ Multipart upload completed successfully.")
        
    except Exception as e:
        print(f"❌ Error in multipart upload: {e}")
        # Abort if possible
        if 'upload_id' in locals():
            try:
                s3.abort_multipart_upload(Bucket=bucket_name, Key=file_name, UploadId=upload_id)
                print("Aborted multipart upload.")
            except:
                pass

if __name__ == "__main__":
    main()

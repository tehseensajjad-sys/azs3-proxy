from common import get_s3_client

def main():
    s3 = get_s3_client()
    try:
        print("Listing buckets...")
        response = s3.list_buckets()
        buckets = response.get('Buckets', [])
        if not buckets:
            print("No buckets found.")
        for bucket in buckets:
            print(f"- {bucket['Name']}")
    except Exception as e:
        print(f"Error listing buckets: {e}")

if __name__ == "__main__":
    main()

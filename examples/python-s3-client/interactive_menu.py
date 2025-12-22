import sys
import subprocess
import os

SCRIPTS = {
    "1": ("List Buckets", "list_buckets.py"),
    "2": ("Check Bucket Exists", "check_bucket_exists.py"),
    "3": ("Create Bucket", "create_bucket.py"),
    "4": ("Delete Bucket", "delete_bucket.py"),
    "5": ("Upload File", "upload_file.py"),
    "6": ("Read File", "read_file.py"),
    "7": ("Delete File", "delete_file.py"),
    "8": ("Rename File", "rename_file.py"),
    "9": ("Multipart Upload", "multipart_upload.py"),
    "10": ("Create Directory", "create_directory.py"),
    "11": ("Delete Directory", "delete_directory.py"),
    "12": ("Rename Directory", "rename_directory.py"),
    "13": ("List Objects", "list_objects.py"),
    "14": ("Get File Properties", "get_file_properties.py"),
    "15": ("Get Folder Properties", "get_folder_properties.py"),
    "16": ("Get Bucket Properties", "get_bucket_properties.py"),
    "17": ("Run Comprehensive Test", "client.py"),
}

def print_menu():
    print("\n=== S3 Proxy Interactive Test Menu ===")
    # Sort keys numerically for display
    sorted_keys = sorted(SCRIPTS.keys(), key=lambda x: int(x))
    for key in sorted_keys:
        desc, _ = SCRIPTS[key]
        print(f"{key}. {desc}")
    print("0. Exit")
    print("======================================")

def main():
    # Ensure we are in the correct directory or can find the scripts
    script_dir = os.path.dirname(os.path.abspath(__file__))
    
    while True:
        print_menu()
        choice = input("\nEnter your choice: ").strip()
        
        if choice == "0":
            print("Exiting...")
            break
            
        if choice in SCRIPTS:
            desc, script_name = SCRIPTS[choice]
            script_path = os.path.join(script_dir, script_name)
            
            print(f"\n--- Running: {desc} ---")
            try:
                # Run the script using the same python interpreter
                subprocess.run([sys.executable, script_path], check=False)
            except Exception as e:
                print(f"Error running script: {e}")
            
            input("\nPress Enter to continue...")
        else:
            print("Invalid choice. Please try again.")

if __name__ == "__main__":
    main()

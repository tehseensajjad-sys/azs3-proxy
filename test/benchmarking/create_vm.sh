#!/bin/bash
set -e

# Argument parsing
EXISTING_VM_NAME="$1"

# ---------------------------
# 1. Check/Install Azure CLI
# ---------------------------
if ! command -v az &> /dev/null; then
    echo "Azure CLI 'az' command not found. Installing..."
    curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
    echo "Azure CLI installed."
    az login
else
    echo "Azure CLI is already installed."
fi

# ---------------------------
# 2. Configuration
# ---------------------------
RG_NAME="VikasFuseGrp"
LOCATION="southindia"
VM_SIZE="Standard_D16s_v5" # 16 vCPUs, 64 GiB RAM
ADMIN_USER="azureuser"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLOUD_INIT="$SCRIPT_DIR/cloud-init.yaml"
LOCAL_ENV_FILE="$PROJECT_ROOT/.env"

if [ -n "$EXISTING_VM_NAME" ]; then
    VM_NAME="$EXISTING_VM_NAME"
    echo "Using existing VM: $VM_NAME"
    SKIP_CREATION=true
else
    VM_NAME="azs3-bench-vm-$(date +%s)"
    echo "Will create new VM: $VM_NAME"
    SKIP_CREATION=false
fi

echo "Using Configuration:"
echo "  Region: $LOCATION"
echo "  VM Size: $VM_SIZE"
echo "  Cloud Init: $CLOUD_INIT"
echo "  Local .env: $LOCAL_ENV_FILE"

if [ ! -f "$CLOUD_INIT" ]; then
    echo "Error: cloud-init.yaml not found at $CLOUD_INIT"
    exit 1
fi

if [ ! -f "$LOCAL_ENV_FILE" ]; then
    echo "Error: .env file not found at $LOCAL_ENV_FILE. Please create it first."
    exit 1
fi

# ---------------------------
# 3. Provision VM
# ---------------------------
if [ "$SKIP_CREATION" = false ]; then
    echo "Creating Resource Group: $RG_NAME in $LOCATION..."
    az group create --name $RG_NAME --location $LOCATION --output none

    echo "Creating VM: $VM_NAME..."
    az vm create \
    --resource-group $RG_NAME \
    --name $VM_NAME \
    --image Ubuntu2204 \
    --size $VM_SIZE \
    --admin-username $ADMIN_USER \
    --generate-ssh-keys \
    --custom-data "@$CLOUD_INIT" \
    --public-ip-sku Standard \
    --output json > vm_create_output.json

    echo "VM Provisioned. Configuring Network Security..."
    az vm open-port --resource-group $RG_NAME --name $VM_NAME --port 8080 --priority 1010 --output none
    # Open port 22 with high priority (900) to override any JIT/Default Deny rules at 1000
    az vm open-port --resource-group $RG_NAME --name $VM_NAME --port 22 --priority 900 --output none
else
    echo "Skipping VM creation steps for existing VM: $VM_NAME"
fi

# Extract IP
echo "Retrieving IP address for $VM_NAME..."
IP_ADDRESS=$(az vm show --show-details --resource-group $RG_NAME --name $VM_NAME --query publicIps -o tsv)

if [ -z "$IP_ADDRESS" ]; then
    echo "Error: Could not retrieve IP address for VM '$VM_NAME' in resource group '$RG_NAME'."
    exit 1
fi

echo ""
echo "========================================================"
echo "VM Ready!"
echo "IP Address: $IP_ADDRESS"
echo "========================================================"

# ---------------------------
# 4. Wait for SSH availability
# ---------------------------
echo "Waiting for SSH to become available..."
MAX_RETRIES=60
COUNT=0
while ! ssh -o LogLevel=QUIET -o ConnectTimeout=5 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "$ADMIN_USER@$IP_ADDRESS" "echo 'SSH Ready'" &>/dev/null; do
    echo "Waiting for SSH... ($COUNT/$MAX_RETRIES)"
    sleep 5
    COUNT=$((COUNT+1))
    if [ $COUNT -ge $MAX_RETRIES ]; then
        echo "Timed out waiting for SSH."
        exit 1
    fi
done
echo "SSH is active."

# ---------------------------
# 5. Fix/Verify Dependencies (Go, Warp) - Robustness for existing VMs
# ---------------------------
echo "Verifying dependencies on VM..."
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "$ADMIN_USER@$IP_ADDRESS" "
    set -e
    # Check Go
    if ! command -v go &> /dev/null; then
        echo 'Go not found. Installing...'
        wget -q https://go.dev/dl/go1.22.4.linux-amd64.tar.gz
        sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.22.4.linux-amd64.tar.gz
        echo 'export PATH=\$PATH:/usr/local/go/bin' >> ~/.bashrc
        echo 'export PATH=\$PATH:~/go/bin' >> ~/.bashrc
        export PATH=\$PATH:/usr/local/go/bin
    fi
    
    # Check Warp
    if ! command -v warp &> /dev/null; then
        echo 'Warp not found. Installing...'
        export PATH=\$PATH:/usr/local/go/bin
        go install github.com/minio/warp@v0.7.6
        # Move to /usr/local/bin requires sudo, or just add go/bin to path (which we did)
        /bin/sudo /bin/cp ~/go/bin/warp /usr/local/bin/warp || echo 'Could not copy to /usr/local/bin, ensuring GOPATH/bin is used'
    fi
"

# ---------------------------
# 6. Sync Project Code
# ---------------------------
echo "Packaging local project..."
TAR_PATH="/tmp/azs3-proxy-source.tar.gz"
# Exclude git, bin, benchmarks results
tar --exclude='.git' --exclude='bin' --exclude='warp_runs' -czf "$TAR_PATH" -C "$PROJECT_ROOT" .

echo "Copying project tarball to VM..."
scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "$TAR_PATH" "$ADMIN_USER@$IP_ADDRESS:/tmp/source.tar.gz"

echo "Unpacking project on VM..."
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "$ADMIN_USER@$IP_ADDRESS" "
    mkdir -p ~/azs3-proxy
    tar -xzf /tmp/source.tar.gz -C ~/azs3-proxy
    # Ensure dependencies are tidy? No, we trust local state or run go mod tidy
"

# ---------------------------
# 7. Run Benchmarks
# ---------------------------
echo "Running Benchmarks on VM..."
ssh -t -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "$ADMIN_USER@$IP_ADDRESS" "
    export PATH=\$PATH:/usr/local/go/bin:~/go/bin:/usr/local/bin
    cd ~/azs3-proxy
    # Ensure .env exists (it was in the tarball)
    if [ ! -f .env ]; then echo 'Warning: .env missing!'; ls -la; exit 1; fi
    
    source .env
    bash test/benchmarking/warp-test.sh
"

echo "========================================================"
echo "Benchmarks Completed."
echo "You can view results or SSH into the VM:"
echo "  ssh $ADMIN_USER@$IP_ADDRESS"
echo "========================================================"

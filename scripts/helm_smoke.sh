#!/usr/bin/env bash
set -euo pipefail

# Self-contained smoke deploy of the Helm chart. Ensures helm, optional Azure login and AKS kubeconfig,
# generates values, lints, templates, and installs/updates the release.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_DIR="$ROOT_DIR/deploy/chart/azs3-proxy"
VALUES_GEN="$CHART_DIR/values.generated.yaml"

# Load .env early so defaults below can pick them up.
if [[ -f "$ROOT_DIR/.env" ]]; then
  # shellcheck disable=SC1090
  source "$ROOT_DIR/.env"
fi

RELEASE_NAME="${RELEASE_NAME:-azs3-proxy}"
NAMESPACE="${NAMESPACE:-default}"
IMAGE_REPO="${IMAGE_REPO:-ghcr.io/vibhansa-msft/azs3-proxy}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
BACKEND_TYPE="${AZURE_BACKEND_TYPE:-blob}"

# Azure / AKS settings (empty by default; filled via env or .env)
AZ_RESOURCE_GROUP="${AZ_RESOURCE_GROUP:-}"
AZ_AKS_CLUSTER="${AZ_AKS_CLUSTER:-}"
AZ_SUBSCRIPTION_ID="${AZ_SUBSCRIPTION_ID:-}"
AZ_TENANT_ID="${AZ_TENANT_ID:-}"
AZ_CLIENT_ID="${AZ_CLIENT_ID:-}"
AZ_CLIENT_SECRET="${AZ_CLIENT_SECRET:-}"

need_bin() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required tool: $1" >&2
    return 1
  fi
}

# Install helm if missing
if ! command -v helm >/dev/null 2>&1; then
  echo "helm not found; installing Helm 3..." >&2
  curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi

# Install Azure CLI if missing (deb/rpm-based detection)
install_az_cli() {
  if command -v az >/dev/null 2>&1; then
    return
  fi
  if command -v apt-get >/dev/null 2>&1; then
    echo "Installing Azure CLI via apt..." >&2
    curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
  elif command -v dnf >/dev/null 2>&1; then
    echo "Installing Azure CLI via dnf..." >&2
    sudo rpm --import https://packages.microsoft.com/keys/microsoft.asc
    sudo sh -c 'echo -e "[azure-cli]\nname=Azure CLI\nbaseurl=https://packages.microsoft.com/yumrepos/azure-cli\nenabled=1\ngpgcheck=1\ngpgkey=https://packages.microsoft.com/keys/microsoft.asc" > /etc/yum.repos.d/azure-cli.repo'
    sudo dnf install -y azure-cli
  else
    echo "Azure CLI (az) is required but no supported package manager (apt-get/dnf) detected. Install az manually: https://learn.microsoft.com/cli/azure/install-azure-cli" >&2
    exit 127
  fi
}

install_az_cli

# Install kubectl if missing (deb/rpm-based detection)
install_kubectl() {
  if command -v kubectl >/dev/null 2>&1; then
    return
  fi
  if command -v apt-get >/dev/null 2>&1; then
    echo "Installing kubectl via apt..." >&2
    sudo apt-get update -y
    sudo apt-get install -y kubectl
  elif command -v dnf >/dev/null 2>&1; then
    echo "Installing kubectl via dnf..." >&2
    sudo dnf install -y kubectl
  else
    echo "kubectl is required but no supported package manager (apt-get/dnf) detected. Install manually: https://kubernetes.io/docs/tasks/tools/" >&2
    exit 127
  fi
}

install_kubectl

# Ensure kubectl is available
need_bin kubectl || { echo "please install kubectl"; exit 127; }

# Install Python3 + pip if missing
install_python() {
  if command -v python3 >/dev/null 2>&1 && command -v pip3 >/dev/null 2>&1; then
    return
  fi
  if command -v apt-get >/dev/null 2>&1; then
    echo "Installing python3 and pip via apt..." >&2
    sudo apt-get update -y
    sudo apt-get install -y python3 python3-pip
  elif command -v dnf >/dev/null 2>&1; then
    echo "Installing python3 and pip via dnf..." >&2
    sudo dnf install -y python3 python3-pip
  else
    echo "python3/pip are required but no supported package manager (apt-get/dnf) detected." >&2
    exit 127
  fi
}

install_python

az_login() {
  need_bin az || { echo "Azure CLI (az) not found; install it or set KUBECONFIG manually"; exit 127; }
  if az account show >/dev/null 2>&1; then
    return 0
  fi
  if [[ -n "$AZ_CLIENT_ID" && -n "$AZ_CLIENT_SECRET" && -n "$AZ_TENANT_ID" ]]; then
    echo "Logging into Azure using service principal..."
    az login --service-principal \
      --username "$AZ_CLIENT_ID" \
      --password "$AZ_CLIENT_SECRET" \
      --tenant "$AZ_TENANT_ID" >/dev/null
  else
    echo "No Azure session detected. Running 'az login' interactively..." >&2
    az login >/dev/null
  fi
}

# Ensure Azure login only when needed
if [[ -n "$AZ_SUBSCRIPTION_ID" || ( -n "$AZ_RESOURCE_GROUP" && -n "$AZ_AKS_CLUSTER" ) ]]; then
  az_login
fi

if [[ -n "$AZ_SUBSCRIPTION_ID" ]]; then
  az account set --subscription "$AZ_SUBSCRIPTION_ID"
fi

# Optionally fetch AKS credentials if variables are set
if [[ -n "$AZ_RESOURCE_GROUP" && -n "$AZ_AKS_CLUSTER" ]]; then
  echo "Fetching AKS kubeconfig for ${AZ_AKS_CLUSTER} in ${AZ_RESOURCE_GROUP}..."
  az aks get-credentials --resource-group "$AZ_RESOURCE_GROUP" --name "$AZ_AKS_CLUSTER" --overwrite-existing
fi

# Verify cluster reachability
if ! kubectl version --output=yaml >/dev/null 2>&1; then
  echo "Kubernetes cluster unreachable. Set KUBECONFIG or provide AZ_RESOURCE_GROUP and AZ_AKS_CLUSTER." >&2
  exit 1
fi

cat >"$VALUES_GEN" <<EOF
replicaCount: 1
image:
  repository: ${IMAGE_REPO}
  tag: ${IMAGE_TAG}
  pullPolicy: IfNotPresent
service:
  type: ClusterIP
  port: 8080
env:
  AZURE_BACKEND_TYPE: "${BACKEND_TYPE}"
  LISTEN_ADDR: ":8080"
secretEnv:
  AZURE_STORAGE_ACCOUNT: "${AZURE_STORAGE_ACCOUNT:-}"
  AZURE_STORAGE_SAS_TOKEN: "${AZURE_STORAGE_SAS_TOKEN:-}"
  AZURE_STORAGE_KEY: "${AZURE_STORAGE_KEY:-}"
  S3_ACCESS_KEY: "${S3_ACCESS_KEY:-}"
  S3_SECRET_KEY: "${S3_SECRET_KEY:-}"
resources: {}
ingress:
  enabled: false
EOF

echo "Linting chart..."
helm lint "$CHART_DIR" --values "$VALUES_GEN"

echo "Rendering templates..."
helm template "$RELEASE_NAME" "$CHART_DIR" --namespace "$NAMESPACE" --values "$VALUES_GEN" >/dev/null

echo "Installing/upgrading release ${RELEASE_NAME} in namespace ${NAMESPACE}..."
helm upgrade --install "$RELEASE_NAME" "$CHART_DIR" \
  --namespace "$NAMESPACE" \
  --create-namespace \
  --values "$VALUES_GEN"

helm status "$RELEASE_NAME" --namespace "$NAMESPACE"

# Smoke test: port-forward and list buckets via example client
EXAMPLES_DIR="$ROOT_DIR/examples/python-s3-client"
REQUIREMENTS_FILE="$EXAMPLES_DIR/requirements.txt"

if [[ -f "$REQUIREMENTS_FILE" ]]; then
  echo "Installing Python client requirements..."
  pip3 install -r "$REQUIREMENTS_FILE"
fi

echo "Starting port-forward to service ${RELEASE_NAME}-azs3-proxy..."
kubectl port-forward svc/${RELEASE_NAME}-azs3-proxy 8080:8080 --namespace "$NAMESPACE" >/tmp/azs3-proxy-portforward.log 2>&1 &
PF_PID=$!
cleanup_pf() {
  if kill -0 "$PF_PID" >/dev/null 2>&1; then
    kill "$PF_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup_pf EXIT
sleep 5

echo "Running list_buckets smoke test..."
PROXY_URL="http://127.0.0.1:8080" \
  python3 "$EXAMPLES_DIR/list_buckets.py"

cleanup_pf
trap - EXIT
echo "Done."

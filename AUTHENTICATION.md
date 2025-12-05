# Azure Storage Authentication Configuration

This guide explains how to configure authentication for the s3-azure-proxy. The proxy supports 6 different Azure authentication methods, and automatically detects which one to use based on your environment variables.

## Quick Reference

| Authentication Mode | Best For | Required Environment Variables |
|---|---|---|
| **Account Key** | Development & Testing | `AZURE_STORAGE_ACCOUNT`, `AZURE_STORAGE_KEY` |
| **SAS Token** | Time-limited & Delegated Access | `AZURE_STORAGE_ACCOUNT`, `AZURE_STORAGE_SAS_TOKEN` |
| **Managed Identity (MSI)** | Azure VMs, App Service, AKS | `AZURE_STORAGE_ACCOUNT`, `AZURE_USE_MSI=true` |
| **Service Principal** | CI/CD, On-Premises, Automation | `AZURE_STORAGE_ACCOUNT`, `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET` |
| **Federated Token (OIDC)** | GitHub Actions, GitLab CI | `AZURE_STORAGE_ACCOUNT`, `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_FEDERATED_TOKEN_FILE` |
| **Azure CLI** | Local Development | `AZURE_STORAGE_ACCOUNT`, `AZURE_USE_CLI_AUTH=true` |

## Configuration Details

### 1. Account Key Authentication

**When to use:** Development, local testing, and simple non-production setups.

**Environment Variables:**
```bash
AZURE_STORAGE_ACCOUNT=<storage-account-name>
AZURE_STORAGE_KEY=<storage-account-key>
S3_ACCESS_KEY=<s3-access-key>           # For S3 signature verification
S3_SECRET_KEY=<s3-secret-key>           # For S3 signature verification
```

**Example:**
```bash
export AZURE_STORAGE_ACCOUNT=mystorageacct
export AZURE_STORAGE_KEY=DefaultEndpointsProtocol=https;AccountName=...
export S3_ACCESS_KEY=AKIA1234567890ABCDEF
export S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY
./s3-proxy
```

**⚠️ Security Warning:** Account keys are permanent credentials. Only use this in development environments. For production, use Managed Identity or Service Principal instead.

---

### 2. SAS Token Authentication

**When to use:** Granting time-limited or restricted access to specific containers/blobs.

**Environment Variables:**
```bash
AZURE_STORAGE_ACCOUNT=<storage-account-name>
AZURE_STORAGE_SAS_TOKEN=<sas-token-string>
S3_ACCESS_KEY=<s3-access-key>
S3_SECRET_KEY=<s3-secret-key>
```

**Example:**
```bash
export AZURE_STORAGE_ACCOUNT=mystorageacct
export AZURE_STORAGE_SAS_TOKEN=sv=2021-06-08&st=2024-01-01&se=2025-01-01&sr=c&sp=racwd
export S3_ACCESS_KEY=AKIA1234567890ABCDEF
export S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY
./s3-proxy
```

**To generate a SAS token:**
```bash
# Using Azure CLI
az storage container generate-sas \
  --account-name mystorageacct \
  --name mycontainer \
  --permissions racwd \
  --expiry 2025-12-31
```

---

### 3. Managed Identity (MSI) Authentication

**When to use:** Running in Azure (VM, Container Instance, App Service, Functions, AKS) - **this is the recommended approach for Azure deployments**.

**For System-Assigned Managed Identity:**
```bash
AZURE_STORAGE_ACCOUNT=<storage-account-name>
AZURE_USE_MSI=true
S3_ACCESS_KEY=<s3-access-key>
S3_SECRET_KEY=<s3-secret-key>
```

**For User-Assigned Managed Identity:**
```bash
AZURE_STORAGE_ACCOUNT=<storage-account-name>
AZURE_USE_MSI=true
AZURE_CLIENT_ID=<user-assigned-identity-client-id>
S3_ACCESS_KEY=<s3-access-key>
S3_SECRET_KEY=<s3-secret-key>
```

**Example (System-Assigned on VM):**
```bash
export AZURE_STORAGE_ACCOUNT=mystorageacct
export AZURE_USE_MSI=true
export S3_ACCESS_KEY=AKIA1234567890ABCDEF
export S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY
./s3-proxy
```

**Setup Steps:**
1. Enable Managed Identity on your Azure resource (VM, App Service, AKS, etc.)
2. Grant the managed identity RBAC access to your storage account (e.g., "Storage Blob Data Contributor" role)
3. Set the environment variables above
4. No secrets needed - credentials are automatically obtained from Azure's metadata service

**✅ Recommended for Azure resources** - Most secure, no secrets, automatic token rotation.

---

### 4. Service Principal Authentication

**When to use:** CI/CD pipelines, on-premises servers, automation, or any non-Azure environment requiring authentication.

**Environment Variables:**
```bash
AZURE_STORAGE_ACCOUNT=<storage-account-name>
AZURE_TENANT_ID=<azure-tenant-id>
AZURE_CLIENT_ID=<service-principal-client-id>
AZURE_CLIENT_SECRET=<service-principal-client-secret>
S3_ACCESS_KEY=<s3-access-key>
S3_SECRET_KEY=<s3-secret-key>
```

**Example:**
```bash
export AZURE_STORAGE_ACCOUNT=mystorageacct
export AZURE_TENANT_ID=12345678-1234-1234-1234-123456789012
export AZURE_CLIENT_ID=87654321-4321-4321-4321-210987654321
export AZURE_CLIENT_SECRET=your-secret-value-here
export S3_ACCESS_KEY=AKIA1234567890ABCDEF
export S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY
./s3-proxy
```

**Create a Service Principal:**
```bash
# Create service principal
az ad sp create-for-rbac --name s3-proxy-sp --role "Storage Blob Data Contributor" \
  --scopes /subscriptions/{subscription-id}/resourceGroups/{resource-group}/providers/Microsoft.Storage/storageAccounts/{storage-account-name}
```

This will output:
```json
{
  "appId": "87654321-4321-4321-4321-210987654321",
  "displayName": "s3-proxy-sp",
  "password": "your-secret-value-here",
  "tenant": "12345678-1234-1234-1234-123456789012"
}
```

**Using with HashiCorp Vault (Recommended):**
```bash
# Store SPN credentials in Vault
vault kv put secret/s3-proxy \
  tenant_id=12345678-1234-1234-1234-123456789012 \
  client_id=87654321-4321-4321-4321-210987654321 \
  client_secret=your-secret-value-here

# Retrieve and use
export CREDS=$(vault kv get -format=json secret/s3-proxy)
export AZURE_TENANT_ID=$(echo $CREDS | jq -r '.data.data.tenant_id')
export AZURE_CLIENT_ID=$(echo $CREDS | jq -r '.data.data.client_id')
export AZURE_CLIENT_SECRET=$(echo $CREDS | jq -r '.data.data.client_secret')
export AZURE_STORAGE_ACCOUNT=mystorageacct
./s3-proxy
```

---

### 5. Federated Token (OIDC) Authentication

**When to use:** GitHub Actions, GitLab CI, or any OIDC-compliant CI/CD system - **no long-lived secrets needed**.

**Environment Variables:**
```bash
AZURE_STORAGE_ACCOUNT=<storage-account-name>
AZURE_TENANT_ID=<azure-tenant-id>
AZURE_CLIENT_ID=<application-client-id>
AZURE_FEDERATED_TOKEN_FILE=<path-to-token-file>
S3_ACCESS_KEY=<s3-access-key>
S3_SECRET_KEY=<s3-secret-key>
```

**GitHub Actions Example:**
```yaml
name: Deploy with S3 Azure Proxy
on: [push]

jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      id-token: write
    steps:
      - uses: actions/checkout@v3
      
      # Use Azure Login action to set up federated token
      - uses: azure/login@v1
        with:
          client-id: ${{ secrets.AZURE_CLIENT_ID }}
          tenant-id: ${{ secrets.AZURE_TENANT_ID }}
          subscription-id: ${{ secrets.AZURE_SUBSCRIPTION_ID }}
      
      - run: |
          export AZURE_STORAGE_ACCOUNT=mystorageacct
          export AZURE_TENANT_ID=${{ secrets.AZURE_TENANT_ID }}
          export AZURE_CLIENT_ID=${{ secrets.AZURE_CLIENT_ID }}
          export AZURE_FEDERATED_TOKEN_FILE=$ACTIONS_ID_TOKEN_REQUEST_TOKEN
          export S3_ACCESS_KEY=AKIA1234567890ABCDEF
          export S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY
          ./s3-proxy
```

**GitLab CI Example:**
```yaml
stages:
  - deploy

deploy:
  stage: deploy
  image: alpine:latest
  script:
    - export AZURE_STORAGE_ACCOUNT=mystorageacct
    - export AZURE_TENANT_ID=$AZURE_TENANT_ID
    - export AZURE_CLIENT_ID=$AZURE_CLIENT_ID
    - export AZURE_FEDERATED_TOKEN_FILE=$CI_JOB_JWT_V2
    - export S3_ACCESS_KEY=AKIA1234567890ABCDEF
    - export S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY
    - ./s3-proxy
```

**Setup OIDC Trust in Azure:**
```bash
# Create app registration for OIDC
az ad app create --display-name "s3-proxy-oidc"

# Get the application ID
APP_ID=$(az ad app list --display-name "s3-proxy-oidc" --query "[0].id" -o tsv)

# Add federated credential for GitHub
az ad app federated-credential create \
  --id $APP_ID \
  --parameters @federation.json
```

**federation.json:**
```json
{
  "name": "github-federation",
  "issuer": "https://token.actions.githubusercontent.com",
  "subject": "repo:yourorg/yourrepo:ref:refs/heads/main",
  "description": "GitHub Actions federation for s3-proxy",
  "audiences": ["api://AzureADTokenExchange"]
}
```

**✅ Recommended for CI/CD** - No long-lived secrets, automatic token refresh, zero secret rotation overhead.

---

### 6. Azure CLI Authentication

**When to use:** Local development only - uses credentials from `az login`.

**Environment Variables:**
```bash
AZURE_STORAGE_ACCOUNT=<storage-account-name>
AZURE_USE_CLI_AUTH=true
S3_ACCESS_KEY=<s3-access-key>
S3_SECRET_KEY=<s3-secret-key>
```

**Example:**
```bash
# First, authenticate
az login

# Then run the proxy
export AZURE_STORAGE_ACCOUNT=mystorageacct
export AZURE_USE_CLI_AUTH=true
export S3_ACCESS_KEY=AKIA1234567890ABCDEF
export S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY
./s3-proxy
```

**⚠️ Development Only** - Not suitable for production environments.

---

## Authentication Detection Order

The proxy automatically detects authentication mode based on environment variables in this priority order:

1. **Account Key** - `AZURE_STORAGE_KEY` is set
2. **SAS Token** - `AZURE_STORAGE_SAS_TOKEN` is set
3. **Federated Token** - `AZURE_FEDERATED_TOKEN_FILE` is set
4. **Service Principal** - `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, and `AZURE_TENANT_ID` are set
5. **Managed Identity** - `AZURE_USE_MSI=true` or `IMDS_ENDPOINT` is detected
6. **Azure CLI** - `AZURE_USE_CLI_AUTH=true` is set

**Important:** Only set environment variables for ONE authentication method. Setting variables from multiple methods may cause unpredictable behavior.

---

## Production Deployment Recommendations

### For Azure Cloud Resources (VMs, App Services, AKS, Functions)
```
Use: Managed Identity (MSI)
```
- ✅ Most secure - no secrets stored
- ✅ Automatic token rotation
- ✅ Native Azure RBAC integration
- ✅ Zero credential rotation needed
- ✅ Azure best practice

**Kubernetes Deployment Example:**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: s3-proxy
  annotations:
    aadpodidbinding: s3-proxy-identity  # Azure AD Pod Identity binding
spec:
  containers:
  - name: s3-proxy
    image: s3-azure-proxy:latest
    env:
    - name: AZURE_STORAGE_ACCOUNT
      value: mystorageacct
    - name: AZURE_USE_MSI
      value: "true"
    - name: S3_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: s3-credentials
          key: access-key
    - name: S3_SECRET_KEY
      valueFrom:
        secretKeyRef:
          name: s3-credentials
          key: secret-key
```

### For CI/CD Pipelines (GitHub Actions, GitLab CI, etc.)
```
Use: Federated Token (OIDC)
```
- ✅ No long-lived secrets
- ✅ Automatic token refresh
- ✅ Works with all major CI/CD platforms
- ✅ Zero credential rotation needed
- ✅ Industry-standard approach

### For On-Premises or External Services
```
Use: Service Principal with Secure Credential Storage
```
- ✅ Works anywhere
- ✅ RBAC support
- ✅ Audit trail capability
- ⚠️ Requires secure credential storage (Vault, AWS Secrets Manager, etc.)

**Never use in production:**
- ❌ Account Key (permanent credentials, manual rotation)
- ❌ SAS Token (limited scope, expiration management overhead)
- ❌ Azure CLI (local only, interactive)

---

## Troubleshooting

### "No Azure authentication method configured"
**Solution:** Ensure you've set the required environment variables for your chosen authentication method. Check the "Configuration Details" section above for your specific method.

### "AZURE_STORAGE_ACCOUNT is required"
**Solution:** Set the `AZURE_STORAGE_ACCOUNT` environment variable to your storage account name.

### "Failed to create blob client: unauthorized"
**Common causes:**
- Credentials are incorrect or expired
- Identity doesn't have proper RBAC role (need "Storage Blob Data Contributor" or similar)
- SAS token has expired or doesn't include required permissions
- Service Principal secret is incorrect

**Solution:** Verify credentials and RBAC assignments in Azure Portal.

### "Failed to get token credential" (for token-based auth)
**For Managed Identity:**
- Verify the VM/App Service has Managed Identity enabled
- Check RBAC role assignment on the storage account
- Ensure IMDS endpoint is accessible

**For Service Principal:**
- Verify `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, and `AZURE_CLIENT_SECRET` are all set
- Confirm the service principal hasn't been deleted

**For Federated Token:**
- Verify the token file path exists and is readable
- Confirm OIDC federation is configured in Azure AD

### "Cannot connect to Azure"
**Solutions:**
- Test network connectivity to Azure endpoints
- For MSI, verify IMDS endpoint availability
- Check for firewall or proxy restrictions
- Verify subscription and resource access

---

## S3 Signature Configuration

In addition to Azure authentication, you must also provide S3 signature credentials for request verification:

```bash
S3_ACCESS_KEY=<s3-compatible-access-key>
S3_SECRET_KEY=<s3-compatible-secret-key>
```

These are used by the proxy to verify incoming S3 API requests. They are separate from Azure authentication credentials. You can set these to any values you want - clients will use these same values to sign their S3 requests.

---

## Additional Resources

- [Azure Storage Account documentation](https://docs.microsoft.com/en-us/azure/storage/common/storage-account-overview)
- [Azure Managed Identities documentation](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/)
- [Azure Service Principal documentation](https://docs.microsoft.com/en-us/azure/active-directory/develop/app-objects-and-service-principals)
- [Federated Identity Credentials documentation](https://docs.microsoft.com/en-us/azure/active-directory/develop/workload-identity-federation)

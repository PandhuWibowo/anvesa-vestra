# Managing Connections

A **connection** is a named reference to a bucket along with the credentials needed to access it. Connections are stored locally in a SQLite database (`server/data.db`) and never sent to any external service.

---

## Connection Fields

| Field | Required | Description |
|---|---|---|
| Name | Yes | A human-readable label shown in the sidebar |
| Provider | Yes | `gcp`, `aws`, `huawei`, `alibaba`, or `azure` — set on creation, cannot be changed |
| Bucket | Yes | The bucket (or container) name without any protocol prefix |
| Credentials | Depends | JSON credentials object — format varies per provider |

---

## Google Cloud Storage (GCS)

### Obtaining a Service Account Key

1. Open [Google Cloud Console → IAM & Admin → Service Accounts](https://console.cloud.google.com/iam-admin/serviceaccounts).
2. Select or create a service account.
3. Under **Keys**, click **Add Key → Create new key → JSON**.
4. Download the `.json` file.

### Required IAM Roles

| Role | When needed |
|---|---|
| `Storage Object Viewer` | Browse and download only |
| `Storage Object Creator` | Upload files |
| `Storage Object Admin` | Upload, delete, rename, metadata edit |
| `Storage Admin` | Full bucket management |

Assign the minimum role needed for your use case.

### Credential Format

Paste the full contents of the downloaded JSON key file into the Credentials field:

```json
{
  "type": "service_account",
  "project_id": "my-project",
  "private_key_id": "key-id",
  "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----\n",
  "client_email": "my-sa@my-project.iam.gserviceaccount.com",
  "client_id": "123456789",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token"
}
```

### Public Buckets

If your bucket is publicly readable, leave the Credentials field empty. Anveesa Vestra will access it without authentication. Upload, delete, and metadata-edit operations still require credentials.

---

## Google Drive

> **Requires Google Workspace.** Google Drive connections rely on the **Shared Drives** (Team Drives) feature, which is only available on Google Workspace accounts (Business Starter and above). Personal "My Drive" folders are **not supported** — service accounts have no storage quota, so uploads will fail with a `storageQuotaExceeded` error.

### Prerequisites

| Requirement | Details |
|---|---|
| Google Workspace account | Business Starter / Standard / Plus / Enterprise |
| Google Drive API enabled | Enable in Google Cloud Console → APIs & Services → Library |
| Shared Drive created | Create one under **Shared drives** in Google Drive |
| Service account added to Shared Drive | Role: **Contributor** or **Content manager** |

### Step 1 — Enable Google Drive API

1. Open [Google Cloud Console → APIs & Services → Library](https://console.cloud.google.com/apis/library).
2. Search for **Google Drive API** and click **Enable**.

### Step 2 — Create a Service Account

1. Go to **IAM & Admin → Service Accounts**.
2. Click **Create Service Account**, fill in a name, and finish.
3. Under **Keys**, click **Add Key → Create new key → JSON**.
4. Download the `.json` file — this is your credential.

### Step 3 — Create a Shared Drive and add the Service Account

1. Open Google Drive → click **Shared drives** in the left sidebar → **New**.
2. Give it a name and create it.
3. Right-click the Shared Drive → **Manage members**.
4. Add the service account email (found in the JSON as `client_email`) with role **Contributor** or **Content manager**.

### Step 4 — Get the Folder ID

The **Folder ID** is the part of the URL after `/folders/` when you open the Shared Drive or any subfolder:

```
https://drive.google.com/drive/folders/1AbCdEfGhIjKlMnOpQrStUvWx
                                        └─────── Folder ID ──────┘
```

You can use the Shared Drive root ID, or any subfolder ID within it.

### Credential Format

Paste the full contents of the downloaded service account JSON file:

```json
{
  "type": "service_account",
  "project_id": "my-project",
  "private_key_id": "key-id",
  "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----\n",
  "client_email": "my-sa@my-project.iam.gserviceaccount.com",
  "client_id": "123456789",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token"
}
```

### Connection Fields

| Field | Value |
|---|---|
| **Name** | Any label, e.g. `Team Drive` |
| **Folder ID** | Shared Drive or subfolder ID from the URL |
| **Credentials** | Full contents of the service account JSON file |

---

## Amazon S3

### Credential Format

```json
{
  "access_key_id": "AKIAIOSFODNN7EXAMPLE",
  "secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
  "region": "us-east-1"
}
```

| Field | Required | Default | Description |
|---|---|---|---|
| `access_key_id` | Yes | — | AWS IAM access key |
| `secret_access_key` | Yes | — | AWS IAM secret |
| `region` | No | `us-east-1` | AWS region where the bucket lives |
| `session_token` | No | — | Temporary session token (STS/assumed role) |
| `endpoint` | No | — | Custom endpoint for S3-compatible services |

### Required IAM Permissions

```json
{
  "Effect": "Allow",
  "Action": [
    "s3:ListBucket",
    "s3:GetObject",
    "s3:PutObject",
    "s3:DeleteObject",
    "s3:CopyObject",
    "s3:HeadObject",
    "s3:HeadBucket"
  ],
  "Resource": [
    "arn:aws:s3:::your-bucket-name",
    "arn:aws:s3:::your-bucket-name/*"
  ]
}
```

---

## Cloudflare R2

R2 uses the S3-compatible API. Set a custom endpoint to point to your R2 account.

```json
{
  "access_key_id": "your-r2-access-key-id",
  "secret_access_key": "your-r2-secret-access-key",
  "endpoint": "https://<ACCOUNT_ID>.r2.cloudflarestorage.com",
  "region": "auto"
}
```

Replace `<ACCOUNT_ID>` with your Cloudflare account ID, found in the R2 dashboard.

---

## MinIO

MinIO also uses the S3-compatible API. Use your MinIO server's address as the endpoint.

```json
{
  "access_key_id": "minioadmin",
  "secret_access_key": "minioadmin",
  "endpoint": "http://localhost:9000",
  "region": "us-east-1"
}
```

> For self-signed TLS, ensure the MinIO server certificate is trusted on the machine running Anveesa Vestra's backend.

---

## Huawei OBS

### Obtaining Credentials

1. Log in to [Huawei Cloud Console](https://console.huaweicloud.com).
2. Go to **My Credentials → Access Keys → Create Access Key**.
3. Download the CSV containing your AK/SK pair.
4. Find your OBS endpoint in **OBS Console → Bucket → Overview → Endpoint**.

### Credential Format

```json
{
  "access_key_id": "your-access-key",
  "secret_access_key": "your-secret-key",
  "endpoint": "https://obs.cn-north-4.myhuaweicloud.com",
  "region": "cn-north-4"
}
```

| Field | Required | Description |
|---|---|---|
| `access_key_id` | Yes | Huawei Cloud AK |
| `secret_access_key` | Yes | Huawei Cloud SK |
| `endpoint` | Yes | OBS service endpoint for your region |
| `region` | No | Region identifier (e.g. `cn-north-4`) |

---

## Alibaba Cloud OSS

### Obtaining Credentials

1. Log in to [Alibaba Cloud Console](https://console.aliyun.com).
2. Go to **AccessKey Management → Create AccessKey**.
3. Find your OSS endpoint in **OSS Console → Bucket → Overview → Endpoint**.

### Credential Format

```json
{
  "access_key_id": "your-access-key-id",
  "secret_access_key": "your-access-key-secret",
  "endpoint": "https://oss-cn-hangzhou.aliyuncs.com",
  "region": "cn-hangzhou"
}
```

| Field | Required | Description |
|---|---|---|
| `access_key_id` | Yes | Alibaba Cloud AccessKey ID |
| `secret_access_key` | Yes | Alibaba Cloud AccessKey Secret |
| `endpoint` | Yes | OSS endpoint for your region |
| `region` | No | Region identifier (e.g. `cn-hangzhou`) |

---

## Azure Blob Storage

### Obtaining Credentials

1. Open [Azure Portal](https://portal.azure.com) → **Storage accounts**.
2. Select your storage account.
3. Go to **Security + networking → Access keys**.
4. Copy **Storage account name** and either **key1** or **key2**.

> The **container** name (used in the Bucket field) is found under **Data storage → Containers** in your storage account.

### Credential Format

```json
{
  "account_name": "mystorageaccount",
  "account_key": "base64encodedkey=="
}
```

| Field | Required | Description |
|---|---|---|
| `account_name` | Yes | Azure Storage account name |
| `account_key` | Yes | Base64-encoded storage account key |

---

## Testing a Connection

Before saving, click **Test Connection**. The backend verifies access by performing a lightweight bucket/container probe:

| Provider | Test operation |
|---|---|
| GCS | List first object in the bucket |
| AWS / R2 / MinIO | `HeadBucket` |
| Huawei OBS | List bucket metadata |
| Alibaba OSS | `GetBucketInfo` |
| Azure | List containers / check container existence |

A green notice confirms success. A red notice shows the error returned by the cloud provider.

---

## Editing a Connection

Click the **pencil icon** next to any connection in the sidebar to open the edit form. The provider card is locked — you cannot switch providers on an existing connection. All other fields (name, bucket, credentials) can be updated.

---

## Deleting a Connection

Click the **× icon** next to a connection in the sidebar. A confirmation dialog will appear. Deleting a connection removes it from the local database only — it does not delete any data from your cloud bucket.

---

## Data Storage

Connections are stored in `server/data.db` (SQLite). This file is created automatically on first run and is excluded from version control via `.gitignore`.

**Back up this file** if you want to preserve your connections across reinstalls.

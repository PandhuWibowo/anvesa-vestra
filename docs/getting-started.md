# Getting Started

This guide walks you through running Anveesa Vestra locally for the first time.

---

## Option A — Docker (Quickest)

No toolchain needed. Pull the pre-built image and run it:

```bash
docker run -d \
  --name anveesa-vestra \
  -p 80:80 \
  -e JWT_SECRET=your-secret-here \
  -e ENCRYPTION_KEY=your-encryption-key \
  -v anveesa-data:/data \
  pandhuwibowo/anveesa-vestra:latest
```

Open [http://localhost](http://localhost) in your browser. On first launch you will be prompted to **create an admin account** — see [Authentication](./authentication.md) for details.

The `-v anveesa-data:/data` flag persists your SQLite database across container restarts. See [Deployment](./deployment.md) for more Docker options and environment variables.

---

## Option B — From Source

### Prerequisites

| Requirement | Version | Notes |
|---|---|---|
| Go | 1.21+ | [golang.org](https://golang.org/dl/) |
| Bun | 1.0+ | [bun.sh](https://bun.sh) — or use npm/Node 18+ |
| Make | any | Available on macOS/Linux by default |

### 1. Clone the Repository

```bash
git clone https://github.com/PandhuWibowo/anveesa-vestra.git
cd anveesa-vestra
```

### 2. Install Frontend Dependencies

```bash
cd web
bun install
cd ..
```

> Using npm? Run `npm install` instead of `bun install`.

### 3. Start the Development Server

```bash
make dev
```

This command starts both the Go backend (port **8080**) and the Vite dev server (port **5173**) in parallel. It also waits for the backend to be ready before launching the frontend.

Open [http://localhost:5173](http://localhost:5173) in your browser.

> By default, authentication is enabled. On first launch you will see the **Create Admin Account** screen. Enter a username and password (min 8 characters) to create your admin account. See [Authentication](./authentication.md) for configuration options.

---

## Add Your First Connection

After logging in, you will see the welcome screen. Click **New Connection** to open the connection form. Select a provider card, fill in the fields, and click **Test Connection** before saving.

### Google Cloud Storage

1. Select the **Google Cloud Storage** card.
2. Enter a **Connection name** (e.g. `my-production-bucket`).
3. Paste your **GCS bucket name** (without `gs://`).
4. Paste your **Service account JSON** key into the Credentials field.
5. Click **Test Connection** — a green notice confirms access.
6. Click **Save**.

> For public buckets you can leave Credentials empty.

### Amazon S3 / S3-Compatible (R2, MinIO)

1. Select the **AWS S3** card.
2. Enter a **Connection name**.
3. Enter your **S3 bucket name**.
4. Paste a JSON credentials object:

```json
{
  "access_key_id": "AKIAIOSFODNN7EXAMPLE",
  "secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
  "region": "us-east-1"
}
```

5. Click **Test Connection**, then **Save**.

For Cloudflare R2 or MinIO, add an `"endpoint"` key — see [Managing Connections](./connections.md#cloudflare-r2).

### Huawei OBS

1. Select the **Huawei OBS** card.
2. Enter a **Connection name** and your **OBS bucket name**.
3. Paste a JSON credentials object:

```json
{
  "access_key_id": "your-ak",
  "secret_access_key": "your-sk",
  "endpoint": "https://obs.cn-north-4.myhuaweicloud.com",
  "region": "cn-north-4"
}
```

4. Click **Test Connection**, then **Save**.

### Alibaba Cloud OSS

1. Select the **Alibaba Cloud OSS** card.
2. Enter a **Connection name** and your **OSS bucket name**.
3. Paste a JSON credentials object:

```json
{
  "access_key_id": "your-ak",
  "secret_access_key": "your-sk",
  "endpoint": "https://oss-cn-hangzhou.aliyuncs.com",
  "region": "cn-hangzhou"
}
```

4. Click **Test Connection**, then **Save**.

### Azure Blob Storage

1. Select the **Azure Blob Storage** card.
2. Enter a **Connection name** and your **container name** (the Azure Blob container, not the storage account).
3. Paste a JSON credentials object:

```json
{
  "account_name": "mystorageaccount",
  "account_key": "base64encodedkey=="
}
```

4. Click **Test Connection**, then **Save**.

> Find the account key in the Azure Portal → Storage account → **Security + networking → Access keys**.

### Google Drive

1. Select the **Google Drive** card.
2. Enter a **Connection name** (e.g. `team-drive-docs`).
3. Enter the **Folder ID** as the bucket name. This is the long string in the Google Drive folder URL: `https://drive.google.com/drive/folders/{FOLDER_ID}`.
4. Paste your **Service account JSON** key into the Credentials field. The service account must have access to the folder (share the folder with the service account email).
5. Click **Test Connection** — a green notice confirms the folder is accessible.
6. Click **Save**.

> See [Managing Connections — Google Drive](./connections.md#google-drive) for detailed credential setup.

---

## Browse Your Bucket

Click any saved connection in the sidebar to open the file browser. From there you can:

- Navigate folders
- Upload files (drag-and-drop or click)
- Download, delete, rename, or view file metadata

See [File Browser](./browser.md) for a full feature walkthrough.

---

## Project Layout (Source)

```
anveesa-vestra/
├── server/              Go backend — API server and database
│   ├── main.go          Route definitions and server startup
│   ├── db/              SQLite schema and initialization
│   ├── handlers/        Request handlers per provider
│   │   ├── gcp.go       Google Cloud Storage
│   │   ├── aws.go       Amazon S3 / R2 / MinIO
│   │   ├── azure.go     Azure Blob Storage
│   │   ├── alibaba.go   Alibaba Cloud OSS
│   │   ├── huawei.go    Huawei OBS
│   │   ├── gdrive.go    Google Drive
│   │   ├── auth.go      Authentication (login, register)
│   │   ├── shared.go    Shared links
│   │   ├── jobs.go      Background jobs
│   │   ├── webhooks.go  Webhook management
│   │   ├── audit.go     Audit logging
│   │   ├── analytics.go Dashboard analytics
│   │   ├── search.go    Cross-provider search
│   │   └── ...          Transfer, zip, proxy, health
│   └── middleware/      CORS, Auth, Rate limiting
├── web/                 Vue 3 frontend
│   ├── src/
│   │   ├── App.vue              Root component and navigation logic
│   │   ├── components/          UI and feature components
│   │   │   ├── connections/     Browser + connection form
│   │   │   ├── views/           Dashboard, search, audit, jobs, webhooks
│   │   │   ├── auth/            Login / registration screen
│   │   │   └── ui/              Reusable UI components
│   │   └── composables/         Shared state and API logic
│   └── vite.config.js           Dev server and proxy config
├── docs/                This documentation
├── deploy/              Container runtime configuration
├── Dockerfile           Multi-stage image build (bun → go → nginx)
└── Makefile             Dev and build commands
```

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Backend server port |
| `DB_PATH` | `data.db` | SQLite database file path |
| `AUTH_ENABLED` | `true` | Enable/disable login requirement |
| `JWT_SECRET` | `change-me-in-production` | JWT signing key |
| `ENCRYPTION_KEY` | *(auto-generated)* | AES key for credential encryption |
| `CORS_ORIGIN` | `*` | Allowed CORS origin |

See [Authentication](./authentication.md) and [Deployment](./deployment.md) for full details.

---

## Keyboard Shortcuts

| Key | Action |
|---|---|
| `j` / `↓` | Move focus to next row |
| `k` / `↑` | Move focus to previous row |
| `Enter` | Open folder or preview file |
| `Space` | Toggle preview |
| `f` | Toggle fullscreen preview |
| `d` | Download focused file |
| `Delete` | Delete focused file/folder |
| `/` | Focus search box |
| `r` | Refresh current folder |
| `Backspace` | Navigate up one folder level |
| `Escape` | Exit fullscreen / close panel / clear search |
| `n` | New connection |
| `?` | Show all shortcuts |

# Management Views

Anveesa Vestra includes six management views accessible from the sidebar navigation. These provide visibility and control over platform-wide operations beyond basic file browsing.

---

## Analytics Dashboard

The dashboard gives you an at-a-glance overview of the entire platform.

### Connection Summary

A card grid showing the number of connections per cloud provider (GCS, S3, Azure, Alibaba, Huawei, Google Drive), each with the provider's color-coded icon.

### Activity (Last 24 Hours)

Counts of uploads, downloads, deletes, and transfers performed in the last 24 hours, pulled from the server-side audit log.

### Background Jobs

Current job counts broken down by status: pending, running, completed, failed.

### Shared Links

Number of active (non-expired) shared links and total download count across all links.

### Connection Details

A table listing every saved connection with its provider, name, and bucket.

Click **Refresh** to reload all dashboard data.

---

## Cross-Connection Search

Search for objects by prefix or path across all your connections at once.

### How to Search

1. Select a **Provider** from the dropdown (or leave as "All").
2. Optionally select a specific **Connection**.
3. Enter a **search query** (prefix or partial path).
4. Click **Search**.

### Results

Results appear in a table showing:

| Column | Description |
|---|---|
| Connection | Provider icon + connection name |
| Key | Full object path |
| Size | File size |
| Modified | Last-modified timestamp |

Click any result row to navigate directly to that file in the file browser.

> Search is performed server-side across all objects matching the prefix. It is not a full-text search of file contents.

---

## Shared Links

The shared links view lets you manage all generated download links from one place.

### Link List

Each shared link shows:

| Column | Description |
|---|---|
| Object | File name and storage path |
| Provider | Cloud provider badge |
| Downloads | Download count vs. maximum allowed (if set) |
| Expires | Expiry date/time — expired links are visually dimmed |
| Created | When the link was generated |

### Actions

- **Copy URL** — copies the public shareable URL (`/api/share/{token}`) to clipboard.
- **Revoke** — permanently deletes the shared link. A confirmation dialog appears first.
- **Refresh** — reloads the list.

> Shared links are created from the file browser's share icon. This view only manages existing links.

---

## Audit Log

A server-side audit trail of all operations performed through the platform.

### Log Entries

Each entry shows:

| Column | Description |
|---|---|
| Action | Color-coded badge: upload, download, delete, transfer |
| Provider | Which cloud provider was involved |
| Object | The file or path affected |
| Details | Additional context (e.g. destination for transfers) |
| IP | Client IP address |
| Timestamp | When the action occurred |

### Pagination

The log loads **100 entries** at a time. Click **Load more** at the bottom to fetch the next page.

> The audit log is stored server-side in SQLite and persists across sessions. It is independent of the client-side activity panel.

---

## Background Jobs

Monitor long-running operations like bulk transfers and sync tasks.

### Status Tabs

Filter jobs by status using the tab bar:

| Tab | Description |
|---|---|
| All | Every job regardless of status |
| Pending | Queued but not yet started |
| Running | Currently executing |
| Completed | Finished successfully |
| Failed | Encountered an error |

Each tab shows a count badge.

### Job Cards

Each job card displays:

- **Type** — the operation type (e.g. `bulk_transfer`)
- **Status** — color-coded badge (pending/running/completed/failed)
- **Progress bar** — percentage complete (for running jobs)
- **Timestamps** — created, started, and completed times
- **Error** — error message (for failed jobs)

### Job Details

Click any job card to expand a detail view with:

- Full status, progress, and timestamps
- Error message (if failed)
- Result data (JSON)
- Raw job payload (JSON)

### Auto-Refresh

Toggle the **auto-refresh** switch to poll for job updates every 5 seconds. Useful for monitoring running transfers.

---

## Webhooks

Configure HTTP webhook endpoints to receive real-time notifications when events occur.

### Creating a Webhook

1. Enter the **URL** of your webhook endpoint (must be `https://` in production).
2. Select one or more **events** to subscribe to:
   - `upload` — a file was uploaded
   - `download` — a file was downloaded
   - `delete` — a file was deleted
   - `transfer` — a file was transferred between connections
   - `share` — a shared link was created
3. Optionally enter a **Secret** — used to sign the webhook payload with HMAC-SHA256 so your endpoint can verify authenticity.
4. Click **Add Webhook**.

### Active Webhooks

All configured webhooks are listed with their URL and subscribed events. Click the **×** button to delete a webhook.

### Payload Format

When an event occurs, Anveesa Vestra sends a `POST` request to each matching webhook URL with a JSON body containing event details. If a secret is configured, the request includes an `X-Signature` header with the HMAC-SHA256 hex digest.

---

## API Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/api/analytics` | Dashboard summary data |
| POST | `/api/search` | Cross-connection object search |
| GET | `/api/shares` | List all shared links |
| DELETE | `/api/shares/{id}` | Delete a shared link |
| POST | `/api/share` | Create a shared link |
| GET | `/api/share/{token}` | Access a shared link (public) |
| GET | `/api/audit` | List audit log entries (`?limit=&offset=`) |
| GET | `/api/jobs` | List all jobs |
| GET | `/api/jobs/{id}` | Get job details |
| POST | `/api/jobs` | Create a background job |
| GET | `/api/webhooks` | List all webhooks |
| POST | `/api/webhooks` | Create a webhook |
| DELETE | `/api/webhooks/{id}` | Delete a webhook |

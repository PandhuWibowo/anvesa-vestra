# File Browser

The file browser is the main workspace in Anveesa Vestra. Click any connection in the sidebar to open it.

---

## Navigation

### Folders

Buckets are displayed as a hierarchical folder tree. Click any folder row to enter it. The **breadcrumb bar** below the connection name shows the current path. Click any segment in the breadcrumb to jump back to that level. Click **root** to return to the bucket root.

### Pagination

The browser loads **200 objects per page**. As you scroll to the bottom, the next page is fetched automatically (infinite scroll). A loading indicator appears in the last row during fetching.

### Navigate Up

Click any breadcrumb segment, or press `Backspace` to go up one level.

---

## Toolbar

| Control | Description |
|---|---|
| Search box | Filter visible files by name — press `/` to focus |
| Grid / List toggle | Switch between table view and grid/thumbnail view |
| New Folder | Create an empty folder (uploads a hidden `.keep` placeholder) |
| Upload | Pick files to upload to the current folder |
| Upload Folder | Pick a local folder to upload, preserving subfolder structure |
| Stats button | Toggle the bucket stats bar (object count + total size) |
| Refresh button | Reload the current folder listing (`r`) |
| Delete Connection | Remove this connection from the app |

---

## Uploading Files

### Click to Upload

Click **Upload** in the toolbar to open the system file picker. You can select multiple files. Files are uploaded to the **current folder prefix**.

### Drag-and-Drop

Drag files from your desktop and drop them anywhere on the file list area. A drop overlay appears to confirm the drop zone. Files go to the current folder.

### Upload Progress

While uploading, a **per-file progress panel** appears above the file list. Each row shows:

- File name
- Live percentage (`0%` → `100%`)
- ✓ when done, ✗ on error

The panel fades out 1.5 seconds after all uploads finish.

### Folder Upload

Click **Folder** in the toolbar to pick an entire local directory. The subfolder structure is preserved:

- A local file at `photos/2024/jan/img.jpg` uploads to `<current-prefix>photos/2024/jan/img.jpg`.
- Each file shows its relative path in the upload progress panel.

---

## Downloading Files

### Single File

Click the **download icon** in a file's action column. The backend generates a time-limited URL and opens it in a new tab:

| Provider | URL type | Default expiry |
|---|---|---|
| GCS | Signed URL (V4) | 15 minutes |
| AWS / R2 / MinIO | Presigned URL | 15 minutes |
| Huawei OBS | Presigned URL | 15 minutes |
| Alibaba OSS | Presigned URL | 15 minutes |
| Azure | SAS URL | 15 minutes |

> Signed/presigned URLs bypass public-access restrictions — the file does not need to be publicly readable.

### Zip Download

Download an entire folder or a multi-file selection as a `.zip` archive:

- **Folder zip** — click the **download icon** on any folder row. The backend lists all objects under the prefix and streams a zip.
- **Bulk zip** — select files with checkboxes, then click **Download as Zip** in the selection bar.

The archive filename matches the folder name (or `selection.zip` for multi-select).

---

## File Preview

Click the **eye icon** on any file row, or press `Space` on a focused row, to open the preview panel.

The preview supports **images, video, audio, PDF, Markdown, JSON, CSV/TSV, Excel, Word, code, and plain text** — with advanced features:

| Feature | Description |
|---|---|
| **Fullscreen mode** | Expand the panel to full viewport — press `f` or click the expand icon |
| **Image zoom** | Toolbar with +/−, Fit, 1:1 buttons. `Ctrl+Scroll` for mouse zoom (10%–500%) |
| **Progressive loading** | Text/code files load up to 5 MB. Content renders in 500-line chunks with infinite scroll |
| **Line numbers** | Toggle-able gutter for code, config, JSON, and text files |
| **Word wrap** | Toggle word wrap on/off from the toolbar |
| **Load all** | One-click button to load every remaining line at once |
| **CSV/TSV tables** | Parsed into sortable tables with up to 2,000 rows |
| **Excel sheets** | Multi-sheet tabs with up to 2,000 rows per sheet |
| **Word documents** | `.docx` rendered as formatted HTML |

The preview footer shows the file size, type label, and a **Download** button. Press `Escape` to exit fullscreen, then again to close the panel.

For the full reference, see [File Preview](./file-preview.md).

---

## Deleting Files

### Single File

Click the **trash icon** next to a file. A confirmation dialog appears; confirm to delete.

Press `Delete` on the keyboard to delete the currently focused row.

### Folder (Recursive)

Click the **trash icon** on a folder row. This performs a **recursive delete** — all objects under that prefix are removed. The confirmation dialog shows the folder name. A success toast reports how many files were removed.

### Bulk Delete

Select files with their checkboxes, then click **Delete all** in the selection bar. One confirmation covers the entire selection.

---

## Renaming / Moving Files

Click the **rename icon** (pencil) next to any file. Enter a new name in the dialog and click **Move**. The operation is implemented as copy + delete:

1. Object is copied to `current-prefix/new-name`.
2. Original is deleted.

The rename only changes the filename within the current folder. To move to a different folder, include the path in the new name field (e.g. `archive/old-report.pdf`).

---

## Multi-Select & Bulk Operations

Click any file's checkbox to select it. The **header checkbox** selects all visible files. When one or more files are selected, the **selection bar** appears:

| Action | Description |
|---|---|
| Count badge | Shows how many files are selected |
| Download all | Downloads each selected file sequentially (one signed URL per file) |
| Download as Zip | Streams a single `.zip` of all selected files |
| Delete all | Deletes all selected files with a single confirmation |
| ✕ button | Clears the selection |

---

## Sorting

Click any **column header** to sort:

| Column | Sorts by |
|---|---|
| Name | Alphabetical (case-insensitive) |
| Size | File size in bytes |
| Modified | Last-modified timestamp |

Click the same header again to reverse the order. Directories always appear before files regardless of sort.

---

## Search / Filter

Type in the **search box** in the toolbar to instantly filter the current folder by filename. The search is client-side and applies to the objects already loaded. Press `/` to focus the box from the keyboard, and `Escape` to clear it.

> The search only filters what is loaded in memory. Scroll down to load more objects before searching for items further in the list.

---

## Grid / Thumbnail View

Click the **grid icon** in the toolbar to switch from the default table view to a card grid. In grid view:

- Image files show a **thumbnail** fetched from the signed URL.
- Folders and non-image files show a generic icon.
- Clicking a card enters the folder or opens the preview panel.
- Multi-select checkboxes appear on hover.
- All keyboard shortcuts work normally.

Click the **list icon** to switch back to table view. The preference is stored in `localStorage`.

---

## Sharing Files (Presigned Link Generator)

Click the **share icon** (three dots connected by lines) on any file row to open the shareable link dialog.

1. Choose an expiry preset: **15 min**, **1 hour**, **24 hours**, or **7 days**.
2. Click **Generate Link** — the backend creates a signed URL with the selected expiry.
3. The URL appears in a copyable input. Click **Copy** to send it to the clipboard.
4. Click **Regenerate** to create a fresh link with a new expiry.

> The link gives read access to that specific file for the selected duration regardless of bucket permissions. It does not make the file permanently public.

---

## CLI Command Copy

Click the **terminal icon** on any file row to open the CLI commands dialog. It generates provider-specific commands for common operations:

| Command | Description |
|---|---|
| `download` | Copy the file to your local machine |
| `delete` | Remove the file from the bucket |
| `ls` | List objects in the current folder |

Click **Copy** next to any command to send it to the clipboard.

| Provider | Tool used |
|---|---|
| AWS / R2 / MinIO | `aws s3` |
| GCS | `gsutil` |
| Azure | `azcopy` |
| Alibaba OSS | `ossutil` |
| Huawei OBS | `obsutil` |

---

## Cross-Connection Transfer

Click the **send icon** on any file row to open the transfer dialog. This copies the file from the current connection to any other saved connection:

1. Pick a **Destination Connection** from the list.
2. Optionally enter a **Destination Prefix** (e.g. `backups/`) — leave blank to place the file at the root.
3. Click **Transfer**.

The file is downloaded server-side and re-uploaded to the destination. The original is not deleted. A success toast confirms the destination path.

---

## Metadata Editor

Click the **info icon** (ⓘ) on any file row to open the metadata panel.

| Field | Editable | Description |
|---|---|---|
| Content-Type | Yes | MIME type used when the file is served |
| Cache-Control | Yes | HTTP caching directive (e.g. `public, max-age=31536000`) |
| Custom metadata | Yes | Arbitrary key-value pairs stored on the object |
| Size | No | File size (formatted) |
| Last modified | No | UTC timestamp |
| ETag | No | Entity tag for cache validation |
| MD5 | No (GCS only) | Base64-encoded MD5 hash |

Click **+ Add** to add a custom metadata key-value pair. Click the **✕** on any row to remove it. Click **Save** to write changes back.

> For S3, OBS, and OSS, metadata updates are implemented as a copy-to-self with `MetadataDirective: REPLACE`, because these APIs don't allow in-place metadata edits.

---

## Bucket Statistics

Click the **bar-chart icon** in the browser header to toggle the stats bar. The stats are fetched once and cached until the connection changes or the page is refreshed.

| Stat | Description |
|---|---|
| Object count | Total number of objects in the bucket (may be estimated) |
| Total size | Sum of all object sizes, formatted as KB / MB / GB / TB |

If the bucket has more than 1,000 objects, the count is marked **(est.)**.

---

## Copying File Paths and Links

Each file row has two copy buttons:

| Button | What it copies |
|---|---|
| Clipboard icon | Storage protocol URL: `gs://`, `s3://`, or `az://` |
| Link icon | Presigned HTTP download link (15-minute expiry) |

---

## Creating Folders

Click the **folder+ icon** in the toolbar, enter a name, and click **Create**. The backend uploads a hidden `.keep` placeholder file inside the new folder to materialise the prefix. The folder appears immediately in the listing.

---

## Keyboard Shortcuts

The browser responds to the following keyboard shortcuts when focus is not in a text input:

| Key | Action |
|---|---|
| `j` / `↓` | Move highlight to the next row |
| `k` / `↑` | Move highlight to the previous row |
| `Enter` | Open the highlighted folder, or preview the highlighted file |
| `Space` | Toggle preview for the focused file |
| `f` | Toggle fullscreen preview |
| `d` | Download the highlighted file |
| `Delete` | Delete the highlighted file or folder (with confirmation) |
| `/` | Focus the search box |
| `r` | Refresh the current folder listing |
| `Backspace` | Navigate up one folder level |
| `Escape` | Exit fullscreen → close preview → clear search → deselect rows |

Row focus also tracks mouse hover, so j/k picks up from wherever the cursor is.

A hint strip at the bottom of the file list summarises all shortcuts.

---

## Pinned Connections

Click the **star icon** on any connection item in the sidebar to pin it. Pinned connections float to the top of the list above unpinned ones. The star fills in to indicate an active pin. The section label changes to **Pinned · All** when at least one connection is pinned. Pins are stored in `localStorage` and persist across sessions.

---

## Provider Badge & Icon

The browser header shows a colored badge indicating which provider the active connection uses:

| Provider | Badge color |
|---|---|
| Google Cloud Storage | Blue |
| Amazon S3 / R2 / MinIO | Amber |
| Huawei OBS | Red |
| Alibaba Cloud OSS | Orange |
| Azure Blob Storage | Sky blue |
| Google Drive | Blue |

---

## Sidebar Provider Filter

When you have connections from more than one provider, **filter chips** appear above the connection list. Click a chip to show only connections from that provider. Multiple chips can be active simultaneously. Click a chip again to remove its filter.

---

## Bookmarks

Bookmark any folder path for quick access from the sidebar.

### Creating a Bookmark

Click the **bookmark icon** (ribbon) in the browser header bar. The current connection + folder path is saved. A filled icon indicates the current location is already bookmarked; click again to remove it.

### Using Bookmarks

Bookmarks appear in a dedicated **Bookmarks** section in the sidebar, below the connection list. Each bookmark shows:

- The deepest folder name (or bucket name for root)
- The connection name

Click any bookmark to navigate directly to that connection and folder path.

Bookmarks are stored in `localStorage` and persist across sessions.

---

## Split View (Dual Pane)

Open two connections side by side for comparing or transferring files.

### Enabling Split View

Click the **split icon** in the sidebar toolbar. The main area divides into two panes with a vertical divider.

### Navigating Panes

In split mode, sidebar clicks alternate between the left and right panes:

1. First click opens a connection in the **left pane**.
2. Next click opens a connection in the **right pane**.
3. The cycle repeats.

Each pane has its own independent browser with full functionality (search, upload, sort, preview, etc.).

### Cross-Pane Drag-and-Drop

Drag a file row from one pane and drop it into the other pane to **copy** the file between connections. A drop overlay appears showing "Drop to copy here". The transfer goes through the server (download + re-upload) — the file is not deleted from the source.

### Disabling Split View

Click the **split icon** again to return to single-pane mode. The right pane is discarded.

---

## Activity Panel

A slide-out panel showing a chronological log of all actions performed during the current session.

### Opening the Panel

Click the **activity icon** (bell) in the sidebar toolbar. The panel slides in from the right edge.

### Event Types

The activity log tracks:

| Event | Dot color |
|---|---|
| Upload | Teal |
| Download | Green |
| Delete | Red |
| Transfer | Purple |
| Rename | Amber |
| Zip | Cyan |
| Folder create | Orange |
| Copy | Gray |

Each entry shows the action description, provider, and timestamp.

### Clearing

Click **Clear** at the top of the panel to remove all entries. The log is in-memory only — it resets on page refresh.

> For a persistent server-side audit trail, see [Audit Log](./management.md#audit-log).

---

## Navigation Persistence

Your current view and browsing location are saved to `localStorage` automatically. When you refresh the page or reopen the browser:

- If you were browsing a connection, the same connection and folder path are restored.
- If you were on a management view (dashboard, search, audit, etc.), that view is reopened.
- Transient states (new connection form, edit form) are not restored — you return to the welcome screen instead.

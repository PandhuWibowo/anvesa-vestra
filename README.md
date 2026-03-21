# Anveesa Vestra

A self-hosted, open-source cloud storage manager. Browse, upload, download, rename, preview, and manage files across **Google Cloud Storage**, **Amazon S3** (including Cloudflare R2 and MinIO), **Azure Blob Storage**, **Alibaba Cloud OSS**, **Huawei OBS**, and **Google Drive** from a single clean web interface.

**Website**: [anveesa.com](https://anveesa.com) | **GitHub**: [github.com/PandhuWibowo/anveesa-vestra](https://github.com/PandhuWibowo/anveesa-vestra)

---

## Quick Start

**Docker (recommended)**

```bash
docker run -d -p 80:80 \
  -e JWT_SECRET=your-secret-here \
  -e ENCRYPTION_KEY=your-encryption-key \
  -v anveesa-data:/data \
  pandhuwibowo/anveesa-vestra:latest
```

Open [http://localhost](http://localhost) and create your admin account.

**From source**

```bash
git clone https://github.com/PandhuWibowo/anveesa-vestra.git
cd anveesa-vestra
cd web && bun install && cd ..
make dev
```

Open [http://localhost:5173](http://localhost:5173). Create your admin account, then press **New Connection** to add your first bucket.

---

## Features

- **6 cloud providers** — GCS, S3/R2/MinIO, Azure, Alibaba OSS, Huawei OBS, Google Drive
- **Authentication** — JWT-based login with first-run admin setup
- **File preview** — images, PDF, video, audio, Markdown, JSON, CSV, Excel, Word, code
- **Advanced preview** — fullscreen, image zoom, progressive text loading, line numbers
- **Split view** — dual-pane browser with cross-pane drag-and-drop transfer
- **Cross-connection transfer** — copy files between any two providers
- **Bulk operations** — multi-select download, delete, zip, and bulk transfer as background jobs
- **Shared links** — generate time-limited public download URLs
- **Management views** — analytics dashboard, cross-connection search, audit log, jobs, webhooks
- **Bookmarks** — pin folder paths for quick sidebar access
- **Connection backup** — export/import all connections as JSON
- **Keyboard-first** — full keyboard navigation with shortcuts modal (`?`)
- **Dark mode** — system-aware theme toggle
- **Credential encryption** — AES encryption at rest for stored credentials

---

## Documentation

Full documentation lives in [`docs/`](./docs/) and is also available in-app via the Docs panel.

| Document | Description |
|---|---|
| [Overview](./docs/index.md) | Features, providers, and architecture |
| [Getting Started](./docs/getting-started.md) | Installation, auth setup, first connection |
| [Authentication](./docs/authentication.md) | Login, admin setup, JWT, security settings |
| [Managing Connections](./docs/connections.md) | Credential setup for all 6 providers |
| [File Browser](./docs/browser.md) | Browser features, split view, bookmarks, navigation |
| [File Preview](./docs/file-preview.md) | Fullscreen, zoom, progressive loading, line numbers |
| [Management Views](./docs/management.md) | Dashboard, search, shared links, audit, jobs, webhooks |
| [API Reference](./docs/api-reference.md) | Complete REST API (104 endpoints) |
| [Deployment](./docs/deployment.md) | Docker, binary, systemd, env vars |
| [Contributing](./docs/contributing.md) | Local dev setup, conventions, adding providers |

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.24, `net/http` |
| Storage SDKs | GCS, AWS S3, Azure Blob, Alibaba OSS, Huawei OBS, Google Drive API v3 |
| Database | SQLite (`modernc.org/sqlite`, pure Go) |
| Frontend | Vue 3, Vite 5, Composition API |
| Package manager | Bun (or npm) |
| Container | Docker + nginx + supervisord |

---

## Development Commands

```bash
make dev      # Start backend (port 8080) + frontend (port 5173)
make build    # Compile Go binary to bin/server and build web/dist/
```

---

## Community

- **Website**: [anveesa.com](https://anveesa.com) — announcements, guides, and community resources
- **GitHub**: [github.com/PandhuWibowo/anveesa-vestra](https://github.com/PandhuWibowo/anveesa-vestra) — source code, issues, and pull requests
- **Issues**: [Report a bug or request a feature](https://github.com/PandhuWibowo/anveesa-vestra/issues)

---

## License

Open source. See [LICENSE](./LICENSE) for details.

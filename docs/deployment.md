# Deployment

Anveesa Vestra can be deployed in two ways: as a **Docker container** (recommended — no toolchain required) or as a **native binary** built from source.

---

## Docker (Recommended)

A pre-built image is published to DockerHub on every push to `main`. It bundles the Go API server and the Vue frontend in a single container (nginx + supervisord).

### Pull and Run

```bash
docker run -d \
  --name anveesa-vestra \
  -p 80:80 \
  -e AUTH_ENABLED=true \
  -e JWT_SECRET=your-secret-here \
  -e ENCRYPTION_KEY=your-encryption-key \
  -v anveesa-data:/data \
  pandhuwibowo/anveesa-vestra:latest
```

Open [http://localhost](http://localhost) in your browser. On first launch, create your admin account (see [Authentication](./authentication.md)).

| Flag | Purpose |
|---|---|
| `-p 80:80` | Expose the app on host port 80 |
| `-v anveesa-data:/data` | Persist the SQLite database across restarts |

### Available Tags

| Tag | Description |
|---|---|
| `latest` | Latest build from `main` |
| `main` | Same as `latest` |
| `v1.2.3` | Specific release version |

### Container Management

```bash
# View live logs
docker logs -f anveesa-vestra

# Stop
docker stop anveesa-vestra

# Start again
docker start anveesa-vestra

# Remove container (data volume is kept)
docker rm -f anveesa-vestra
```

### Data Persistence

The SQLite database (`data.db`) is written to `/data` inside the container. The named volume `anveesa-data` ensures it survives container removal. To inspect or back it up:

```bash
# Find volume path on disk
docker volume inspect anveesa-data

# Backup
docker run --rm -v anveesa-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/anveesa-data-backup.tar.gz -C /data .
```

### Building the Image Locally

```bash
docker build -t anveesa-vestra:local .
docker run -d -p 80:80 -v anveesa-data:/data anveesa-vestra:local
```

---

## CI/CD — GitHub Actions → DockerHub

The repository includes a workflow at `.github/workflows/docker.yml` that automatically builds and pushes the image to DockerHub.

**Triggers:**
- Push to `main` → builds and pushes `latest` + `main` tags
- Version tag (`v1.2.3`) → also pushes semver tags (`1.2.3`, `1.2`)
- Pull requests → build-only (no push), to verify the image compiles

**Required secrets** (set in _Settings → Secrets → Actions_):

| Secret | Value |
|---|---|
| `DOCKERHUB_USERNAME` | Your DockerHub username |
| `DOCKERHUB_TOKEN` | DockerHub access token (not your password) |

The workflow builds multi-platform images (`linux/amd64` + `linux/arm64`) using Docker Buildx and caches layers via GitHub Actions cache for faster subsequent builds.

---

## Build from Source

Run the following from the project root:

```bash
make build
```

This command:
1. Compiles the Go backend to `bin/server`
2. Runs `bun run build` in `web/` to produce `web/dist/`

The resulting `bin/server` binary is self-contained for the backend. Copy it alongside the `web/dist/` folder.

---

### Running the Production Binary

```bash
./bin/server
```

The server listens on port **8080** by default. The SQLite database is created as `server/data.db` relative to the working directory on first run.

---

### Serving the Frontend

In development the Vite dev server proxies API requests. In production you have two options:

#### Option A — Reverse Proxy (Recommended)

Run Nginx (or Caddy) in front of the Go server:

```nginx
server {
    listen 80;
    server_name storage.example.com;

    # Serve built frontend
    root /var/www/anveesa-vestra/web/dist;
    index index.html;

    # API — proxy to Go backend
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # SPA fallback
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

Copy `web/dist/` to `/var/www/anveesa-vestra/web/dist/` after each build.

#### Option B — Serve from Go (Embedded)

Embed the `web/dist/` directory into the binary using Go's `embed` package. Add the following to `server/main.go`:

```go
import "embed"

//go:embed ../web/dist
var staticFiles embed.FS

// Then serve with http.FileServer(http.FS(staticFiles))
```

This packages everything into one binary — no separate static file deployment needed.

---

### Running as a System Service

#### systemd (Linux)

Create `/etc/systemd/system/anveesa-vestra.service`:

```ini
[Unit]
Description=Anveesa Vestra Cloud Storage Manager
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/anveesa-vestra
ExecStart=/opt/anveesa-vestra/bin/server
Environment=JWT_SECRET=your-secret-here
Environment=ENCRYPTION_KEY=your-encryption-key
Environment=DB_PATH=/opt/anveesa-vestra/data.db
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable anveesa-vestra
sudo systemctl start anveesa-vestra
```

#### macOS launchd

Create `~/Library/LaunchAgents/com.anveesa.vestra.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.anveesa.vestra</string>
  <key>ProgramArguments</key>
  <array>
    <string>/opt/anveesa-vestra/bin/server</string>
  </array>
  <key>WorkingDirectory</key>
  <string>/opt/anveesa-vestra</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/com.anveesa.vestra.plist
```

---

### Data Persistence

The SQLite database file (`server/data.db`) holds all saved connections. In production, ensure this path is:

- **Writable** by the user running the server process
- **Backed up** regularly — losing it means losing all saved connections (not bucket data)
- **Not publicly accessible** — it contains encrypted credentials

---

### Environment Reference

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listening port |
| `DB_PATH` | `data.db` | SQLite database file path |
| `CORS_ORIGIN` | `*` | Allowed CORS origin (set to your domain in production) |
| `AUTH_ENABLED` | `true` | Enable/disable login requirement |
| `JWT_SECRET` | `change-me-in-production` | HMAC-SHA256 signing key for JWT tokens |
| `JWT_EXPIRY` | `24h` | Token lifetime (e.g. `1h`, `12h`, `7d`) |
| `ENCRYPTION_KEY` | *(auto-generated)* | AES key for encrypting stored credentials |

> **Security**: Always set `JWT_SECRET` and `ENCRYPTION_KEY` to unique, strong values in production. The defaults are insecure and intended for development only.

See [Authentication](./authentication.md) for details on the auth system.

---

### Cross-Platform Builds

To build for a different OS/architecture from macOS:

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o bin/server-linux-amd64 ./server

# Linux arm64 (Raspberry Pi, AWS Graviton)
GOOS=linux GOARCH=arm64 go build -o bin/server-linux-arm64 ./server

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/server.exe ./server
```

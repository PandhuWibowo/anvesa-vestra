# Contributing to Anveesa Vestra

Thank you for your interest in contributing! This guide covers everything you need to get started.

## Development Setup

### Prerequisites

- **Go 1.24+** — backend
- **Bun** (or npm) — frontend package manager
- **Node.js 18+** — for Vite dev server

### Quick Start

```bash
# Clone the repository
git clone https://github.com/PandhuWibowo/oss-portable.git
cd oss-portable

# Install frontend dependencies
cd web && bun install && cd ..

# Start backend + frontend together
make dev
```

The backend runs on `http://localhost:8080` and the Vite dev server on `http://localhost:5173`. API requests are proxied automatically.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Backend server port |
| `DB_PATH` | `data.db` | SQLite database file path |
| `AUTH_ENABLED` | `true` | Enable JWT authentication |
| `JWT_SECRET` | `change-me-in-production` | JWT signing key |
| `ENCRYPTION_KEY` | *(none)* | 32+ char key for encrypting stored credentials |
| `TRUST_PROXY` | `false` | Trust X-Forwarded-For headers |
| `CORS_ORIGIN` | `*` | Allowed CORS origin |

## Project Structure

- `server/` — Go backend (`net/http`, no framework)
  - `handlers/` — HTTP handler files, one per provider + feature
  - `middleware/` — Auth, CORS, rate limiting
  - `db/` — SQLite init + migrations
  - `config/` — Environment-based config
- `web/` — Vue 3 frontend (Naive UI)
  - `src/composables/` — Shared state + API client logic
  - `src/components/` — UI components
- `docs/` — Product documentation (Markdown)
- `deploy/` — Docker, Nginx, supervisord config

## Adding a New Cloud Provider

1. **Backend**: Create `server/handlers/<provider>.go` following the `S3Provider` pattern in `s3compat.go` (or write standalone handlers like `gcp.go` for non-S3 providers)
2. **Database**: Add a migration in `server/db/db.go` to create the connection table
3. **Routes**: Register routes in `server/main.go`
4. **Frontend**: Add the provider to `useConnections.js` BASE map and `AddConnectionForm.vue`

## Code Conventions

- **Go**: Standard library HTTP patterns. Use `jsonError()` for all error responses (never `http.Error()`). Use `jsonOK()` for success responses.
- **Vue**: Composition API with `<script setup>`. Composables for shared logic.
- **Security**: Never expose credentials in API responses. All operations use `connection_id` to reference stored credentials.

## Running Tests

```bash
# Backend tests
cd server && go test ./...

# Frontend tests
cd web && bun run test
```

## Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Write tests for your changes
4. Ensure all tests pass
5. Submit a pull request with a clear description

## Database Migrations

Migrations are defined in `server/db/db.go` in the `migrations` slice. Each migration has a version number and SQL statement. Migrations are applied automatically on startup and tracked in the `schema_migrations` table.

To add a migration:
1. Add a new entry to the `migrations` slice with the next version number
2. Write the SQL (ALTER TABLE, CREATE INDEX, etc.)
3. Test locally by running the server

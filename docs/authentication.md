# Authentication

Anveesa Vestra includes an optional authentication system that protects the web interface and API behind a login screen.

---

## Overview

- **First-run setup** — on first launch, you create an admin account directly from the browser.
- **JWT-based** — after login, a signed token is stored in the browser and sent with every API request.
- **Optional** — authentication can be disabled entirely for private/trusted networks.
- **Single admin** — only the first registered user is accepted; subsequent registration attempts are rejected.

---

## First-Run Setup

When you open Anveesa Vestra for the first time (or after a fresh database), the login screen shows a **Create Admin Account** form instead of the normal login.

1. Enter a **username**.
2. Enter a **password** (minimum 8 characters).
3. Confirm the password.
4. Click **Create Account**.

After registration, you are automatically redirected to the login form.

> Only one admin account can be created. If you need to reset credentials, delete the `data.db` SQLite file and restart the server.

---

## Login

After the admin account exists, the login screen shows a standard username/password form.

1. Enter your credentials.
2. Click **Sign In**.
3. On success, a JWT token is stored in `localStorage` and you are taken to the main interface.

The token expires after **24 hours** by default (configurable via `JWT_EXPIRY`).

---

## Session Persistence

- The JWT token persists across browser tabs and page refreshes.
- On each page load, the app calls `GET /api/auth/me` to verify the token is still valid.
- If the token is expired or invalid, you are returned to the login screen.

---

## Logout

Click the **Sign Out** button in the sidebar footer. This clears the token from `localStorage` and returns to the login screen.

---

## Disabling Authentication

Set the environment variable `AUTH_ENABLED=false` to skip authentication entirely. When disabled:

- No login screen is shown.
- All API endpoints are accessible without a token.
- The sidebar does not show the username or sign-out button.

This is useful for local development or private networks where access control is handled at the network level.

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `AUTH_ENABLED` | `true` | Enable or disable the login requirement |
| `JWT_SECRET` | `change-me-in-production` | HMAC-SHA256 signing key for JWT tokens |
| `JWT_EXPIRY` | `24h` | Token expiry duration (e.g. `1h`, `7d`) |
| `ENCRYPTION_KEY` | *(generated)* | AES key used to encrypt stored connection credentials |

> **Important**: Change `JWT_SECRET` and `ENCRYPTION_KEY` in production. The default values are insecure.

---

## API Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/auth/setup-status` | Public | Check if auth is enabled and if initial setup is needed |
| POST | `/api/auth/register` | Public | Create the first admin account |
| POST | `/api/auth/login` | Public | Authenticate and receive a JWT |
| GET | `/api/auth/me` | Protected | Validate token and return current user info |

---

## Security Notes

- Passwords are hashed with bcrypt before storage.
- Connection credentials are encrypted at rest using AES with the `ENCRYPTION_KEY`.
- All protected API endpoints require a valid `Authorization: Bearer <token>` header.
- A **rate limiter** (20 requests/second per IP, burst of 60) is applied to all API endpoints to prevent brute-force attacks.
- RBAC (role-based access control) middleware exists in the codebase but is not yet enforced on routes.

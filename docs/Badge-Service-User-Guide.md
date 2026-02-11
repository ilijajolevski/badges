### Badge Service — User & Operator Guide

This document explains how the Badge/Certificate service works and how to use it both as an issuer (operator) and a recipient (viewer). It covers authentication and authorization, database schema, API keys, caching, generation pipeline, error handling, deployment, data migration/seed data, dependencies, and other practical topics.

#### 1. Authentication & Authorization Architecture

- Identity model:
  - Users and Roles are stored in the database (`users`, `roles` tables).
  - A default `admin` role with full permissions is created on first run.
  - A default admin user is created if there are no users: username `admin`, password `Admin@123`, email `admin@example.com`.
- Permissions:
  - Role permissions are stored as JSON in `roles.permissions` and embedded into JWT claims on login.
  - Permissions are grouped by resource: `badges`, `users`, `api_keys` with `read/write/delete` flags.
- JWT-based sessions:
  - Login endpoint: `POST /api/auth/login` with JSON `{"username": "...", "password": "..."}`.
  - On success, a JWT is returned in the body and set as an `HttpOnly` cookie `jwt` (15-minute expiry). For production, enable `Secure` cookie attribute.
  - Session info: `GET /api/auth/session` returns current user/role if cookie is present.
  - Logout: `POST /api/auth/logout` clears the cookie.
- Middleware:
  - `OptionalJWTFromCookie` injects claims when a cookie is present (used by pages like `/details/`, `/certificates`, `/edit/`).
  - `JWTAuthMiddleware` enforces a valid token (used for protected APIs like `/api/keys`).
  - `RequirePermissionMiddleware(resource, action)` enforces fine-grained permissions (e.g., write permission for `/certificates/new`).

Recipient experience (no login required):
- Public assets and pages are accessible without authentication: `/`, `/badge/{commit_id}`, `/certificate/{commit_id}`, `/details/{commit_id}`, `/certificates`.

Issuer/operator experience (login required for protected actions):
- Login via `/api/auth/login`, then use browser (cookie session) or pass the bearer token for API calls.
- Create and manage content through admin/edit/create routes guarded by middleware.

#### 2. Database Schema Details

The service uses SQLite. Schema is initialized automatically at startup (`internal/database/database.go`). Key tables:

- `badges` (primary content)
  - `commit_id` TEXT PRIMARY KEY
  - `type`, `status`, `issuer`, `issue_date`
  - `software_name`, `software_version`, `software_url`
  - `notes`, `public_note`, `internal_note`, `contact_details`
  - `covered_version`, `repository_link`
  - `certificate_name`, `specialty_domain`, `issuer_url`
  - `custom_config` TEXT (JSON with display customizations)
  - `svg_content` TEXT; `jpg_content` BLOB; `png_content` BLOB (generated and cached image content)
  - `expiry_date`, `last_review`, `software_sc_id`, `software_sc_url`

- `roles`
  - `role_id` TEXT PRIMARY KEY
  - `name` UNIQUE, `description`
  - `permissions` TEXT (JSON with resource actions)
  - `created_at`, `updated_at`

- `users`
  - `user_id` TEXT PRIMARY KEY; `username` UNIQUE; `email` UNIQUE
  - `password_hash` (bcrypt)
  - `first_name`, `last_name`, `role_id` (FK to `roles`)
  - `status` (e.g., `active`, `locked`), `failed_attempts` (lockout after 5 failed logins)
  - `created_at`, `updated_at`, `last_login`

- `api_keys`
  - `api_key_id` TEXT PRIMARY KEY; `user_id` (FK to `users`)
  - `api_key` TEXT (stored as a hash; the raw key is shown only once on creation)
  - `name`, `permissions` (JSON), `ip_restrictions` (JSON array)
  - `created_at`, `expires_at`, `last_used`, `status`

Initial Data:
- Default `admin` role and a default `admin` user are inserted if empty.
- Initial sample badges are loaded from `db/initial_badges.json`.

#### 3. API Key Management

- API keys are created per user and stored hashed. Only the plaintext key shown once in the create response.
- IP restrictions: optional list of allowed CIDRs/addresses stored in `api_keys.ip_restrictions`.
- Permissions: stored in `api_keys.permissions` JSON (currently scoped to `badges.read/write`).
- Endpoints (current routing):
  - List keys: `GET /api/keys` (requires valid JWT). Shows keys for the authenticated user, with metadata.
  - Create/Revoke/Update handlers exist in code, but only `GET /api/keys` is wired in current server routes. Operators may wire `POST/PUT/DELETE /api/keys` later if needed.
- Usage:
  - For backend-to-backend calls, send the API key as `Authorization: ApiKey <key>` if an API-key protected endpoint is introduced. The current product primarily uses JWT for operator flows.

#### 4. Cache Architecture / Implementation / Operation

- In-memory cache (`internal/cache`): simple thread-safe map with TTL per item and a janitor goroutine that purges expired items every minute.
- Used by badge/certificate handlers to cache rendered images/content by computed keys (e.g., commit + format + options).
- Invalidation:
  - Explicit `Set/Delete/Clear` APIs exist. On edits or updates, handlers delete/overwrite relevant cache entries.
  - Expiry-based eviction ensures stale items are eventually removed.

#### 5. Badge/Certificate Generation Pipeline

- Data source: a `badges` row identified by `commit_id`.
- Customization:
  - `custom_config` JSON per badge stores defaults such as `color_left`, `color_right`, `text_color`, `text_color_left/right`, `logo`, `font_size`, `style`.
  - Query parameters can override display at request time (e.g., `?color_right=%23ff9900&style=3d`).
- Templates:
  - SVG templates under `templates/svg/` for small badges and big certificates.
  - HTML templates for pages: `templates/details/`, `templates/list/`, `templates/home/`, `templates/edit/`, `templates/admin/`.
- Rendering flow:
  1. Handler loads badge from DB (`internal/badge` or `internal/certificate`).
  2. Merge `custom_config` with query param overrides.
  3. Generate SVG; optionally rasterize to PNG/JPG if requested; cache the result.
  4. Return the image/content with appropriate headers.

Recipient tips:
- To view a certificate: open `/certificate/{commit_id}` for a large printable layout; `/details/{commit_id}` shows metadata; `/badge/{commit_id}` shows the small badge.
- Some styling can be adjusted via query params if allowed by the issuer.

Issuer tips:
- Use `/certificates` to browse existing entries.
- Use `/certificates/new` to create new entries (requires login and `badges.write` permission).
- Use `/edit/{commit_id}` to update metadata or `custom_config` JSON.

#### 9. Error Handling

- Centralized error handling middleware renders a friendly HTML error page for non-2xx responses and logs the error context with `zap`.
- JSON APIs respond with consistent JSON error payloads like `{"error": "message"}` and an appropriate HTTP status.
- Input sanitization middleware normalizes/validates inputs to reduce reflected data issues.
- Rate limiter protects endpoints from abuse (per-client IP over a sliding time window).

#### 12. Deployment

- Container image: multi-stage Dockerfile builds a static binary (`./cmd/server`) and packages it in Alpine with `librsvg` for SVG operations.
- Runtime configuration via environment variables:
  - `PORT` (default 8080)
  - `LOG_LEVEL` (`production` or `development`)
  - `DB_PATH` (path to SQLite DB file; container default `/app/db/badges.db`)
- Volumes: persist `/app/db` to retain data. Only `initial_badges.json` is copied into the image; the DB file is created at runtime.
- Health: logs on start will list available badges. Exposes port 8080.

#### 14. Data Migration & Seed Data

- On startup, the service auto-creates tables if missing and seeds:
  - `roles`: inserts `admin` role with full permissions if absent.
  - `users`: inserts default `admin` user if empty.
  - `badges`: loads samples from `db/initial_badges.json` if missing.
- For upgrades: because SQLite is used and the schema is created programmatically, introduce migrations by versioning schema changes in code or adding a migration step before `initDB`.

#### 15. Dependencies

- Go 1.24.x
- Core libraries:
  - `go.uber.org/zap` — structured logging
  - `github.com/golang-jwt/jwt/v5` — JWT handling
  - `github.com/mattn/go-sqlite3` — SQLite driver (CGO enabled)
  - `golang.org/x/crypto` — bcrypt password hashing
  - `github.com/disintegration/imaging`, `golang.org/x/image` — image processing

#### Additional Important Areas

- Rate Limiting:
  - Configured in server with a default limit (e.g., 100 requests/minute). Adjust in `cmd/server` when initializing `RateLimiter`.

- Logging:
  - Development vs production logger configuration determined by `LOG_LEVEL`. All requests are wrapped by a request logger recording method, path, status, and latency.

- Security Notes:
  - Change the default admin password immediately in production.
  - Set a strong JWT secret via environment and configure it at startup (code supports `auth.SetJWTSecret`). Enable `Secure` cookie in HTTPS.
  - API keys are hashed at rest. Treat the returned plaintext key as sensitive.
  - Consider enabling CORS and CSRF protections if adding browser-side forms/APIs.

- UI Pages Overview:
  - `/` — Home page.
  - `/certificates` — List view with search/filter.
  - `/details/{commit_id}` — Detailed view; shows metadata; optionally tailored if logged in.
  - `/edit/{commit_id}` — Edit form with optional `custom_config` JSON.
  - `/admin` — Administrative dashboard components.

- How to Start Locally
  - Prereqs: Go toolchain with CGO, or Docker.
  - Env: `PORT`, `DB_PATH`, `LOG_LEVEL`. Example: `PORT=8080 LOG_LEVEL=development DB_PATH=./db/badges.db`.
  - Run: `go run ./cmd/server` or `make run` (if available). Docker: `docker build -t badge-service . && docker run -p 8080:8080 -v $(pwd)/db:/app/db badge-service`.

- How to Explore the Database
  - The service stores data in SQLite at `DB_PATH`. You can open the file with any SQLite browser.
  - Utility: `cmd/dbtest` can be run to list existing badges and inspect `custom_config` fields.

- Troubleshooting
  - If images don’t render: check cache invalidation and the stored SVG/JPG/PNG columns; ensure `librsvg` exists in container (Dockerfile installs it).
  - If login fails: verify default admin exists and that `jwt` cookie is set on successful login; check server logs.
  - If protected routes return 403: ensure your role permissions include the required resource/action; check `RequirePermissionMiddleware` usage for the route.

#### Public API/Route Summary

- Public:
  - `GET /` — home
  - `GET /badge/{commit_id}` — small badge
  - `GET /certificate/{commit_id}` — large certificate
  - `GET /details/{commit_id}` — details page
  - `GET /certificates` — list
  - `GET /static/*`, favicon routes
- Auth:
  - `POST /api/auth/login` — login (returns JWT, sets cookie)
  - `POST /api/auth/logout` — logout (clears cookie)
  - `GET /api/auth/session` — current session
- Operator APIs:
  - `GET /api/keys` — list API keys (JWT required)
  - `POST /certificates/new` (HTML form flow) — create certificate (JWT cookie + `badges.write` permission)

Notes:
- Additional API key management endpoints exist in code (create/update/revoke) but are not wired in the default router. Wire them before exposing to clients.

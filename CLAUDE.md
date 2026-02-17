# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CertifyHub is a Go web service that serves GÉANT software certificates as SVG/PNG/JPG badge images, identified by a unique short certificate ID. It provides both small inline badges and large certificate visuals, plus HTML detail pages and an admin interface.

## Build & Development Commands

```bash
make deps          # Install Go dependencies
make db-init       # Create the db/ directory
make run           # Run locally (default port 9000, override with PORT=8080 make run)
make build         # Build binary to bin/badge-service
make test          # Run all tests (go test -v ./...)
make build-image   # Build Docker image
```

Run a single test:
```bash
go test -v -run TestFunctionName ./internal/badge/
```

The service requires **CGO** (sqlite3 driver) and **librsvg** (`rsvg-convert`) for SVG-to-PNG/JPG conversion.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `80` | Server port |
| `LOG_LEVEL` | `development` | `development` or `production` (zap) |
| `DB_PATH` | `./db/badges.db` | SQLite database path |

## Architecture

**Entry point:** `cmd/server/main.go` — initializes config, logger, DB, cache, all handlers, registers routes on `net/http.ServeMux`, starts server with graceful shutdown.

**No web framework.** Uses stdlib `net/http` with a hand-rolled middleware chain (request logger → error handler → rate limiter → sanitizer → [optional auth] → handler).

### Internal packages (each under `internal/`)

| Package | Purpose |
|---------|---------|
| `badge/` | Small inline badge SVG generation (`Generator`) + HTTP handler |
| `certificate/` | Large certificate SVG generation (`Generator`) + HTTP handler |
| `details/` | HTML detail page for a certificate |
| `list/` | HTML list page showing all certificates |
| `home/` | Home page handler |
| `admin/` | Admin page handler |
| `edit/` | Edit certificate handler |
| `create/` | Create new certificate handler |
| `auth/` | JWT auth (cookie-based for browsers), API key auth, password hashing (bcrypt), auth middleware |
| `apikey/` | API key management handler |
| `database/` | SQLite via `mattn/go-sqlite3`. Models (`Badge`, `User`, `Role`, `APIKey`) and all CRUD operations. Schema auto-created on startup in `initDB()`. |
| `cache/` | In-memory cache with TTL and background janitor |
| `config/` | Loads config from environment variables |
| `middleware/` | `ErrorHandler`, `Sanitizer` (validates commit ID format), `RateLimiter`, `RequestLogger` |

### Other directories

- `pkg/utils/` — SVG-to-PNG/JPG conversion using `rsvg-convert` + `imaging` library
- `templates/svg/` — SVG templates (`small-template.svg`, `big-template.svg`) parsed by Go `html/template`
- `templates/` — HTML templates for web pages (home, admin, details, edit, list, error)
- `static/` — CSS, logos, favicons
- `db/initial_badges.json` — Seed data loaded on first startup if badges don't exist

### Key data flow

1. Request hits `/badge/<id>` or `/certificate/<id>` → middleware chain → badge/certificate handler
2. Handler looks up `Badge` from SQLite by commit ID
3. `Generator.GenerateSVG()` reads the SVG template file, merges badge data via Go templates, returns SVG bytes
4. For PNG/JPG: SVG is piped through `rsvg-convert` then processed with `imaging` library
5. Results are cached in-memory with TTL

### Auth model

- **Browser auth:** JWT stored in HTTP-only cookie (15-min expiry). `OptionalJWTFromCookie` injects claims into context; `RequirePermissionMiddleware` enforces access.
- **API auth:** API keys with per-key permissions (badges read/write).
- **RBAC:** Roles with JSON permissions covering badges, users, and api_keys (read/write/delete each).
- Default admin user created on first startup (username: `admin`, password: `Admin@123`).

### Routes

- `GET /badge/<id>` — Small SVG badge (supports `?format=svg|png|jpg`)
- `GET /certificate/<id>` — Large SVG certificate
- `GET /details/<id>` — HTML details page
- `GET /certificates` — List all certificates
- `GET /certificates/new` — Create form (requires auth + write permission)
- `GET /edit/<id>` — Edit form (requires auth)
- `GET /admin` — Admin page
- `POST /api/auth/login` — Login endpoint
- `POST /api/auth/logout` — Logout endpoint
- `GET /api/auth/session` — Session info
- `GET /api/keys` — List API keys (requires JWT auth)

### Commit ID format

Certificate IDs must match `^[a-zA-Z0-9_]{6,40}$` (enforced by the sanitizer middleware).

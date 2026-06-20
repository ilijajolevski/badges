# SoftwareCertHub

**Status:** Development &nbsp;|&nbsp; **Latest stable:** v0.1.2

Copyright (c) 2024-2026 GÉANT Association &nbsp;|&nbsp;
Licensed under the [Apache-2.0 License](LICENSE)

**Tags:** badges, certificates, svg, go, geant, software-licensing, compliance, verification, rest-api

A service for serving GÉANT software licensing certificates as SVG/PNG/JPG badge
images, identified by a unique short certificate ID. It provides both small
inline badges and large certificate visuals, plus HTML detail pages and an admin
interface.

## Features

- Serve SVG/PNG/JPG images for pre-issued certificates
- Provide a details page for each certificate
- Customizable badge styles via query parameters
- Admin interface for creating and managing certificates
- JWT (cookie-based) browser auth and API-key auth with role-based permissions
- Input sanitization and validation

## Scope

**Use SoftwareCertHub when you want to:**

- Display GÉANT software licensing certificates as embeddable badges in
  repositories, READMEs, documentation, and web pages.
- Provide a public, verifiable detail page for each issued certificate.
- Manage the certificate lifecycle (create, edit, list) through an authenticated
  admin interface.

**SoftwareCertHub is _not_:**

- A certificate authority (CA) or PKI — it does not issue X.509 certificates or
  perform cryptographic signing.
- A general-purpose image host — IDs must match the certificate ID format and
  map to records in the database.

**Requirements & constraints:** certificate IDs must match
`^[a-zA-Z0-9_]{6,40}$`; PNG/JPG output requires `librsvg` (`rsvg-convert`) to be
installed on the host.

## Compatibility

| Area | Supported |
|------|-----------|
| Language/runtime | Go 1.24+ (CGO required for the SQLite driver) |
| Database | SQLite 3 (via `mattn/go-sqlite3`) |
| OS | Linux, macOS (anywhere Go + librsvg run) |
| External tools | `librsvg` / `rsvg-convert` for SVG→PNG/JPG conversion |
| Containers | Docker (image exposes port `8080`) |
| Badge rendering | Any modern browser or client that renders SVG/PNG/JPG |
| Integration targets | Markdown/README files, HTML pages, GÉANT Software Catalogue (planned) |

## Architecture & Project Structure

No web framework is used — the service is built on the Go stdlib `net/http`
with a hand-rolled middleware chain (request logger → error handler → rate
limiter → sanitizer → optional auth → handler).

| Path | Purpose |
|------|---------|
| `cmd/server/` | Entry point: config, logger, DB, cache, handlers, routes, graceful shutdown |
| `internal/badge/` | Small inline badge SVG generation + HTTP handler |
| `internal/certificate/` | Large certificate SVG generation + HTTP handler |
| `internal/details/` | HTML detail page for a certificate |
| `internal/list/` | HTML list page of all certificates |
| `internal/home/`, `internal/admin/` | Home and admin page handlers |
| `internal/edit/`, `internal/create/` | Edit / create certificate handlers |
| `internal/auth/` | JWT (cookie) auth, API-key auth, bcrypt hashing, auth middleware |
| `internal/apikey/` | API key management handler |
| `internal/database/` | SQLite models (`Badge`, `User`, `Role`, `APIKey`) and CRUD |
| `internal/cache/` | In-memory cache with TTL and background janitor |
| `internal/config/` | Configuration loaded from environment variables |
| `internal/middleware/` | Error handler, sanitizer, rate limiter, request logger |
| `pkg/utils/` | SVG→PNG/JPG conversion (`rsvg-convert` + `imaging`) |
| `templates/svg/`, `templates/` | SVG and HTML templates |
| `static/` | CSS, logos, favicons |
| `db/` | SQLite database and seed data (`initial_badges.json`) |

## API Endpoints

### Badge Endpoint

```
GET /badge/<commit_id>
```

Returns an SVG for a small inline badge. Used in `<img>` tags.

Query parameters:
- `format=svg|jpg|png`: Specifies the image format (default: `svg`)
- `color_left=<hex>`: Custom left section color
- `color_right=<hex>`: Custom right section color
- `text_color=<hex>`: Custom text color
- `logo=<url>`: URL of a logo image for the left section
- `font_size=<px>`: Custom font size
- `style=<flat|3d>`: Badge style

### Certificate Endpoint

```
GET /certificate/<commit_id>
```

Returns an SVG for a large certificate. Used in `<object>` tags. Supports the
same query parameters as the badge endpoint.

### Details Page Endpoint

```
GET /details/<commit_id>
```

Returns an HTML page with details about the certificate.

## System Requirements

- Go 1.24 or higher (with CGO enabled)
- SQLite 3
- librsvg (`rsvg-convert`) for SVG to PNG/JPG conversion

## Installation

### Local Development

1. Clone the repository:
   ```
   git clone https://github.com/ilijajolevski/badges.git
   cd badges
   ```

2. Install dependencies:
   ```
   make deps
   ```

3. Initialize the database directory:
   ```
   make db-init
   ```

4. Run the service:
   ```
   make run
   ```

The service will be available at http://localhost:9000 (the `make run` default;
override with `PORT=8080 make run`).

### Docker

1. Build the Docker image:
   ```
   make build-image
   ```

2. Run the Docker container (the image listens on port `8080`):
   ```
   docker run -p 8080:8080 -v $(pwd)/db:/app/db badge-service:latest
   ```

## Usage

### Live demo

A running instance is available at
[certificates.software.geant.org](https://certificates.software.geant.org/).

### Embed a small inline badge

```html
<a href="https://certificates.software.geant.org/details/abc123">
    <img src="https://certificates.software.geant.org/badge/abc123" alt="Badge">
</a>
```

### Embed a large certificate

```html
<a href="https://certificates.software.geant.org/details/abc123">
    <object data="https://certificates.software.geant.org/certificate/abc123" type="image/svg+xml" width="400" height="300">
        Certificate
    </object>
</a>
```

Small (inline) badge look of the certificate:

[![Badge Service v1.0.0 Badge](https://certificates.software.geant.org/badge/SOFTCAT_slSAD)](https://certificates.software.geant.org/details/SOFTCAT_slSAD)

Big certificate look:

[![Badge Service v1.0.0 Badge](https://certificates.software.geant.org/certificate/SOFTCAT_slSAD)](https://certificates.software.geant.org/details/SOFTCAT_slSAD)

### Fetch a badge from the command line

```bash
curl -o badge.svg "http://localhost:9000/badge/SOFTCAT_slSAD"
curl -o badge.png "http://localhost:9000/badge/SOFTCAT_slSAD?format=png"
```

See the [User Guide](docs/Badge-Service-User-Guide.md) for full usage details.

## Configuration

The service can be configured using environment variables:

- `PORT`: The port to listen on (default: `80`; `make run` uses `9000`; the
  Docker image uses `8080`)
- `LOG_LEVEL`: The log level — `development` or `production` (default:
  `development`)
- `DB_PATH`: The path to the SQLite database (default: `./db/badges.db`)
- `ADMIN_PASSWORD`: Password for the default `admin` user, created on first
  startup when no users exist (default: `Admin@123`)

> **Note:** `ADMIN_PASSWORD` only takes effect when the default admin user is
> first created (i.e. on an empty database). Changing it later has no effect on
> an existing admin account — update the password through the admin interface
> instead. Set this in production to avoid using the well-known default.

## Documentation

- [User Guide](docs/Badge-Service-User-Guide.md) — usage and integration details
- [CHANGELOG.md](CHANGELOG.md) — release history
- [CONTRIBUTING.md](CONTRIBUTING.md) — contribution guidelines
- [CLAUDE.md](CLAUDE.md) — architecture and developer notes
- GÉANT branding assets under [docs/GEANT_Branding](docs/GEANT_Branding)

## Troubleshooting & FAQ

- **PNG/JPG requests fail or return an error.** `librsvg` (`rsvg-convert`) is not
  installed or not on `PATH`. Install it (e.g. `brew install librsvg` or
  `apt-get install librsvg2-bin`).
- **Build fails with CGO / sqlite errors.** The SQLite driver requires CGO.
  Ensure a C toolchain is available and `CGO_ENABLED=1` (the default).
- **Cannot log in to the admin interface.** On an empty database the default user
  is `admin` with the password from `ADMIN_PASSWORD` (default `Admin@123`).
- **"address already in use" on startup.** Another process holds the port —
  choose a different one, e.g. `PORT=9001 make run`.

## Version Control

Repository (GitHub): https://github.com/ilijajolevski/badges
Repository (GÉANT GitLab): https://gitlab.software.geant.org/software-licensing/softwarecerthub
Main branch: `main`
Releases: tagged as `vMAJOR.MINOR.PATCH` following [Semantic Versioning](https://semver.org/)
Changelog: see [CHANGELOG.md](CHANGELOG.md)

## Roadmap

- Automatic visibility of certificates in the GÉANT Software Catalogue
- Continued improvements to usability, interoperability, and integration

Planned features and known issues are tracked on the issue trackers:

- GitHub: https://github.com/ilijajolevski/badges/issues
- GÉANT GitLab: https://gitlab.software.geant.org/software-licensing/softwarecerthub/-/issues

## Privacy

SoftwareCertHub processes minimal data:

- It stores **only administrator account credentials** (username, optional email,
  and a bcrypt-hashed password) required for authentication.
- It does **not** collect or process personal data of certificate viewers or site
  visitors, and uses **no third-party analytics or tracking**.
- Certificate records contain software/project metadata, not personal data.

## Authors

See [AUTHORS](AUTHORS) for the list of developers and contributors.

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) for
guidelines on reporting issues and submitting changes.

## Support

- Issue tracker (GitHub): https://github.com/ilijajolevski/badges/issues
- Issue tracker (GÉANT GitLab): https://gitlab.software.geant.org/software-licensing/softwarecerthub/-/issues

## Funding

This software was developed as part of the GÉANT project (GN5-1), co-funded by
the European Union's Horizon Europe research and innovation programme under Grant
Agreement No. 101100680. The work is carried out by the GÉANT Association on
behalf of the GN5-1 project.

![Co-funded by the European Union](eu-logo.png)

## Acknowledgements

SoftwareCertHub is developed under the **GÉANT** project and is **co-funded by
the European Union**. We gratefully acknowledge the open-source projects this
service builds on:

- [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) — SQLite driver (MIT)
- [golang-jwt/jwt](https://github.com/golang-jwt/jwt) — JWT handling (MIT)
- [uber-go/zap](https://github.com/uber-go/zap) — structured logging (MIT)
- [disintegration/imaging](https://github.com/disintegration/imaging) — image processing (MIT)
- [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) and
  [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) (BSD-3-Clause)
- [librsvg](https://wiki.gnome.org/Projects/LibRsvg) — SVG rasterisation (LGPL-2.1+)

## Dependencies

Main third-party dependencies (full details and copyright in the [NOTICE](NOTICE)
file):

| Component | Purpose | Licence |
|-----------|---------|---------|
| [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) | SQLite database driver | MIT |
| [golang-jwt/jwt](https://github.com/golang-jwt/jwt) | JWT authentication | MIT |
| [go.uber.org/zap](https://github.com/uber-go/zap) | Structured logging | MIT |
| [go.uber.org/multierr](https://github.com/uber-go/multierr) | Error aggregation | MIT |
| [disintegration/imaging](https://github.com/disintegration/imaging) | Image processing | MIT |
| [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) | bcrypt password hashing | BSD-3-Clause |
| [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) | Image format support | BSD-3-Clause |
| [librsvg](https://wiki.gnome.org/Projects/LibRsvg) (runtime tool) | SVG→PNG/JPG conversion | LGPL-2.1+ |

## Licence

This project is licenced under the **Apache-2.0 Licence**. You may use, modify,
and distribute it under the terms of that licence — see the [LICENSE](LICENSE)
file for the full text and the [NOTICE](NOTICE) file for third-party attributions.

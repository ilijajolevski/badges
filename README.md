# CertifyHub Service

A service for serving pre-issued certificates as SVG images, identified by a unique short certificate ID.

## Features

- Serve SVG images for pre-issued certificates
- Provide a details page for each certificate
- Customizable badge styles via query parameters
- Input sanitization and validation

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

Returns an SVG for a large certificate. Used in `<object>` tags.

Supports the same query parameters as the badge endpoint.

### Details Page Endpoint

```
GET /details/<commit_id>
```

Returns an HTML page with details about the certificate.

## Building and Running

### Prerequisites

- Go 1.21 or higher
- SQLite
- librsvg (for SVG to PNG/JPG conversion)

### Local Development

1. Clone the repository:
   ```
   git clone https://github.com/finki/badges.git
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

The service will be available at http://localhost:8080.

### Docker

1. Build the Docker image:
   ```
   make build-image
   ```

2. Run the Docker container:
   ```
   docker run -p 8080:8080 -v $(pwd)/db:/app/db finki/badge-service:latest
   ```

## Configuration

The service can be configured using environment variables:

- `PORT`: The port to listen on (default: 8080)
- `LOG_LEVEL`: The log level (default: development)
- `DB_PATH`: The path to the SQLite database (default: ./db/badges.db)

## Integration

To embed a small inline badge in your HTML:

```html
<a href="https://certificates.software.geant.org/details/abc123">
    <img src="https://certificates.software.geant.org/badge/abc123" alt="Badge">
</a>
```

To embed a bigger certificate in your HTML:

```html
<a href="https://certificates.software.geant.org/details/abc123">
    <object data="https://certificates.software.geant.org/certificate/abc123" type="image/svg+xml" width="400" height="300">
        Certificate
    </object>
</a>
```
Small (inline) Badge look of the certificate:

[![Badge Service v1.0.0 Badge](https://certificates.software.geant.org/badge/test123)](https://certificates.software.geant.org/details/test123)


Big Certificate Look:

[![Badge Service v1.0.0 Badge](https://certificates.software.geant.org/certificate/test123)](https://certificates.software.geant.org/details/test123)

## Licence

This project is licenced under the Apache-2.0 Licence - see the LICENCE file for details.

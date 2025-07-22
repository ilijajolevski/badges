# Product Requirements Document (PRD) for Badge Service

## 1. Overview
The badge service, hosted at `certificates.software.geant.org`, serves pre-issued badges and certificates as SVG images, identified by a unique short Git commit ID (e.g., `abc123`). The service verifies the validity of each ID against a database and returns SVGs for embedding in HTML via `<img>` or `<object>` tags. A new feature adds a details page, accessible via `/details/<commit_id>`, to display comprehensive information about each badge or certificate when clicked.

## 2. Goals and Objectives
- Serve SVG images for pre-issued badges and certificates, ensuring only valid items are displayed.
- Provide a details page for each badge or certificate, showing metadata like issuer, issuance date, and software details.
- Maintain a simple URL structure similar to [Shields.io](https://shields.io/) or [Badgen.net](https://badgen.net/).
- Ensure badges and certificates are clickable, linking to a details page.
- Allow effortless customization of badge styles.
- Support generation of JPG or PNG images in addition to SVG.

## 3. Functional Requirements

### 3.1 Existing Endpoints (Recap)
- **Small Badge Endpoint**: `certificates.software.geant.org/badge/<commit_id>`
  - Returns an SVG for a small badge (e.g., `certificates.software.geant.org/badge/abc123`).
  - Used in `<img>` tags.
- **Large Certificate Endpoint**: `certificates.software.geant.org/certificate/<commit_id>`
  - Returns an SVG for a large certificate (e.g., `certificates.software.geant.org/certificate/abc123`).
  - Used in `<object>` tags.
- **Behavior**:
  - Verify `<commit_id>` in the database.
  - Return SVG with `Content-Type: image/svg+xml` if valid.
  - Return 404 or error SVG if invalid.

#### 3.1.1 Small Badge Design
The small badges served by the `/badge/<commit_id>` endpoint follow a specific design to ensure consistency and readability:

- **Shape and Size**:
  - Rectangular with rounded corners (radius: 3-5 pixels).
  - Dimensions: Approximately 80-200 pixels wide and 20 pixels tall. The width is flexible to accommodate data, with a maximum of 200 pixels. If the label or value exceeds this, abbreviate with "..." or truncate.
- **Layout**:
  - Divided into two sections: a left label and a right value.
  - **Left Section**: Displays the badge category (e.g., "version", "status") or an optional logo image.
  - **Right Section**: Displays the corresponding value (e.g., "v1.3.1", "valid", "platinum badge").
  - No explicit divider; sections are distinguished by background color.
- **Colors**:
  - **Left Section Background**: Configurable (default: Dark gray, e.g., `#333` or RGB(51, 51, 51)).
  - **Right Section Background**: Configurable (default: Color-coded based on the value):
    - Blue gradient (e.g., `#4B6CB7` to `#182848`) for version numbers (e.g., "v1.3.1").
    - Green (e.g., `#4CAF50`) for positive statuses (e.g., "valid", "100%").
    - Light purple (e.g., `#D7BDE2`) for neutral/unknown statuses.
    - Orange (e.g., `#FF9800`) for availability indicators (e.g., "available").
    - Light cyan (e.g., `#B2EBF2`) for style-related badges.
  - **Text Color**: 
    - Default: Contrasted on the background for readability (e.g., White `#FFFFFF` on dark backgrounds)
    - Customizable: Can specify a single text color for both sections or different text colors for left and right sections separately
- **Text**:
  - **Font**: Sans-serif (e.g., Arial or web-safe equivalent).
  - **Font Size**: 10-12 pixels.
  - **Alignment**: Centered within each section.
  - **Content**: Left section shows the category; right section shows the value.
- **Additional Styling**:
  - Subtle shadow or border for a 3D effect.
- **Format**: The badge is an SVG to ensure scalability and compatibility with HTML `<img>` tags.
- **Style Customization**:
  - **Query Parameters**: Allow effortless customization via URL query parameters:
    - `color_left=<hex>`: Sets the left section background color (e.g., `?color_left=#FF0000`).
    - `color_right=<hex>`: Sets the right section background color (e.g., `?color_right=#00FF00`).
    - `text_color=<hex>`: Sets the text color for both sections (e.g., `?text_color=#000000`).
    - `text_color_left=<hex>`: Sets the text color for the left section only (e.g., `?text_color_left=#EEEEEE`).
    - `text_color_right=<hex>`: Sets the text color for the right section only (e.g., `?text_color_right=#FFFFFF`).
    - `logo=<url>`: Replaces the left section text with an image from the specified URL (e.g., `?logo=https://example.com/logo.png`).
    - `font_size=<px>`: Adjusts text size (e.g., `?font_size=14`).
    - `style=<flat|3d>`: Switches between flat or 3D styling (default: 3D).
  - **Configuration File**: Optionally, support a JSON configuration file per `<commit_id>` (e.g., stored in the database) to define default styles (e.g., `{ "color_left": "#333", "color_right": "#4CAF50", "text_color": "#FFFFFF", "text_color_left": "#EEEEEE", "text_color_right": "#FFFFFF" }`).
  - **Default Fallback**: If no parameters or config are provided, use the default design described above.
  - **Validation**: Ensure colors are valid hex codes and sizes are within reasonable limits (e.g., font size 8-16px).

This design ensures the badges are visually consistent, readable, and highly customizable, aligning with industry standards (e.g., [Shields.io](https://shields.io/)).

### 3.2 New Endpoint: Details Page
- **Endpoint**: `certificates.software.geant.org/details/<commit_id>`
  - Returns an HTML page with details about the badge or certificate.
  - Example: `certificates.software.geant.org/details/abc123`
- **Purpose**: Destination for badge/certificate clicks, providing metadata.
- **Content Requirements**:
  - Issuer (e.g., "FINKI Certification Board")
  - Issuance Date (e.g., "2025-05-01")
  - Software Name and Version (e.g., "MyApp v1.3.1")
  - Notes (e.g., "Certified for security compliance")
  - Type (Badge or Certificate)
  - Status (e.g., "Valid")
  - Optional: Expiry date, issuer website link, embedded SVG.
- **Behavior**:
  - Query database for `<commit_id>`.
  - Render HTML page if valid; return 404 with error HTML if invalid.
- **Content-Type**: `text/html`.

### 3.3 Integration with Badge/Certificate Images
- Badges/certificates link to the details page:
  ```html
  <a href="https://certificates.software.geant.org/details/abc123">
    <img src="https://certificates.software.geant.org/badge/abc123" alt="Certification Badge">
  </a>
  ```

### 3.4 Image Format Support
- **Option 1: New Endpoint**
  - **Endpoint**: `certificates.software.geant.org/badge/<commit_id>/image`
  - Returns a JPG or PNG image based on a query parameter.
  - Example: `certificates.software.geant.org/badge/abc123/image?format=png`
  - Supported formats: `format=jpg` or `format=png` (default: SVG if no format specified).
  - Behavior: Convert the SVG to the requested format using an image processing library (e.g., ImageMagick or Go’s `image` package).
  - Content-Type: `image/jpeg` for JPG, `image/png` for PNG.
- **Option 2: Existing Endpoint with Parameter**
  - Enhance `/badge/<commit_id>` to support a `format` query parameter.
  - Example: `certificates.software.geant.org/badge/abc123?format=png`
  - Supported formats: `format=svg` (default), `format=jpg`, `format=png`.
  - Behavior: Return SVG, JPG, or PNG based on the parameter.
  - Content-Type: Adjusts to `image/svg+xml`, `image/jpeg`, or `image/png` accordingly.
- **Recommendation**: Use Option 2 for simplicity, allowing a single endpoint to handle all formats.
- **Behavior for All Formats**:
  - Verify `<commit_id>` in the database.
  - Return the requested image format if valid; return 404 or error image if invalid.

## 4. Non-Functional Requirements
- **Performance**: Respond within 500ms for SVGs/JPG/PNG, 1s for details page.
- **Scalability**: Handle 1,000 concurrent requests with caching.
- **Security**:
  - Sanitize `<commit_id>` and query parameters.
  - Implement rate limiting.
- **Compatibility**: Details page must be responsive and work in modern browsers.
- **Accessibility**: Follow basic accessibility guidelines (e.g., alt text, semantic HTML).

## 5. Database Schema

### 5.1 Badge Schema
| Field             | Type   | Description                     | Example                     |
|-------------------|--------|---------------------------------|-----------------------------|
| `commit_id`       | String | Unique ID                      | `abc123`                   |
| `type`            | String | "badge" or "certificate"       | `badge`                    |
| `status`          | String | "valid", "expired", "revoked"  | `valid`                    |
| `issuer`          | String | Certifying authority           | `FINKI Certification Board`|
| `issue_date`      | Date   | Issuance date                  | `2025-05-01`               |
| `software_name`   | String | Software name                  | `MyApp`                    |
| `software_version`| String | Software version               | `v1.3.1`                   |
| `software_url`    | String | Software website URL (optional)| `https://myapp.com`        |
| `notes`           | String | Additional notes               | `Certified for security`   |
| `svg_content`     | String | Pre-generated SVG (optional)   | `<svg>...</svg>`           |
| `expiry_date`     | Date   | Expiry date (optional)         | `2026-05-01`               |
| `issuer_url`      | String | Issuer's website (optional)    | `https://finki.edu.mk`     |
| `custom_config`   | JSON   | Customization settings (e.g., colors, logo URL) | `{"color_left": "#333", "color_right": "#4CAF50", "text_color": "#FFFFFF", "text_color_left": "#EEEEEE", "text_color_right": "#FFFFFF"}` |
| `last_review`     | Date   | Last review date (optional)    | `2025-06-15`               |
| `jpg_content`     | Blob   | Pre-generated JPG (optional)   | Binary JPG data            |
| `png_content`     | Blob   | Pre-generated PNG (optional)   | Binary PNG data            |
| `covered_version` | String | Semantic versioning or git tag (optional) | `1.2.3` or `v2.0.1` or `release-tag` |
| `repository_link` | String | Code repository URL (optional) | `https://github.com/org/repo` |
| `public_note`     | String | Long text note for public display (optional) | `This certificate verifies compliance with security standards...` |
| `internal_note`   | String | Long text note for internal use only (optional) | `Internal review comments and notes...` |
| `contact_details` | String | Contact information for public display (optional) | `support@example.com, +1-123-456-7890` |
| `certificate_name` | String | Name of the certificate (optional) | `Self-Assessed Dependencies` |
| `specialty_domain` | String | Specialty domain of the certificate (optional) | `SOFTWARE LICENCING` |
| `software_sc_id`  | String | Software Catalogue Project ID (optional) | `project-123` |
| `software_sc_url` | String | Software Catalogue Link (optional) | `https://sc.geant.org/ui/project/project-123` |

- **Notes on Badge Fields**:
  - `custom_config`: Stores JSON with default customization options for each badge, including:
    - Colors for left and right sections (`color_left`, `color_right`)
    - Text colors, with options for both sections (`text_color`) or individual sections (`text_color_left`, `text_color_right`)
    - Logo URL, font size, and style options
  - `last_review`: Stores the date when the badge was last reviewed or verified, useful for tracking badge maintenance and validity checks.
  - `jpg_content` and `png_content`: Store pre-generated images to reduce conversion overhead, populated on demand or during issuance.

### 5.2 User Authentication Schema

#### 5.2.1 Users Table
| Field             | Type     | Description                     | Example                     |
|-------------------|----------|---------------------------------|-----------------------------|
| `user_id`         | UUID     | Unique user identifier          | `550e8400-e29b-41d4-a716-446655440000` |
| `username`        | String   | Unique username                 | `admin_user`                |
| `email`           | String   | User email address              | `admin@example.com`         |
| `password_hash`   | String   | Bcrypt hashed password          | `$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy` |
| `first_name`      | String   | User's first name               | `John`                      |
| `last_name`       | String   | User's last name                | `Doe`                       |
| `role_id`         | UUID     | Reference to role               | `123e4567-e89b-12d3-a456-426614174000` |
| `created_at`      | DateTime | Account creation timestamp      | `2025-07-01T10:30:00Z`      |
| `updated_at`      | DateTime | Last update timestamp           | `2025-07-15T14:45:00Z`      |
| `last_login`      | DateTime | Last login timestamp            | `2025-07-22T09:15:00Z`      |
| `status`          | String   | Account status                  | `active`                    |
| `failed_attempts` | Integer  | Failed login attempts           | `0`                         |

#### 5.2.2 Roles Table
| Field             | Type     | Description                     | Example                     |
|-------------------|----------|---------------------------------|-----------------------------|
| `role_id`         | UUID     | Unique role identifier          | `123e4567-e89b-12d3-a456-426614174000` |
| `name`            | String   | Role name                       | `admin`                     |
| `description`     | String   | Role description                | `Administrator with full access` |
| `permissions`     | JSON     | Permission set                  | `{"badges": {"read": true, "write": true, "delete": true}}` |
| `created_at`      | DateTime | Creation timestamp              | `2025-07-01T10:30:00Z`      |
| `updated_at`      | DateTime | Last update timestamp           | `2025-07-15T14:45:00Z`      |

#### 5.2.3 API Keys Table
| Field             | Type     | Description                     | Example                     |
|-------------------|----------|---------------------------------|-----------------------------|
| `api_key_id`      | UUID     | Unique API key identifier       | `98765432-abcd-efgh-ijkl-123456789012` |
| `user_id`         | UUID     | Reference to user               | `550e8400-e29b-41d4-a716-446655440000` |
| `api_key`         | String   | Hashed API key                  | `2c17c6393771ee3048ae34d6b380c5ec` |
| `name`            | String   | Key name/description            | `Service Integration Key`   |
| `permissions`     | JSON     | Specific permissions for key    | `{"badges": {"read": true, "write": false}}` |
| `created_at`      | DateTime | Creation timestamp              | `2025-07-01T10:30:00Z`      |
| `expires_at`      | DateTime | Expiration timestamp            | `2026-07-01T10:30:00Z`      |
| `last_used`       | DateTime | Last usage timestamp            | `2025-07-22T09:15:00Z`      |
| `status`          | String   | Key status                      | `active`                    |
| `ip_restrictions` | JSON     | Optional IP address restrictions| `["192.168.1.0/24", "10.0.0.1"]` |

## 6. API Specification

### 6.1 Authentication

All API endpoints that require authentication support two authentication methods:

1. **JWT Token Authentication**:
   ```
   Authorization: Bearer <jwt_token>
   ```

2. **API Key Authentication** (for read-only operations):
   ```
   X-API-Key: <api_key>
   ```

#### 6.1.1 JWT Token Structure

JWT tokens include the following claims:

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "username": "admin_user",
  "email": "admin@example.com",
  "role": "admin",
  "permissions": {
    "badges": {
      "read": true,
      "write": true,
      "delete": true
    },
    "users": {
      "read": true,
      "write": true,
      "delete": true
    },
    "api_keys": {
      "read": true,
      "write": true,
      "delete": true
    }
  },
  "iat": 1690048380,
  "exp": 1690051980,
  "iss": "certificates.software.geant.org"
}
```

**Note**: The JWT token claims include:
- `sub`: User ID
- `iat`: Issued at timestamp
- `exp`: Expiration timestamp (1 hour later)
- `iss`: Issuer
```

#### 6.1.2 Authentication Endpoints

| Endpoint                  | Method | Description               | Authentication | Response         |
|---------------------------|--------|---------------------------|----------------|------------------|
| `/api/v1/auth/login`      | POST   | Authenticate user and get JWT token | None | JSON with token and user info |
| `/api/v1/auth/refresh`    | POST   | Refresh JWT token         | JWT Token      | JSON with new token |
| `/api/v1/auth/logout`     | POST   | Invalidate JWT token      | JWT Token      | JSON with success message |

**Request Body for `/api/v1/auth/login`**:
```json
{
  "username": "admin_user",
  "password": "secure_password"
}
```

**Response for `/api/v1/auth/login`** (200 OK):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "admin_user",
    "email": "admin@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "admin"
  },
  "expires_at": "2025-07-22T19:53:00Z"
}
```

### 6.2 Public Badge Endpoints

These endpoints are publicly accessible and do not require authentication.

| Endpoint                  | Method | Description               | Authentication | Response         |
|---------------------------|--------|---------------------------|----------------|------------------|
| `/badge/<commit_id>`      | GET    | Retrieve small badge SVG, JPG, or PNG | None | SVG (`image/svg+xml`), JPG (`image/jpeg`), or PNG (`image/png`) |
| `/certificate/<commit_id>`| GET    | Retrieve certificate SVG, JPG, or PNG | None | SVG (`image/svg+xml`), JPG (`image/jpeg`), or PNG (`image/png`) |
| `/details/<commit_id>`    | GET    | Retrieve details page    | None | HTML (`text/html`) |
| `/badges`                 | GET    | Retrieve badges list page | None | HTML (`text/html`) |

**Query Parameters for `/badge/<commit_id>`**:
- `format=svg|jpg|png`: Specifies the image format (default: `svg`).
- `color_left=<hex>`: Custom left section color.
- `color_right=<hex>`: Custom right section color.
- `text_color=<hex>`: Custom text color for both sections (overridden by section-specific colors if provided).
- `text_color_left=<hex>`: Custom text color for the left section only.
- `text_color_right=<hex>`: Custom text color for the right section only.
- `logo=<url>`: URL of a logo image for the left section.
- `font_size=<px>`: Custom font size.
- `style=<flat|3d>`: Badge style.
- `no_cache=true`: Bypasses the cache and generates a fresh badge. Useful for immediately seeing style changes during development.

**Query Parameters for `/certificate/<commit_id>`**:
- `format=svg|jpg|png`: Specifies the image format (default: `svg`).
- `color_left=<hex>`: Custom left section color.
- `color_right=<hex>`: Custom right section color.
- `text_color=<hex>`: Custom text color for both sections (overridden by section-specific colors if provided).
- `text_color_left=<hex>`: Custom text color for the left section only.
- `text_color_right=<hex>`: Custom text color for the right section only.
- `logo=<url>`: URL of a logo image for the left section.
- `font_size=<px>`: Custom font size.
- `style=<flat|3d>`: Badge style.
- `no_cache=true`: Bypasses the cache and generates a fresh certificate. Useful for immediately seeing style changes during development.

### 6.3 Integration API Endpoints (Read-Only)

These endpoints allow other services to retrieve badge data in JSON format. They require authentication with either JWT token or API key.

| Endpoint                  | Method | Description               | Authentication | Response         |
|---------------------------|--------|---------------------------|----------------|------------------|
| `/api/v1/badges`          | GET    | List all badges (paginated) | JWT Token or API Key | JSON |
| `/api/v1/badges/<commit_id>` | GET | Get badge details        | JWT Token or API Key | JSON |

**Query Parameters for `/api/v1/badges`**:
- `page` (optional): Page number (default: 1)
- `limit` (optional): Items per page (default: 20, max: 100)
- `status` (optional): Filter by status ("valid", "expired", "revoked")
- `issuer` (optional): Filter by issuer name
- `software_name` (optional): Filter by software name
- `sort` (optional): Sort field (default: "issue_date")
- `order` (optional): Sort order ("asc" or "desc", default: "desc")

**Response for `/api/v1/badges`** (200 OK):
```json
{
  "total": 125,
  "page": 1,
  "limit": 20,
  "badges": [
    {
      "commit_id": "abc123",
      "status": "valid",
      "issuer": "FINKI Certification Board",
      "issue_date": "2025-05-01",
      "software_name": "MyApp",
      "software_version": "v1.3.1",
      "certificate_name": "Self-Assessed Dependencies",
      "specialty_domain": "SOFTWARE LICENCING",
      "expiry_date": "2026-05-01"
    }
  ]
}
```

**Note**: The actual response will include multiple badge entries in the array.
```

**Response for `/api/v1/badges/<commit_id>`** (200 OK):
```json
{
  "commit_id": "abc123",
  "status": "valid",
  "issuer": "FINKI Certification Board",
  "issue_date": "2025-05-01",
  "software_name": "MyApp",
  "software_version": "v1.3.1",
  "software_url": "https://myapp.com",
  "notes": "Certified for security compliance",
  "expiry_date": "2026-05-01",
  "issuer_url": "https://finki.edu.mk",
  "last_review": "2025-06-15",
  "covered_version": "1.2.3",
  "repository_link": "https://github.com/org/repo",
  "public_note": "This certificate verifies compliance with security standards...",
  "contact_details": "support@example.com, +1-123-456-7890",
  "certificate_name": "Self-Assessed Dependencies",
  "specialty_domain": "SOFTWARE LICENCING",
  "software_sc_id": "project-123",
  "software_sc_url": "https://sc.geant.org/ui/project/project-123",
  "badge_url": "https://certificates.software.geant.org/badge/abc123",
  "certificate_url": "https://certificates.software.geant.org/certificate/abc123",
  "details_url": "https://certificates.software.geant.org/details/abc123"
}
```

### 6.4 Admin API Endpoints (CRUD Operations)

These endpoints allow administrators to manage badges through a web admin interface. They require authentication with JWT token and appropriate permissions.

#### 6.4.1 Badge Management Endpoints

| Endpoint                  | Method | Description               | Authentication | Response         |
|---------------------------|--------|---------------------------|----------------|------------------|
| `/api/v1/admin/badges`    | POST   | Create a new badge        | JWT Token (admin/editor) | JSON |
| `/api/v1/admin/badges/<commit_id>` | PUT | Update an existing badge | JWT Token (admin/editor) | JSON |
| `/api/v1/admin/badges/<commit_id>` | DELETE | Delete a badge   | JWT Token (admin) | JSON |
| `/api/v1/admin/badges/batch` | POST | Perform batch operations | JWT Token (admin) | JSON |
| `/api/v1/admin/badges/<commit_id>/svg` | POST | Upload custom SVG | JWT Token (admin/editor) | JSON |

**Request Body for `/api/v1/admin/badges`**:
```json
{
  "commit_id": "abc123",
  "status": "valid",
  "issuer": "FINKI Certification Board",
  "issue_date": "2025-05-01",
  "software_name": "MyApp",
  "software_version": "v1.3.1",
  "software_url": "https://myapp.com",
  "notes": "Certified for security compliance",
  "expiry_date": "2026-05-01",
  "issuer_url": "https://finki.edu.mk",
  "covered_version": "1.2.3",
  "repository_link": "https://github.com/org/repo",
  "public_note": "This certificate verifies compliance with security standards...",
  "internal_note": "Internal review comments and notes...",
  "contact_details": "support@example.com, +1-123-456-7890",
  "certificate_name": "Self-Assessed Dependencies",
  "specialty_domain": "SOFTWARE LICENCING",
  "software_sc_id": "project-123",
  "software_sc_url": "https://sc.geant.org/ui/project/project-123",
  "custom_config": {
    "color_left": "#333",
    "color_right": "#4CAF50",
    "text_color": "#FFFFFF",
    "text_color_left": "#EEEEEE",
    "text_color_right": "#FFFFFF",
    "logo": "https://example.com/logo.png",
    "font_size": 12,
    "style": "3d"
  }
}
```

**Response for `/api/v1/admin/badges`** (201 Created):
```json
{
  "commit_id": "abc123",
  "message": "Badge created successfully",
  "badge_url": "https://certificates.software.geant.org/badge/abc123",
  "certificate_url": "https://certificates.software.geant.org/certificate/abc123",
  "details_url": "https://certificates.software.geant.org/details/abc123"
}
```

#### 6.4.2 User Management Endpoints

| Endpoint                  | Method | Description               | Authentication | Response         |
|---------------------------|--------|---------------------------|----------------|------------------|
| `/api/v1/admin/users`     | POST   | Create a new user         | JWT Token (admin) | JSON |
| `/api/v1/admin/users`     | GET    | List all users (paginated) | JWT Token (admin) | JSON |
| `/api/v1/admin/users/<user_id>` | GET | Get user details       | JWT Token (admin) | JSON |
| `/api/v1/admin/users/<user_id>` | PUT | Update an existing user | JWT Token (admin) | JSON |
| `/api/v1/admin/users/<user_id>` | DELETE | Delete a user       | JWT Token (admin) | JSON |
| `/api/v1/admin/users/<user_id>/password` | PUT | Change user password | JWT Token (admin or self) | JSON |

**Request Body for `/api/v1/admin/users`**:
```json
{
  "username": "new_user",
  "email": "user@example.com",
  "password": "secure_password",
  "first_name": "Jane",
  "last_name": "Smith",
  "role_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

**Response for `/api/v1/admin/users`** (201 Created):
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "new_user",
  "email": "user@example.com",
  "first_name": "Jane",
  "last_name": "Smith",
  "role": "editor",
  "created_at": "2025-07-22T18:53:00Z"
}
```

#### 6.4.3 API Key Management Endpoints

| Endpoint                  | Method | Description               | Authentication | Response         |
|---------------------------|--------|---------------------------|----------------|------------------|
| `/api/v1/admin/api-keys`  | POST   | Create a new API key      | JWT Token | JSON |
| `/api/v1/admin/api-keys`  | GET    | List all API keys         | JWT Token | JSON |
| `/api/v1/admin/api-keys/<api_key_id>` | GET | Get API key details | JWT Token | JSON |
| `/api/v1/admin/api-keys/<api_key_id>` | PUT | Update an API key | JWT Token | JSON |
| `/api/v1/admin/api-keys/<api_key_id>` | DELETE | Delete an API key | JWT Token | JSON |
| `/api/v1/admin/api-keys/<api_key_id>/revoke` | POST | Revoke an API key | JWT Token | JSON |

**Request Body for `/api/v1/admin/api-keys`**:
```json
{
  "name": "Service Integration Key",
  "permissions": {
    "badges": {
      "read": true,
      "write": false
    }
  },
  "expires_at": "2026-07-22T18:53:00Z",
  "ip_restrictions": ["192.168.1.0/24", "10.0.0.1"]
}
```

**Response for `/api/v1/admin/api-keys`** (201 Created):
```json
{
  "api_key_id": "98765432-abcd-efgh-ijkl-123456789012",
  "api_key": "...",
  "name": "Service Integration Key",
  "permissions": {
    "badges": {
      "read": true,
      "write": false
    }
  },
  "created_at": "2025-07-22T18:53:00Z",
  "expires_at": "2026-07-22T18:53:00Z",
  "status": "active"
}
```

**Note**: The `api_key` value is only shown once when the key is created. It cannot be retrieved later.
```

### 6.5 Error Handling

All API endpoints follow a consistent error response format:

```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Badge with commit_id 'abc123' not found",
    "details": {
      "commit_id": "abc123"
    }
  }
}
```

Common error codes:
- `UNAUTHORIZED`: Missing or invalid authentication
- `FORBIDDEN`: Insufficient permissions
- `RESOURCE_NOT_FOUND`: Requested resource not found
- `VALIDATION_ERROR`: Invalid request data
- `INTERNAL_ERROR`: Server-side error

For public endpoints:
- Return 404 for invalid `<commit_id>`.
- For `/badge` and `/certificate`, return error image in the requested format.
- For `/details`, return error HTML page.

### 6.6 Rate Limiting

API endpoints implement rate limiting to prevent abuse:

- Public endpoints: 200 requests per minute per IP
- Integration API: 100 requests per minute per token/key
- Admin API: 60 requests per minute per token

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1627834096
```

## 7. User Interface

### 7.1 Details Page
- **Header**: Title (e.g., "Badge/Certificate Details: abc123"), issuer logo.
- **Main Content**:
  - Type, Status, Issuer, Issuance Date, Software Name and Version, Notes, Expiry Date, Covered Version, Repository Link, Public Note, Contact Details, Embedded SVG (optional).
- **Footer**: Link to `certificates.software.geant.org`, copyright notice.

### 7.2 Badges List Page
- **Header**: Title ("Badge List"), issuer logo.
- **Main Content**:
  - Table with columns for Software Name, Status (valid/expired), and Issue Date.
  - Each row links to the corresponding badge details page.
  - Responsive design for mobile devices.
- **Footer**: Link to `certificates.software.geant.org`, copyright notice.

## 8. Implementation Notes

### 8.1 Core Technology Stack
- Use Go (Golang) with the standard `net/http` library.
- Use SQLite as the database, stored in the `/db` folder.
- Use the `zap` library for logging.
- Use a template engine (e.g., `html/template` package) for `/details` HTML.
- Cache SVGs, JPGs, and HTML pages (e.g., using an in-memory cache like `sync.Map` or a library like `gocache`).
- Sanitize inputs, use HTTPS.

### 8.2 Authentication and Authorization
- **JWT Token Management**:
  - Use a library like `github.com/golang-jwt/jwt` for JWT token generation and validation.
  - Store JWT secret in environment variables or a secure configuration.
  - Implement token refresh mechanism to extend sessions without requiring re-login.
  - Include comprehensive claims in tokens (user ID, username, role, permissions).
  - Set token expiration to 1 hour for security.

- **Password Security**:
  - Use bcrypt with a work factor of 12+ for password hashing.
  - Implement password validation rules (minimum length, complexity requirements).
  - Store only password hashes, never plaintext passwords.
  - Implement account lockout after multiple failed login attempts.

- **API Key Management**:
  - Generate API keys with sufficient entropy (at least 32 bytes).
  - Store only hashed API keys in the database.
  - Implement IP restriction validation for API keys.
  - Log API key usage for audit purposes.

- **Authentication Middleware**:
  - Create middleware for JWT token validation.
  - Create middleware for API key validation.
  - Implement role-based access control (RBAC) for endpoints.
  - Add rate limiting to prevent abuse.

### 8.3 Database and Data Management
- **Database Schema**:
  - Implement the schema as defined in Section 5.
  - Create appropriate indexes for performance optimization.
  - Implement database migrations for schema updates.

- **Data Validation**:
  - Validate all input data before storing in the database.
  - Implement comprehensive error handling for database operations.

### 8.4 Logging and Monitoring
- **Logging**: 
  - Log all available badges at server startup for monitoring and debugging purposes.
  - Each badge's commit ID, type, status, and software information is logged.
  - Log authentication events (login, logout, failed attempts).
  - Log API key creation, usage, and revocation.
  - Use structured logging for easier parsing and analysis.

- **Monitoring**:
  - Implement health check endpoints.
  - Add metrics for API usage and performance.
  - Monitor authentication failures and rate limit triggers.

### 8.5 Deployment
- **Dockerfile**: Implement a multi-stage build to optimize the image size:
  - Stage 1: Build the Go binary.
  - Stage 2: Copy the binary and SQLite DB into a lightweight image (e.g., `alpine`).

- **Makefile**: Include targets:
  - `run`: Start the service locally.
  - `build`: Compile the Go binary.
  - `build-image`: Build the Docker image.
  - `push-image`: Push the Docker image to a registry.
  - `migrate`: Run database migrations.
  - `test`: Run unit and integration tests.

### 8.6 Image Processing
- **Image Conversion**: Use a library like `github.com/disintegration/imaging` to convert SVG to JPG/PNG on demand if not pre-generated.
- **SVG Generation**: Use templates for generating SVG badges and certificates.

### 8.7 Security Considerations
- Implement HTTPS for all endpoints.
- Set secure HTTP headers (HSTS, Content-Security-Policy, etc.).
- Sanitize all user inputs to prevent injection attacks.
- Implement rate limiting to prevent brute force and DoS attacks.
- Regularly rotate JWT secrets and API keys.
- Implement proper error handling to avoid information leakage.

## 9. Success Criteria
- Badges/certificates display correctly and link to details page.
- Details page shows all required metadata.
- Service handles invalid requests gracefully.
- Service shuts down gracefully (e.g., handle SIGTERM/SIGINT).
- Customizations apply correctly via query parameters or config.
- JPG/PNG images are generated and served alongside SVG.

## 10. Future Enhancements
- Provide JSON API for metadata.
- Allow badge style customization via a web UI.
- Support certificate image formats (JPG/PNG).

## 11. Implementation Task List

### Project Structure
- [x] Set up Go project structure with proper module organization
- [x] Create `/db` directory for SQLite database
- [x] Set up configuration management for environment variables
- [x] Create HTML templates directory for the details page
- [x] Set up static assets directory for CSS, JS, and images
- [x] Configure logging with zap library

### Database Implementation
- [x] Design and implement SQLite database schema for badges as per section 5.1
- [x] Create database initialization and migration scripts
- [x] Implement database connection and query functions
- [x] Create CRUD operations for badge/certificate data
- [x] Implement data validation for database operations
- [ ] Extend database schema to include authentication tables as per section 5.2
- [ ] Create migration scripts for authentication tables
- [ ] Implement CRUD operations for users, roles, and API keys

### Authentication and Authorization
- [ ] Implement JWT token generation and validation
- [ ] Create authentication middleware for JWT tokens
- [ ] Implement API key generation and validation
- [ ] Create authentication middleware for API keys
- [ ] Implement role-based access control (RBAC)
- [ ] Create password hashing and validation functions
- [ ] Implement account lockout mechanism for failed login attempts
- [ ] Add IP restriction validation for API keys
- [ ] Implement rate limiting for authentication endpoints

### User Management
- [ ] Create user registration and login functionality
- [ ] Implement user profile management
- [ ] Create role management functionality
- [ ] Implement password reset mechanism
- [ ] Create user listing and search functionality
- [ ] Implement user account deactivation/reactivation

### API Key Management
- [ ] Implement API key generation with sufficient entropy
- [ ] Create API key listing and management functionality
- [ ] Implement API key revocation
- [ ] Add API key usage logging for audit purposes
- [ ] Create API key permission management

### Core Badge Service
- [x] Implement badge generation logic for SVG format
- [x] Implement certificate generation logic for SVG format
- [x] Create utility functions for customizing badge appearance
- [x] Implement SVG to JPG/PNG conversion using imaging library
- [x] Create caching mechanism for generated images

### API Endpoints
- [x] Implement `/badge/<commit_id>` endpoint with format parameter support
- [x] Implement `/certificate/<commit_id>` endpoint
- [x] Implement `/details/<commit_id>` endpoint
- [x] Implement `/badges` endpoint for listing all badges
- [x] Create middleware for input sanitization and validation
- [x] Implement rate limiting middleware
- [x] Create error handling for all endpoints
- [ ] Implement authentication endpoints as per section 6.1.2
- [ ] Create integration API endpoints as per section 6.3
- [ ] Implement admin API endpoints as per section 6.4

### User Interface
- [x] Design and implement HTML template for details page
- [x] Design and implement HTML template for badges list page
- [x] Create CSS styles for responsive design
- [x] Implement error page templates
- [x] Ensure accessibility compliance
- [ ] Create login and user management interface
- [ ] Implement API key management interface
- [ ] Design and implement admin dashboard
- [ ] Create forms for badge creation and management

### Testing
- [x] Write unit tests for core badge generation functions
- [x] Write integration tests for database operations
- [x] Create API endpoint tests
- [ ] Implement performance testing for response time requirements
- [ ] Test image format conversion
- [ ] Write unit tests for authentication functions
- [ ] Create integration tests for authentication endpoints
- [ ] Test user management functionality
- [ ] Implement security testing for authentication mechanisms

### Deployment
- [x] Create Dockerfile with multi-stage build
- [x] Implement Makefile with required targets
- [ ] Set up CI/CD pipeline configuration
- [ ] Create deployment documentation
- [x] Implement graceful shutdown handling
- [ ] Configure secure environment for JWT secrets and API keys
- [ ] Set up monitoring for authentication failures and rate limit triggers

### Documentation
- [x] Create API documentation
- [x] Write user guide for badge/certificate integration
- [x] Document database schema and operations
- [x] Create developer onboarding documentation
- [ ] Write authentication and authorization documentation
- [ ] Create user management documentation
- [ ] Document API key management
- [ ] Write security best practices guide


## 12. Implementation Phases

### Phase 1: Core Authentication Infrastructure

**Objective**: Establish the foundational authentication infrastructure.

**Tasks**:
- Extend database schema to include users, roles, and API keys tables
- Create database migration scripts if needed
- Implement JWT token generation and validation
- Create password hashing and validation functions
- Implement basic authentication middleware

**Deliverables**:
- Updated database schema with authentication tables
- Working JWT token generation and validation
- Functional password security system
- Basic authentication middleware

### Phase 2: User Management System

**Objective**: Implement user management functionality.

**Tasks**:
- Create user registration and login endpoints
- Implement user profile management
- Develop role management functionality
- Create user listing and search functionality
- Implement account lockout mechanism

**Deliverables**:
- Functional user registration and login system
- User profile management interface
- Role management system
- User administration interface

### Phase 3: API Key Management

**Objective**: Implement API key management system.

**Tasks**:
- Develop API key generation with sufficient entropy
- Create API key listing and management functionality
- Implement API key revocation
- Add IP restriction validation for API keys
- Implement API key usage logging

**Deliverables**:
- API key generation and validation system
- API key management interface
- API key security features (IP restrictions, logging)

### Phase 4: Integration and Admin APIs

**Objective**: Implement integration and admin API endpoints.

**Tasks**:
- Create integration API endpoints for badge listing and details
- Implement admin API endpoints for badge management
- Develop batch operations functionality
- Implement rate limiting for API endpoints
- Create comprehensive error handling

**Deliverables**:
- Functional integration API endpoints
- Admin API endpoints for badge management
- API documentation
- Rate limiting and security features

### Phase 5: Admin Interface

**Objective**: Develop the web admin interface.

**Tasks**:
- Design and implement admin dashboard
- Create forms for badge creation and management
- Implement API key management interface
- Develop user management interface
- Create comprehensive error handling and feedback

**Deliverables**:
- Functional admin dashboard
- Badge management interface
- User and API key management interfaces
- Comprehensive error handling and user feedback

### Phase 6: Testing and Security Hardening

**Objective**: Ensure the system is thoroughly tested and secure.

**Tasks**:
- Write unit tests for authentication functions
- Create integration tests for all endpoints
- Implement security testing for authentication mechanisms
- Perform performance testing
- Conduct security audit and penetration testing

**Deliverables**:
- Comprehensive test suite
- Security audit report
- Performance testing results
- Documentation of security measures



## 13. Unified Badge Entity and Outlook Separation

### Overview

To simplify the data model and enhance flexibility, the distinction between "badge" and "certificate" will be removed at the database/entity level. Instead, the **badge entity** will be unified, and the graphical *outlook* (either "badge" or "certificate" style) will be treated as a rendering/view parameter, not a fundamental property or separate record.

**Key changes:**
- The `type` field in the database will no longer determine the existence or nature of a badge record.
- Both the “small badge” and the “large certificate” are graphical presentations of the same badge entity, configurable at render time.
- The *outlook* is decided by the requested endpoint or a query parameter, not by the badge data model.

### Database Model Changes

1. **Badge Table:**
  - Remove or ignore the `type` column as a controlling factor for badge/certificate distinction.
  - Each badge record (one per `commit_id`) represents a single logical badge, regardless of outlook.

2. **Outlook Rendering:**
  - The visual difference between “badge” (small, inline, e.g. SVG/png/jpg) and “certificate” (large, prominent, e.g. SVG/png/jpg) is determined by the endpoint or a rendering parameter, *not* by the entity.
  - If needed, retain the `type` field only for backward compatibility, but it has no impact on rendering or business logic.

### Certificate SVG Outlook Specification

- The **certificate outlook** must follow the visual and structural guidelines inspired by the provided image (`certificate_look_of_the_badge.png`):
  - **Shape and Layout:** Large horizontal rectangle, designed for prominence, with visually distinct borders (e.g., gold, silver, or institutional color).
  - **Size:** At least 500px wide and 350px high in SVG (scalable for high-DPI displays).
  - **Frame/Border:** Decorative border (rounded corners recommended), with a possible drop shadow or soft edge.
  - **Header:** Top section includes the issuer’s logo (optionally left-aligned), and certificate heading text (e.g., "Certificate of Achievement", "Certification Badge").
  - **Main Body:**
    - Centered large title (the achievement or certification, e.g., "Certified Security Practitioner").
    - Recipient section (if applicable; can be a field for the owner’s name or organization).
    - Meta information (badge name, date, software name/version, status, unique ID).
    - Optional: QR code or short verification URL at the bottom right.
  - **Color Palette:** Neutral, academic, or institutional—background should be white or a very light color, with high-contrast text.
  - **Fonts:** Use web-safe fonts or Google Fonts (e.g., "Lato", "Montserrat", "Roboto Slab") for headings and content; font size for main title: 28-40px, for metadata: 16-20px.
  - **Graphics:** The SVG can include embedded PNG/JPG for logos, seals, or icons if necessary, but the main structure should be vector-based for scalability.

The certificate template is in the /templates/svg directory,
the SVG should be designed to be easily customizable with CSS or inline styles, allowing for color changes, font adjustments, and logo replacements.
the small badge outlook of the is defined ind the file: /templates/svg/small-template.svg
the big certificate outlook of the is defined ind the file: /templates/svg/big-template.svg
the template has comments where the colors can and should be changed to customize the badge and certificate outlooks.

### API & Endpoint Changes

- `/badge/<commit_id>`: Renders the *badge* outlook (small style, see existing design guidelines).
- `/certificate/<commit_id>`: Renders the *certificate* outlook (as detailed above).
- Both endpoints retrieve the same unified badge entity and select the outlook based solely on the endpoint.
- Add a rendering parameter (optional) to let `/badge/<commit_id>?outlook=certificate` or `/certificate/<commit_id>?outlook=badge` force the alternate style for special use-cases.
- All image formats (`svg`, `jpg`, `png`) are supported for both outlooks as previously described.

### Rendering & Presentation Rules

- The visual styles are defined as follows:
  - **Badge Outlook:** Small, compact, rectangular or pill-shaped, suitable for inline display.
  - **Certificate Outlook:** Larger, border-decorated, visually rich, suitable for formal display or printing.
- Each badge entity must supply the SVG/png/jpg for both outlooks. Store as separate fields, or generate dynamically based on the unified data and templates.

### Storage of Outlook Assets

- Add (if not already present) fields to store or generate both graphical outlooks:
  - `badge_svg_content`, `certificate_svg_content` (or use a JSON map or sub-table for outlooks).
  - `badge_png_content`, `certificate_png_content` (optional, for pre-rendered bitmaps).
- Alternatively, store only a single SVG/template and dynamically adjust size and style at render time based on outlook parameter.

### Details Page Layout

- The details page (`/details/<commit_id>`) must present both graphical outlooks and the badge metadata:
  - **Layout:**
    - **Left Column (vertical stack):**
      1. Small badge outlook (SVG, PNG, or JPG)
      2. Certificate outlook (SVG, PNG, or JPG; larger, as described above)
      3. **Integration Snippets** for both outlooks (see below)
    - **Right Column:**
      - Badge metadata and details (issuer, date, software, status, ID, notes, etc.)
      - Optional: Buttons to copy embed snippets
  - **Integration Snippets:**
    - For **small badge**:
      ```html
      <img src="https://certificates.software.geant.org/badge/abc123" alt="Certification Badge">
      ```
    - For **certificate**:
      ```html
      <img src="https://certificates.software.geant.org/certificate/abc123" alt="Certificate">
      ```
    - Optionally, provide `<object>` examples for SVG:
      ```html
      <object type="image/svg+xml" data="https://certificates.software.geant.org/certificate/abc123"></object>
      ```
  - **Responsiveness:** The layout must remain usable and readable on both desktop and mobile. On narrow screens, stack columns vertically.

### Backwards Compatibility

- The system must continue to honor requests to `/badge/<commit_id>` and `/certificate/<commit_id>`.
- Legacy code referencing the `type` field should be updated to only use it for legacy compatibility; all new logic and UIs must treat “badge” and “certificate” as alternative views of a single badge entity.

### Implementation Steps

1. **Database Migration:**
  - Remove the requirement for multiple entries for different types.
  - Migrate all existing badges/certificates into single badge records per unique `commit_id`.
  - Mark `type` as deprecated, or repurpose it for legacy display only.

2. **API Refactor:**
  - Update badge retrieval logic to ignore type and always load by `commit_id`.
  - Update endpoints to select outlook (badge/certificate) purely by URL or query parameter.

3. **Rendering Layer:**
  - Refactor SVG/image generation to support both outlooks from the same data, following the above specifications.
  - Add support for toggling outlook in the details page.

4. **Details Page Update:**
  - Render both badge and certificate outlooks in the left column, one below the other.
  - Display integration snippets directly below each outlook.
  - Keep metadata/details on the right.

5. **Documentation Update:**
  - Update all API and developer documentation to reflect the new model.

### Example

For a badge with `commit_id=abc123`, the following are all possible and valid (all reference the same underlying badge entity):

- `https://certificates.software.geant.org/badge/abc123` → small badge outlook (SVG/png/jpg)
- `https://certificates.software.geant.org/certificate/abc123` → certificate outlook (SVG/png/jpg)
- `https://certificates.software.geant.org/badge/abc123?format=png` → badge outlook in PNG
- `https://certificates.software.geant.org/certificate/abc123?format=svg` → certificate outlook in SVG
- `https://certificates.software.geant.org/badge/abc123?outlook=certificate` → certificate outlook from the badge endpoint
- `https://certificates.software.geant.org/certificate/abc123?outlook=badge` → badge outlook from the certificate endpoint

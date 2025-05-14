# Product Requirements Document (PRD) for Badge Service

## 1. Overview
The badge service, hosted at `badges.finki.edu.mk`, serves pre-issued badges and certificates as SVG images, identified by a unique short Git commit ID (e.g., `abc123`). The service verifies the validity of each ID against a database and returns SVGs for embedding in HTML via `<img>` or `<object>` tags. A new feature adds a details page, accessible via `/details/<commit_id>`, to display comprehensive information about each badge or certificate when clicked.

## 2. Goals and Objectives
- Serve SVG images for pre-issued badges and certificates, ensuring only valid items are displayed.
- Provide a details page for each badge or certificate, showing metadata like issuer, issuance date, and software details.
- Maintain a simple URL structure similar to [Shields.io](https://shields.io/) or [Badgen.net](https://badgen.net/).
- Ensure badges and certificates are clickable, linking to a details page.
- Allow effortless customization of badge styles.
- Support generation of JPG or PNG images in addition to SVG.

## 3. Functional Requirements

### 3.1 Existing Endpoints (Recap)
- **Small Badge Endpoint**: `badges.finki.edu.mk/badge/<commit_id>`
  - Returns an SVG for a small badge (e.g., `badges.finki.edu.mk/badge/abc123`).
  - Used in `<img>` tags.
- **Large Certificate Endpoint**: `badges.finki.edu.mk/certificate/<commit_id>`
  - Returns an SVG for a large certificate (e.g., `badges.finki.edu.mk/certificate/abc123`).
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
  - **Text Color**: Contrasted on the background by default for readability (e.g., White `#FFFFFF`), or specified (e.g., custom color via parameter).
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
    - `text_color=<hex>`: Sets the text color (e.g., `?text_color=#000000`).
    - `logo=<url>`: Replaces the left section text with an image from the specified URL (e.g., `?logo=https://example.com/logo.png`).
    - `font_size=<px>`: Adjusts text size (e.g., `?font_size=14`).
    - `style=<flat|3d>`: Switches between flat or 3D styling (default: 3D).
  - **Configuration File**: Optionally, support a JSON configuration file per `<commit_id>` (e.g., stored in the database) to define default styles (e.g., `{ "color_left": "#333", "color_right": "#4CAF50" }`).
  - **Default Fallback**: If no parameters or config are provided, use the default design described above.
  - **Validation**: Ensure colors are valid hex codes and sizes are within reasonable limits (e.g., font size 8-16px).

This design ensures the badges are visually consistent, readable, and highly customizable, aligning with industry standards (e.g., [Shields.io](https://shields.io/)).

### 3.2 New Endpoint: Details Page
- **Endpoint**: `badges.finki.edu.mk/details/<commit_id>`
  - Returns an HTML page with details about the badge or certificate.
  - Example: `badges.finki.edu.mk/details/abc123`
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
  <a href="badges.finki.edu.mk/details/abc123">
    <img src="badges.finki.edu.mk/badge/abc123" alt="Certification Badge">
  </a>
  ```

### 3.4 Image Format Support
- **Option 1: New Endpoint**
  - **Endpoint**: `badges.finki.edu.mk/badge/<commit_id>/image`
  - Returns a JPG or PNG image based on a query parameter.
  - Example: `badges.finki.edu.mk/badge/abc123/image?format=png`
  - Supported formats: `format=jpg` or `format=png` (default: SVG if no format specified).
  - Behavior: Convert the SVG to the requested format using an image processing library (e.g., ImageMagick or Go’s `image` package).
  - Content-Type: `image/jpeg` for JPG, `image/png` for PNG.
- **Option 2: Existing Endpoint with Parameter**
  - Enhance `/badge/<commit_id>` to support a `format` query parameter.
  - Example: `badges.finki.edu.mk/badge/abc123?format=png`
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
| Field             | Type   | Description                     | Example                     |
|-------------------|--------|---------------------------------|-----------------------------|
| `commit_id`       | String | Unique ID                      | `abc123`                   |
| `type`            | String | "badge" or "certificate"       | `badge`                    |
| `status`          | String | "valid", "expired", "revoked"  | `valid`                    |
| `issuer`          | String | Certifying authority           | `FINKI Certification Board`|
| `issue_date`      | Date   | Issuance date                  | `2025-05-01`               |
| `software_name`   | String | Software name                  | `MyApp`                    |
| `software_version`| String | Software version               | `v1.3.1`                   |
| `notes`           | String | Additional notes               | `Certified for security`   |
| `svg_content`     | String | Pre-generated SVG (optional)   | `<svg>...</svg>`           |
| `expiry_date`     | Date   | Expiry date (optional)         | `2026-05-01`               |
| `issuer_url`      | String | Issuer's website (optional)    | `https://finki.edu.mk`     |
| `custom_config`   | JSON   | Customization settings (e.g., colors, logo URL) | `{"color_left": "#333", "color_right": "#4CAF50"}` |
| `jpg_content`     | Blob   | Pre-generated JPG (optional)   | Binary JPG data            |
| `png_content`     | Blob   | Pre-generated PNG (optional)   | Binary PNG data            |

- **Notes on New Fields**:
  - `custom_config`: Stores JSON with default customization options (e.g., colors, logo URL) for each badge.
  - `jpg_content` and `png_content`: Store pre-generated images to reduce conversion overhead, populated on demand or during issuance.

## 6. API Specification
| Endpoint                  | Method | Description               | Response         |
|---------------------------|--------|---------------------------|------------------|
| `/badge/<commit_id>`      | GET    | Retrieve small badge SVG, JPG, or PNG | SVG (`image/svg+xml`), JPG (`image/jpeg`), or PNG (`image/png`) |
| `/certificate/<commit_id>`| GET    | Retrieve certificate SVG | SVG (`image/svg+xml`) |
| `/details/<commit_id>`    | GET    | Retrieve details page    | HTML (`text/html`) |

**Query Parameters for `/badge/<commit_id>`**:
- `format=svg|jpg|png`: Specifies the image format (default: `svg`).
- `color_left=<hex>`: Custom left section color.
- `color_right=<hex>`: Custom right section color.
- `text_color=<hex>`: Custom text color.
- `logo=<url>`: URL of a logo image for the left section.
- `font_size=<px>`: Custom font size.
- `style=<flat|3d>`: Badge style.

**Error Handling**:
- Return 404 for invalid `<commit_id>`.
- For `/badge` and `/certificate`, return error image in the requested format.
- For `/details`, return error HTML page.

## 7. User Interface for Details Page
- **Header**: Title (e.g., "Badge/Certificate Details: abc123"), issuer logo.
- **Main Content**:
  - Type, Status, Issuer, Issuance Date, Software Name and Version, Notes, Expiry Date, Embedded SVG (optional).
- **Footer**: Link to `badges.finki.edu.mk`, copyright notice.

## 8. Implementation Notes
- Use Go (Golang) with the standard `net/http` library.
- Use SQLite as the database, stored in the `/db` folder.
- Use the `zap` library for logging.
- Use a template engine (e.g., `html/template` package) for `/details` HTML.
- Cache SVGs, JPGs, and HTML pages (e.g., using an in-memory cache like `sync.Map` or a library like `gocache`).
- Sanitize inputs, use HTTPS.
- **Logging**: 
  - Log all available badges at server startup for monitoring and debugging purposes.
  - Each badge's commit ID, type, status, and software information is logged.
- **Dockerfile**: Implement a multi-stage build to optimize the image size:
  - Stage 1: Build the Go binary.
  - Stage 2: Copy the binary and SQLite DB into a lightweight image (e.g., `alpine`).
- **Makefile**: Include targets:
  - `run`: Start the service locally.
  - `build`: Compile the Go binary.
  - `build-image`: Build the Docker image.
  - `push-image`: Push the Docker image to a registry.
- **Image Conversion**: Use a library like `github.com/disintegration/imaging` to convert SVG to JPG/PNG on demand if not pre-generated.

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
- [x] Design and implement SQLite database schema as per section 5
- [x] Create database initialization and migration scripts
- [x] Implement database connection and query functions
- [x] Create CRUD operations for badge/certificate data
- [x] Implement data validation for database operations

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
- [x] Create middleware for input sanitization and validation
- [x] Implement rate limiting middleware
- [x] Create error handling for all endpoints

### User Interface
- [x] Design and implement HTML template for details page
- [x] Create CSS styles for responsive design
- [x] Implement error page templates
- [x] Ensure accessibility compliance

### Testing
- [x] Write unit tests for core badge generation functions
- [x] Write integration tests for database operations
- [x] Create API endpoint tests
- [ ] Implement performance testing for response time requirements
- [ ] Test image format conversion

### Deployment
- [x] Create Dockerfile with multi-stage build
- [x] Implement Makefile with required targets
- [ ] Set up CI/CD pipeline configuration
- [ ] Create deployment documentation
- [x] Implement graceful shutdown handling

### Documentation
- [x] Create API documentation
- [x] Write user guide for badge/certificate integration
- [x] Document database schema and operations
- [x] Create developer onboarding documentation

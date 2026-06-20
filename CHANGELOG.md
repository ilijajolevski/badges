# SoftwareCertHub — Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-06-20

### Added

- Authenticated admin menu with dedicated admin screens
- `ADMIN_PASSWORD` environment variable to set the default admin password on
  first startup
- Funding footer ("Co-funded by the European Union") on the certificates list
  page

### Changed

- Redesigned page headers with GÉANT branding
- Applied new GÉANT branding to badges, certificates, and the page header
- Co-funded EU logo now rendered at a consistent small size across all pages
- Standardised the project name to **SoftwareCertHub** across all documentation
- Brought documentation into GÉANT Software Artefacts Checklist compliance
  (added Scope, Compatibility, Architecture, Usage, Documentation,
  Troubleshooting, Privacy, Roadmap, and Acknowledgements sections)

### Deprecated

- None.

### Removed

- None.

### Security

- `ADMIN_PASSWORD` is now configurable so deployments can avoid the well-known
  default admin password

## [0.1.2] - 2026-04-08

### Added

- CI/CD pipeline to build and push Docker image on tags

### Fixed

- Docker registry URL corrected to use the proper hostname with port
- Allow hyphen characters in certificate commit IDs

## [0.1.1] - 2026-03-10

### Fixed

- Docker image builds now include version info via `--build-arg` ldflags (previously showed "vdev (unknown)")

## [0.1.0] - 2026-03-05

### Added

- Semantic versioning with build-time injection via `-ldflags`
- `/health` endpoint returning JSON with status, version, and commit
- Version label displayed in the footer of all HTML pages
- Makefile targets: `version`, `bump-patch`, `bump-minor`, `bump-major`
- CHANGELOG.md following Keep a Changelog format
- GitLab CI/CD release pipeline (`.gitlab-ci.yml`)

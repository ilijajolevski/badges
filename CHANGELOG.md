# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-03-05

### Added

- Semantic versioning with build-time injection via `-ldflags`
- `/health` endpoint returning JSON with status, version, and commit
- Version label displayed in the footer of all HTML pages
- Makefile targets: `version`, `bump-patch`, `bump-minor`, `bump-major`
- CHANGELOG.md following Keep a Changelog format
- GitLab CI/CD release pipeline (`.gitlab-ci.yml`)

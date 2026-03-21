# Changelog

All notable changes to Anveesa Vestra are documented here.

## [Unreleased]

### Security
- **BREAKING**: Connection list APIs no longer return decrypted credentials. All operations now use `connection_id` to reference stored credentials server-side.
- Shared link passwords are now sent via `X-Share-Password` header instead of URL query parameters, preventing leakage in logs and browser history.
- X-Forwarded-For header is only trusted when `TRUST_PROXY=true` environment variable is set.
- Stricter rate limiting on authentication endpoints (5 req/min) to prevent brute-force attacks.
- Connection export no longer includes plaintext credentials.

### Added
- **Streaming file transfers**: Cross-provider transfers now stream data through `io.Pipe` instead of buffering entire files in memory, enabling large file transfers.
- **Folder creation**: New "Create folder" button in the file browser, supported across all providers.
- **Concurrent job worker**: Background job queue now processes up to 3 jobs simultaneously.
- **Database migration system**: Schema changes are tracked and applied automatically via numbered migrations.
- **CI pipeline**: GitHub Actions workflow runs Go tests/vet and frontend tests on every PR.
- **Bulk delete jobs**: The `bulk_delete` job type is now fully functional.
- **Sync jobs**: Sync scheduler now actually transfers files between providers.

### Fixed
- All error responses now return consistent JSON format (`{"error": "message"}`). Previously some handlers returned plain text errors.
- `updateSyncJob` now uses a single atomic UPDATE statement instead of multiple separate queries.

### Removed
- Orphan `b2_connections` and `do_connections` database tables (no handlers existed for these).

### Changed
- Frontend operations now pass `connection_id` instead of raw credentials, matching the new secure backend API.

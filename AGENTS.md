# PicPilot Personal development

- The owner authorized public publication on 2026-10-30. Follow LICENSE and LICENSING.md, preserving third-party notices.
- User-owned provider keys are encrypted server-side and are never exposed in API responses or logs.
- Personal tasks must never fall back to another user's or a platform operator's provider credentials.
- All automated provider tests use local mock servers; do not create paid model tasks.
- Do not commit credentials, uploads, databases, backups, dependencies or generated binaries.
- The `core` directory is synchronized from the platform repository's versioned shared module. Do not independently fork its business logic.

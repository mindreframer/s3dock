# Changelog

All notable changes to this project will be documented in this file.

## [v0.1.8]

### Added
- **JSON Output Support**: Added `--json` flag for machine-readable output across commands, enabling easier integration with scripts and automation tools

### Fixed
- **Integration Tests**: Fixed integration test suite

## [v0.1.7]

### Added
- **List Command**: Added `s3dock list` command to query apps, images, tags, and environments
  - `s3dock list apps` - List all applications in the registry
  - `s3dock list images <app>` - List all images for an app (with optional `--month` filter)
  - `s3dock list tags <app>` - List all semantic version tags for an app
  - `s3dock list envs <app>` - List all environment pointers for an app
  - `s3dock list tag-for <app> <env>` - Query which semantic tag was used for an environment promotion

## [v0.1.6]

### Fixed
- **Git Operations from Subdirectories**: Fixed issue where running s3dock from a subdirectory would fail with "repository does not exist" error. All git operations (`IsRepositoryDirty`, `GetCurrentHash`, `GetCommitTimestamp`) now automatically find and use the repository root, allowing s3dock to work from any directory within a git repository

## [v0.1.5]

### Fixed
- **Git Repository Auto-Detection**: Added `FindRepositoryRoot()` method to automatically detect git repository root when using `--context` parameter. Now correctly handles builds where the build context is a subdirectory of the git repository
- **Dockerfile Path Handling**: Fixed absolute dockerfile paths to be converted to relative paths for Docker API. Docker build API requires dockerfile paths to be relative to the build context
- **Symlink Handling**: Added symlink detection and skipping to prevent tar format issues. Symlinks in build context (like `.cursorrules -> AGENT.md`) were causing "archive/tar: write too long" errors
- **Tar Path Length Protection**: Added protection for file paths over 90 characters to prevent tar format limit violations
- **Enhanced .dockerignore Processing**: Improved pattern matching logic and added debug logging to ensure directories like `artifacts/` and `.elixir_ls/` are properly excluded from build context

### Enhanced
- Added comprehensive debug logging for build context creation, path processing, and pattern matching
- Improved error messages and diagnostic output for troubleshooting build issues

## [v0.1.4] - 2025-07-27

- implement the current command: `s3dock current myapp production`

## [v0.1.0] - 2025-07-22

### Features
- Build Docker images with git-based stable tags (enforces clean repo, tags use commit timestamp + git hash)
- Push Docker images to S3 as tar.gz with checksum-based deduplication and metadata
- Semantic version tagging with audit trail
- Promote images to environments (direct or via semantic tags) with atomic pointer updates
- Pull images from S3 and import into Docker, with cleanup of old images
- Show current image for environments with `current` command
- Unified CLI for build, push, tag, promote, pull, current, list, and cleanup
- Profile-based configuration management
- Blue-green deployment support with health checks and automated rollback
- Complete audit trail for push, tag, and promotion events
- Automatic archiving of conflicting or replaced images
- Efficient S3 folder structure for scalable queries and storage 
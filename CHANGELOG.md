# Changelog

All notable changes to this project will be documented in this file.

## [v0.1.0] - 2025-07-22

### Features
- Build Docker images with git-based stable tags (enforces clean repo, tags use commit timestamp + git hash)
- Push Docker images to S3 as tar.gz with checksum-based deduplication and metadata
- Semantic version tagging with audit trail
- Promote images to environments (direct or via semantic tags) with atomic pointer updates
- Pull images from S3 and import into Docker, with cleanup of old images
- Unified CLI for build, push, tag, promote, pull, list, and cleanup
- Profile-based configuration management
- Blue-green deployment support with health checks and automated rollback
- Complete audit trail for push, tag, and promotion events
- Automatic archiving of conflicting or replaced images
- Efficient S3 folder structure for scalable queries and storage 
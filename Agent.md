# s3dock - S3-based Docker Registry

A simple, registry-less way to manage Docker containers using S3 storage.

## Overview

s3dock stores Docker images as tar.gz files in S3 buckets with pointer files for tag management. No centralized registry required - just S3 folder structure and local agents.

## Core Components

### 1. Image Builder (`s3dock build`)
- Build Docker images with git-based stable tags
- Enforces clean repository (no uncommitted changes)
- Tags use commit timestamp + git hash: `app:20250721-2118-f7a5a27`
- Ensures perfect correspondence between image and git state

### 2. Image Publisher (`s3dock push`)
- Export local Docker image to tar.gz
- Upload to S3 with consistent naming (timestamp + git hash)
- Calculate and store image metadata
- Future: Checksum-based deduplication

### 3. Pointer Manager (`s3dock tag`)
- Update pointer JSON files (production.json, staging.json)
- Atomic updates with versioning
- Rollback capability

### 4. Image Puller (`s3dock pull`)
- Download pointer file from S3
- Parse target image path
- Download and import tar.gz into Docker
- Cleanup old images

### 5. Registry CLI (`s3dock`)
- Unified interface: build, push, tag, pull, list, cleanup
- Configuration management
- Profile-based configurations

### 6. Blue-Green Deployment (`s3dock deploy`)
- Environment state tracking
- Health checking before traffic switch
- Traffic switching via load balancer updates
- Rollback automation

## Commands

```bash
# Build with git-based stable tag (requires clean repo)
s3dock build myapp
s3dock build myapp --dockerfile Dockerfile.prod --context ./backend

# Push built image to S3
s3dock push myapp:20250721-2118-f7a5a27

# Tag image for environments
s3dock tag myapp:20250721-2118-f7a5a27 production
s3dock tag myapp:20250721-2118-f7a5a27 staging

# Pull image from environment
s3dock pull myapp production

# Deploy with blue-green strategy
s3dock deploy --blue-green myapp production

# Configuration management
s3dock config show
s3dock config list
s3dock config init
```

## Implementation

- **Language**: Go (AWS SDK, Docker client, single binary)
- **Storage**: S3 buckets with folder structure
- **Deployment**: Blue-green with health checks

## Folder Structure

```
s3://bucket/
    images/myapp/202507/myapp-20250721-2118-f7a5a27.tar.gz
    pointers/myapp/production.json
    pointers/myapp/staging.json
    tags/myapp/v5.0.3.json
```

## Key Features

### Git-Based Stable Tagging
- **No "latest" tags**: Every build gets unique, reproducible tag
- **Stable timestamps**: Uses git commit time, not build time
- **Clean repo enforcement**: Prevents builds with uncommitted changes
- **Format**: `{app}:{YYYYMMDD-HHMM}-{git-hash}`
- **Example**: `myapp:20250721-2118-f7a5a27`

### Reproducible Builds
- Same git commit = identical Docker tag
- Perfect traceability between deployments and code
- Enables reliable rollbacks and environment promotion

## Benefits

- **No registry service**: Just S3 storage - no Docker registry to maintain
- **Git-based reproducibility**: Perfect traceability between code and deployments  
- **Cost effective**: Storage-only costs, leverage S3 features (versioning, lifecycle)
- **Clean builds**: Enforced clean repository prevents deployment surprises
- **Simple pipelines**: Build → Push → Tag → Deploy workflow
- **Future-ready**: Designed for checksum-based deduplication
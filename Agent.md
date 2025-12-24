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
- Calculate MD5 checksum and store metadata alongside image
- Upload to S3 with consistent naming (git timestamp + git hash)
- **Checksum-based deduplication**: Skip upload if identical content exists
- **Archive on mismatch**: Moves existing files to archive/ with timestamp

### 3. Semantic Tagger (`s3dock tag`)
- Create version tags pointing to specific images
- Out-of-order versioning supported (v1.1.5 after v1.2.0)
- Stores git metadata and audit trail

### 4. Environment Promoter (`s3dock promote`)
- Promote images directly to environments
- Promote via semantic version tags
- Atomic pointer updates with metadata
- Supports both direct and indirect references

### 5. Image Puller (`s3dock pull`)
- Download pointer file from S3
- Parse target image path
- Download and import tar.gz into Docker
- Cleanup old images

### 6. Current Image Checker (`s3dock current`)
- Show currently desired image for an environment
- Resolves environment pointers to actual image references
- Supports both direct image and tag-based pointers
- Clean output format: `app:timestamp-hash`

### 7. Listing & Querying (`s3dock list`)
- List all apps, images, tags, or environments
- Query semantic version tag for an environment
- Filter images by year-month
- Shows which tag was used for environment promotion

### 8. Registry CLI (`s3dock`)
- Unified interface: build, push, tag, promote, pull, current, list, cleanup
- Configuration management
- Profile-based configurations

### 9. Blue-Green Deployment (`s3dock deploy`)
- Environment state tracking
- Health checking before traffic switch
- Traffic switching via load balancer updates
- Rollback automation

## Commands

```bash
# Build with git-based stable tag (requires clean repo)
s3dock build myapp
s3dock build myapp --dockerfile Dockerfile.prod --context ./backend
s3dock build myapp --platform linux/amd64  # Cross-platform builds
s3dock build myapp --platform linux/arm64

# Push built image to S3
s3dock push myapp:20250721-2118-f7a5a27

# Create semantic version tags
s3dock tag myapp:20250721-2118-f7a5a27 v1.2.0
s3dock tag myapp:20250720-1045-def5678 v1.1.5

# Promote images to environments (direct)
s3dock promote myapp:20250721-2118-f7a5a27 production
s3dock promote myapp:20250720-1045-def5678 staging

# Promote via semantic versions
s3dock promote myapp v1.2.0 production
s3dock promote myapp v1.1.5 staging

# Pull image from environment
s3dock pull myapp production

# Show current image for environment
s3dock current myapp production
# Output: myapp:20250721-2118-f7a5a27

# List apps, images, tags, environments
s3dock list apps
s3dock list images myapp
s3dock list images myapp --month 202507
s3dock list tags myapp
s3dock list envs myapp

# Query semantic tag for environment (if promoted via tag)
s3dock list tag-for myapp production
# Output: v1.2.0  (or message if promoted directly)

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
    images/myapp/202507/myapp-20250721-2118-f7a5a27.json
    
    tags/myapp/v1.2.0.json              ← Semantic version tags
    tags/myapp/v1.1.5.json
    
    pointers/myapp/production.json      ← Environment pointers
    pointers/myapp/staging.json
    
    audit/myapp/202507/                 ← Audit trail by month
        20250721-1400-push-f7a5a27.json      ← Push events
        20250721-1430-tag-f7a5a27.json       ← Tag events  
        20250721-2118-promotion-f7a5a27.json ← Promotion events
    
    archive/myapp/202507/myapp-20250721-2118-f7a5a27-archived-on-20250722-1018.tar.gz
    archive/myapp/202507/myapp-20250721-2118-f7a5a27-archived-on-20250722-1018.json
```

## Key Features

### Git-Based Stable Tagging
- **No "latest" tags**: Every build gets unique, reproducible tag
- **Stable timestamps**: Uses git commit time, not build time
- **Clean repo enforcement**: Prevents builds with uncommitted changes
- **Format**: `{app}:{YYYYMMDD-HHMM}-{git-hash}`
- **Example**: `myapp:20250721-2118-f7a5a27`

### Semantic Versioning & Environment Promotion
- **Flexible tagging**: Assign semantic versions out-of-order
- **Direct promotion**: `myapp:20250721-2118-f7a5a27` → `production`
- **Tag-based promotion**: `myapp v1.2.0` → `staging`
- **Pointer indirection**: Environments can point to images OR tags
- **Audit trail**: Who promoted what and when

### Complete Audit Trail
- **Event tracking**: Push, tag, and promotion events logged with full context
- **Previous state**: Tracks what was replaced during promotions
- **Structured storage**: `audit/{app}/{YYYYMM}/{timestamp}-{event}-{githash}.json`
- **User attribution**: Records who performed each action
- **Scalable queries**: Monthly folders prevent S3 performance issues

### Checksum-Based Deduplication
- **MD5 verification**: Each image has metadata file with checksum and size
- **Smart uploads**: Skip upload if identical content already exists
- **Conflict resolution**: Archive existing files if checksums don't match
- **Archive structure**: `archive/{app}/{yearmonth}/{filename}-archived-on-{timestamp}`

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
- **Complete audit trail**: Every push, tag, and promotion event logged
- **Efficient storage**: Checksum-based deduplication prevents duplicate uploads
- **Conflict handling**: Automatic archiving when same tag has different content
- **S3 performance**: Audit logs organized by month for scalable queries
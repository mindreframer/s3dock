# s3dock - S3-based Docker Registry

A simple, registry-less way to manage Docker containers using S3 storage.

## Overview

s3dock stores Docker images as tar.gz files in S3 buckets with pointer files for tag management. No centralized registry required - just S3 folder structure and local agents.

## Core Components

### 1. Image Publisher (`s3-docker-push`)
- Export local Docker image to tar.gz
- Upload to S3 with consistent naming (timestamp + git hash)
- Calculate and store image metadata

### 2. Pointer Manager (`s3-docker-tag`)
- Update pointer JSON files (production.json, staging.json)
- Atomic updates with versioning
- Rollback capability

### 3. Image Puller (`s3-docker-pull`)
- Download pointer file from S3
- Parse target image path
- Download and import tar.gz into Docker
- Cleanup old images

### 4. Registry CLI (`s3-docker`)
- Unified interface: push, tag, pull, list, cleanup
- Configuration management

### 5. Blue-Green Deployment (`bg-deploy`)
- Environment state tracking
- Health checking before traffic switch
- Traffic switching via load balancer updates
- Rollback automation

## Commands

```bash
s3dock push myapp:latest
s3dock tag myapp:latest production
s3dock pull myapp production
s3dock deploy --blue-green myapp production
```

## Implementation

- **Language**: Go (AWS SDK, Docker client, single binary)
- **Storage**: S3 buckets with folder structure
- **Deployment**: Blue-green with health checks

## Folder Structure

```
s3://bucket/
    images/myapp/202401/myapp-20240101-abc123.tar.gz
    pointers/myapp/production.json
    pointers/myapp/staging.json
    tags/myapp/v5.0.3.json
```

## Benefits

- No registry service to maintain
- Leverage S3 native features (versioning, lifecycle)
- Cost effective (storage only)
- Simple deployment pipelines
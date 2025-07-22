# s3dock

A lightweight, registry-less Docker image distribution system using S3-compatible storage.

## Philosophy: No 'latest', No Manual Builds

s3dock enforces a reproducible, traceable workflow for Docker images:
- **No `latest` tags**: Every image is tagged with a unique, immutable git-based tag (e.g., `myapp:20250721-2118-f7a5a27`).
- **No manual `docker build`**: All builds must use `s3dock build`, which ensures a clean git state and stable tagging.
- **No mutable tags**: All tags are permanent and correspond to a specific git commit.
- **Why?** This guarantees perfect traceability, reproducibility, and safe promotion/rollback across environments.

## Overview

s3dock eliminates the need for running and maintaining a centralized Docker registry by leveraging S3 storage for Docker image distribution. Images are exported as compressed tar archives and stored with predictable naming patterns, making deployment pipelines simple and cost-effective.

## Key Features

- **No Registry Required** - Uses S3 buckets directly for image storage
- **Profile-based Configuration** - Easily switch between environments (dev, staging, prod)
- **JSON5 Configuration** - Configuration files with comments support
- **Git-aware Naming** - Automatic git commit hash inclusion for traceability
- **Cross-platform** - Supports Linux, macOS, and Windows
- **Integration Ready** - Works with existing CI/CD pipelines
- **Cost Effective** - Pay only for S3 storage, no registry infrastructure

## Installation

### Download Release

Download the latest release for your platform from the [Releases page](https://github.com/mindreframer/s3dock/releases).

Set the version you want to install (see the Releases page for the latest version):

```bash
# Set the version you want to install
export S3DOCK_VERSION=0.1.0

# Linux (x86_64)
curl -L "https://github.com/mindreframer/s3dock/releases/download/v${S3DOCK_VERSION}/s3dock_${S3DOCK_VERSION}_linux_amd64.tar.gz" | tar xz
chmod +x s3dock

# Linux (arm64)
curl -L "https://github.com/mindreframer/s3dock/releases/download/v${S3DOCK_VERSION}/s3dock_${S3DOCK_VERSION}_linux_arm64.tar.gz" | tar xz
chmod +x s3dock

# macOS (Apple Silicon)
curl -L "https://github.com/mindreframer/s3dock/releases/download/v${S3DOCK_VERSION}/s3dock_${S3DOCK_VERSION}_darwin_arm64.tar.gz" | tar xz
chmod +x s3dock

# macOS (Intel)
curl -L "https://github.com/mindreframer/s3dock/releases/download/v${S3DOCK_VERSION}/s3dock_${S3DOCK_VERSION}_darwin_amd64.tar.gz" | tar xz
chmod +x s3dock

# Windows (x86_64)
# Download and extract s3dock_${S3DOCK_VERSION}_windows_amd64.zip from the releases page
```

### Build from Source

```bash
git clone https://github.com/your-org/s3dock.git
cd s3dock
make build
```

## Quick Start

### 1. Create Configuration

```bash
s3dock config init
```

This creates `s3dock.json5` with default configuration:

```json5
{
  "default_profile": "default",
  "profiles": {
    "default": {
      "bucket": "s3dock-containers",
      "region": "us-east-1"
    }
  }
}
```

### 2. Configure AWS Credentials

Either set environment variables:
```bash
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
```

Or add credentials to your config file:
```json5
{
  "profiles": {
    "default": {
      "bucket": "your-bucket-name",
      "region": "us-east-1",
      "access_key": "your-access-key",
      "secret_key": "your-secret-key"
    }
  }
}
```

### 3. Build and Push an Image (with s3dock)

```bash
# Build your Docker image with a git-based, stable tag (requires clean repo)
s3dock build myapp
# Example output tag: myapp:20250721-2118-f7a5a27

# Push to S3
s3dock push myapp:20250721-2118-f7a5a27
```

## Configuration

### Configuration File Locations

s3dock looks for configuration files in the following order:

1. `./s3dock.json5` (project-level)
2. `~/.s3dock/config.json5` (user-level)
3. `/etc/s3dock/config.json5` (system-level)

### Multi-Environment Setup

```json5
{
  "default_profile": "dev",
  
  "profiles": {
    "dev": {
      "bucket": "dev-containers",
      "region": "us-east-1",
      "endpoint": "http://localhost:9000", // MinIO local
      "access_key": "testuser",
      "secret_key": "testpass123"
    },
    
    "staging": {
      "bucket": "staging-containers",
      "region": "us-east-1"
      // Uses AWS credential chain
    },
    
    "prod": {
      "bucket": "prod-containers",
      "region": "us-west-2"
      // Uses IAM roles
    }
  },
  
  "docker": {
    "timeout": "30s",
    "compression": "gzip"
  },
  
  "naming": {
    "include_git_branch": false,
    "timestamp_format": "20060102-1504",
    "path_template": "images/{app}/{year_month}/{filename}"
  }
}
```

### Configuration Precedence

1. Command line flags (`--bucket`, `--profile`)
2. Environment variables (`S3DOCK_BUCKET`, `AWS_REGION`)
3. Configuration file values
4. Built-in defaults

## Commands

### Global Flags

- `--config <path>` - Explicit configuration file path
- `--profile <name>` - Profile to use from configuration
- `--bucket <name>` - Override bucket name
- `--log-level <n>` - Log level (1=error, 2=info, 3=debug)

### Logging

s3dock provides configurable logging levels to help with debugging and troubleshooting:

- **Level 1 (Error)**: Only critical errors are logged
- **Level 2 (Info)**: Normal operations and errors (default)
- **Level 3 (Debug)**: Detailed trace information, operations, and errors

```bash
# Only show errors
s3dock --log-level 1 push myapp:20250721-2118-f7a5a27

# Show normal operations (default)
s3dock --log-level 2 push myapp:20250721-2118-f7a5a27

# Show detailed debug information
s3dock --log-level 3 push myapp:20250721-2118-f7a5a27
```

Debug output includes:
- S3 operation details (upload paths, checksums)
- Docker build context and parameters
- Git operations and hash generation
- File system operations
- Network request details

### Available Commands

#### `build`
Build a Docker image with a git-based, stable tag (requires clean repo).

```bash
s3dock build myapp
s3dock build myapp --dockerfile Dockerfile.prod --context ./backend
# Output: myapp:20250721-2118-f7a5a27
```

#### `push`
Push a Docker image to S3 storage.

```bash
s3dock push myapp:20250721-2118-f7a5a27
s3dock --profile prod push myapp:20250721-2118-f7a5a27
s3dock --bucket custom-bucket push myapp:20250721-2118-f7a5a27
```

#### `tag`
Create semantic version tags pointing to specific images.

```bash
s3dock tag myapp:20250721-2118-f7a5a27 v1.2.0
s3dock tag myapp:20250720-1045-def5678 v1.1.5
```

#### `promote`
Promote images to environments (direct or via semantic version tags).

```bash
s3dock promote myapp:20250721-2118-f7a5a27 production
s3dock promote myapp:20250720-1045-def5678 staging
s3dock promote myapp v1.2.0 production
s3dock promote myapp v1.1.5 staging
```

#### `pull`
Pull image from environment pointer.

```bash
s3dock pull myapp production
```

#### `deploy`
Deploy with blue-green strategy.

```bash
s3dock deploy --blue-green myapp production
```

#### `config`
Configuration management commands.

```bash
# Show current configuration
s3dock config show

# Show specific profile
s3dock config show --profile prod

# List all profiles
s3dock config list

# Create default configuration file
s3dock config init [filename]
```

## Storage Structure

Images are stored in S3 with the following structure:

```
s3://your-bucket/
  images/
    myapp/
      202507/
        myapp-20250721-2118-f7a5a27.tar.gz
  pointers/
    myapp/
      production.json
      staging.json
  tags/
    myapp/
      v1.2.0.json
```

**Naming Convention**: `{app}-{timestamp}-{git-hash}.tar.gz`

## Examples

### Development Workflow

```bash
# Build with git-based stable tag (requires clean repo)
s3dock build myapp
# Example output: myapp:20250721-2118-f7a5a27

# Push built image to S3
s3dock push myapp:20250721-2118-f7a5a27

# Create semantic version tags
s3dock tag myapp:20250721-2118-f7a5a27 v1.2.0

# Promote images to environments
s3dock promote myapp:20250721-2118-f7a5a27 production

# Promote via semantic versions
s3dock promote myapp v1.2.0 production

# Pull image from environment
s3dock pull myapp production
```

### CI/CD Pipeline

```bash
# Build and push in CI (requires clean repo)
s3dock build myapp
s3dock --profile staging push myapp:20250721-2118-f7a5a27

# Production deployment
s3dock --profile prod push myapp:20250721-2118-f7a5a27
```

### Local Development with MinIO

```json5
{
  "profiles": {
    "local": {
      "bucket": "local-containers",
      "region": "us-east-1",
      "endpoint": "http://localhost:9000",
      "access_key": "minioadmin",
      "secret_key": "minioadmin"
    }
  }
}
```

## Development

### Prerequisites

- Go 1.24+
- Docker
- Make

### Building

```bash
# Install development tools
make tools

# Run tests
make test

# Run integration tests (requires Docker)
make test-integration

# Build for current platform
make build

# Build for all platforms
make dist

# Full release build
make release
```

### Project Structure

```
cmd/                    Command implementations
internal/               Internal packages
  config.go             Configuration management
  docker.go             Docker client wrapper
  git.go                Git integration
  pusher.go             Main push logic
  s3.go                 S3 client wrapper
test-configs/           Example configurations
Dockerfile.test         Test image
docker-compose.test.yml Test infrastructure
Makefile                Build automation
```

### Testing

The project includes comprehensive testing:

- **Unit Tests** - Mock-based tests for business logic
- **Integration Tests** - End-to-end tests with real Docker and MinIO
- **Config Tests** - Configuration parsing and resolution

## Architecture

s3dock uses a simple, layered architecture:

1. **CLI Layer** - Command parsing and global flag handling
2. **Configuration Layer** - Profile resolution and precedence handling
3. **Service Layer** - Business logic for push operations
4. **Client Layer** - Docker, S3, and Git integrations

The design emphasizes testability with interface-based dependency injection and comprehensive mocking support.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run `make check` to ensure quality
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

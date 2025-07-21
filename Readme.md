# s3dock

A lightweight, registry-less Docker image distribution system using S3-compatible storage.

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

Download the latest release for your platform from the releases page:

```bash
# Linux
curl -L https://github.com/your-org/s3dock/releases/latest/download/s3dock-linux-amd64 -o s3dock
chmod +x s3dock

# macOS
curl -L https://github.com/your-org/s3dock/releases/latest/download/s3dock-darwin-arm64 -o s3dock
chmod +x s3dock

# Windows
# Download s3dock-windows-amd64.exe from releases
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

### 3. Push an Image

```bash
# Build your Docker image
docker build -t myapp:latest .

# Push to S3
s3dock push myapp:latest
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
s3dock --log-level 1 push myapp:latest

# Show normal operations (default)
s3dock --log-level 2 push myapp:latest

# Show detailed debug information
s3dock --log-level 3 push myapp:latest
```

Debug output includes:
- S3 operation details (upload paths, checksums)
- Docker build context and parameters
- Git operations and hash generation
- File system operations
- Network request details

### Available Commands

#### `push`
Push a Docker image to S3 storage.

```bash
s3dock push <image:tag>
s3dock --profile prod push myapp:v1.2.3
s3dock --bucket custom-bucket push myapp:latest
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
        myapp-20250721-1504-abc1234.tar.gz
  pointers/
    myapp/
      production.json
      staging.json
  tags/
    myapp/
      v1.2.3.json
```

**Naming Convention**: `{app}-{timestamp}-{git-hash}.tar.gz`

## Examples

### Development Workflow

```bash
# Use development environment
s3dock --profile dev push myapp:latest

# Use explicit config file for testing
s3dock --config test-configs/local.json5 push myapp:test
```

### CI/CD Pipeline

```bash
# Build and push in CI
docker build -t myapp:${GIT_SHA} .
s3dock --profile staging push myapp:${GIT_SHA}

# Production deployment
s3dock --profile prod push myapp:${GIT_SHA}
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
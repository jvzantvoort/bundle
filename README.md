# Bundle Library

A Go library and CLI tool for content-addressable, immutable file bundles with SHA256-based integrity verification.

## Overview

Bundle Library implements a Digital Asset Management (DAM) system where files are organized into immutable bundles. Each bundle is uniquely identified by the SHA256 checksum of its contents, ensuring data integrity and enabling reliable deduplication across distributed storage.

## Features

- **Content Integrity**: SHA256 checksums for all files with deterministic bundle identification
- **Bundle Creation**: Create bundles from directories with automatic checksum computation
- **Integrity Verification**: Detect file corruption or modifications
- **Metadata Management**: Human-readable titles and searchable tags
- **CLI Interface**: Command-line tools with both table and JSON output formats
- **Library-First Design**: Standalone Go packages that can be used independently

## Installation

### Prerequisites

- Go 1.21 or later

### Build from Source

```bash
git clone https://github.com/jvzantvoort/bundle.git
cd bundle
go mod download
go build -o bundle ./cmd/bundle
```

### Install to $GOPATH/bin

```bash
go install ./cmd/bundle@latest
```

## Quick Start

### Create a Bundle

```bash
# Create bundle from directory
bundle create /path/to/files --title "My Bundle"
```

### Verify Bundle Integrity

```bash
# Verify all file checksums
bundle verify /path/to/bundle
```

### Manage Tags

```bash
# Add tags
bundle tag add /path/to/bundle travel photos 2024

# List tags
bundle tag list /path/to/bundle
```

### Inspect Bundle

```bash
# Show bundle info
bundle info /path/to/bundle

# List all files
bundle list /path/to/bundle

# Get JSON output
bundle info /path/to/bundle --json
```

## Architecture

Bundle Library follows a library-first architecture with independent components:

- `checksum/` - SHA256 computation and deterministic bundle checksums
- `metadata/` - META.json handling (title, author, timestamps)
- `state/` - STATE.json handling (verification status, replicas)
- `tag/` - TAGS.txt handling (searchable labels)
- `scanner/` - Directory traversal and file discovery
- `lock/` - Concurrency control for write operations
- `bundle/` - High-level bundle operations

CLI commands in `cmd/` use dependency injection to call library functions.

## Documentation

- [Quickstart Guide](specs/001-bundle-core/quickstart.md) - Developer onboarding
- [CLI Commands](specs/001-bundle-core/contracts/cli-commands.md) - Complete CLI reference
- [Data Model](specs/001-bundle-core/data-model.md) - Entity definitions
- [Implementation Plan](specs/001-bundle-core/plan.md) - Technical architecture

## Development

See [quickstart.md](specs/001-bundle-core/quickstart.md) for development setup.

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Format code
gofmt -s -w .

# Static analysis
go vet ./...
```

## License

See [LICENSE](LICENSE) file for details.

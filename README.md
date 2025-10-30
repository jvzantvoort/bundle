[![forthebadge](https://forthebadge.com/images/badges/made-with-crayons.svg)](https://forthebadge.com)
[![forthebadge](https://forthebadge.com/images/badges/designed-in-etch-a-sketch.svg)](https://forthebadge.com)
[![forthebadge](https://forthebadge.com/images/badges/you-didnt-ask-for-this.svg)](https://forthebadge.com)

**CAUTION**: this is a project with which I'm investigating using AI
as a tool for coding. Most if not all of this code is generated.


# Bundle Library

A Go library and CLI tool for content-addressable, immutable file bundles with SHA256-based integrity verification.

## Overview

Bundle Library implements a Digital Asset Management (DAM) system where files are organized into immutable bundles. Each bundle is uniquely identified by the SHA256 checksum of its contents, ensuring data integrity and enabling reliable deduplication across distributed storage.

## Features

- **Content Integrity**: SHA256 checksums for all files with deterministic bundle identification
- **Bundle Creation**: Create bundles from directories with automatic checksum computation
- **Integrity Verification**: Detect file corruption or modifications
- **Metadata Management**: Human-readable titles and searchable tags
- **Centralized Storage**: Import bundles to managed pools with content-addressable storage
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

### Centralized Storage (Pools)

```bash
# Import bundle to centralized pool
bundle import /path/to/bundle

# List all bundles in pool
bundle list_bundles

# Import to specific pool
bundle import /path/to/bundle --pool backup

# Move bundle to archive (removes local copy)
bundle import /path/to/bundle --pool archive --move
```

See [POOLS.md](docs/POOLS.md) for complete pool documentation.

## Architecture

Bundle Library follows a library-first architecture with independent components:

- `checksum/` - SHA256 computation and deterministic bundle checksums
- `metadata/` - META.json handling (title, author, timestamps)
- `state/` - STATE.json handling (verification status, replicas)
- `tag/` - TAGS.txt handling (searchable labels)
- `pool/` - Centralized storage and pool management
- `scanner/` - Directory traversal and file discovery
- `lock/` - Concurrency control for write operations
- `bundle/` - High-level bundle operations

CLI commands in `cmd/` use dependency injection to call library functions.

## Documentation

📚 **[Complete Documentation](docs/README.md)** - Start here for comprehensive guides

### Quick Links

**User Documentation:**
- [Examples & Workflows](docs/EXAMPLES.md) - Common usage patterns
- [Pool Management](docs/POOLS.md) - Centralized storage guide
- [Configuration & Debugging](docs/CONFIG_DEBUGGING.md) - Setup and troubleshooting
- [Bundle Rename](docs/RENAME_UPDATE.md) - Update bundle titles

**Developer Documentation:**
- [API Reference](docs/API.md) - Complete Go package documentation
- [Pool Implementation](docs/POOLS_IMPLEMENTATION.md) - Architecture details
- [Code Quality](docs/GOLANGCI_LINT_FIXES.md) - Linting and best practices

**Project Documentation:**
- [Specification Updates](docs/SPEC_UPDATES.md) - Recent changes
- [Development Summary](docs/SUMMARY.md) - Project status
- [Improvements](docs/IMPROVEMENTS.md) - Future roadmap

**Specifications:**
- [Feature Spec](specs/001-bundle-core/spec.md) - Requirements and user stories
- [Data Model](specs/001-bundle-core/data-model.md) - Entity definitions
- [CLI Commands](specs/001-bundle-core/contracts/cli-commands.md) - Command contracts
- [Implementation Plan](specs/001-bundle-core/plan.md) - Technical architecture

## API Documentation

### Core Packages

Bundle Library exposes several independent packages that can be used programmatically:

#### bundle Package

High-level operations for managing bundles.

```go
import "github.com/jvzantvoort/bundle/bundle"

// Create a new bundle
b, err := bundle.Create("/path/to/files", "My Bundle Title")

// Load an existing bundle
b, err := bundle.Load("/path/to/bundle")

// Verify bundle integrity
verified, corruptedFiles, err := bundle.Verify("/path/to/bundle")
```

**Bundle Type:**
```go
type Bundle struct {
    Path     string                 // Absolute path to bundle directory
    Metadata *metadata.Metadata     // Loaded from META.json
    State    *state.State           // Loaded from STATE.json
    Tags     *tag.Tags              // Loaded from TAGS.txt
    Files    *checksum.ChecksumFile // Loaded from SHA256SUM.txt
}
```

#### metadata Package

Manage bundle metadata (title, author, checksums).

```go
import "github.com/jvzantvoort/bundle/metadata"

// Load metadata
meta, err := metadata.Load("/path/to/bundle")

// Create new metadata
meta := &metadata.Metadata{
    Title:          "My Bundle",
    CreatedAt:      time.Now(),
    BundleChecksum: "abc123...",
    Author:         "username",
    Version:        1,
}

// Save metadata
err := meta.Save("/path/to/bundle")

// Validate metadata
err := meta.Validate()
```

**Metadata Type:**
```go
type Metadata struct {
    Title          string    `json:"title"`           // Human-readable name
    CreatedAt      time.Time `json:"created_at"`      // ISO 8601 timestamp
    BundleChecksum string    `json:"bundle_checksum"` // SHA256 of sorted file checksums
    Author         string    `json:"author"`          // System username
    Version        int       `json:"version"`         // Metadata version (starts at 1)
}
```

#### checksum Package

SHA256 checksum computation and verification.

```go
import "github.com/jvzantvoort/bundle/checksum"

// Create checksum file
files := &checksum.ChecksumFile{}

// Compute checksums for directory
err := files.Compute("/path/to/files")

// Load existing checksums
err := files.Load("/path/to/bundle")

// Save checksums
err := files.Save("/path/to/bundle")

// Verify file integrity
corruptedFiles, err := files.Verify("/path/to/bundle")

// Compute bundle checksum from file checksums
bundleChecksum := checksum.ComputeBundleChecksum(checksums)
```

**ChecksumFile Type:**
```go
type ChecksumFile struct {
    Records []ChecksumRecord
}

type ChecksumRecord struct {
    Checksum string // SHA256 hash (64 hex characters)
    FilePath string // Relative path from bundle root
}
```

#### state Package

Manage operational state (verification status, replicas).

```go
import "github.com/jvzantvoort/bundle/state"

// Load state
st, err := state.Load("/path/to/bundle")

// Create new state
st := &state.State{
    Verified:    true,
    LastChecked: time.Now(),
    Replicas:    []string{},
    SizeBytes:   1024000,
}

// Update verification status
st.MarkVerified(true, time.Now())

// Update size
st.UpdateSize(2048000)

// Add replica location
st.AddReplica("s3://bucket/path")

// Save state
err := st.Save("/path/to/bundle")
```

**State Type:**
```go
type State struct {
    Verified    bool      `json:"verified"`     // Last verification result
    LastChecked time.Time `json:"last_checked"` // Last verification timestamp
    Replicas    []string  `json:"replicas"`     // Known replica locations
    SizeBytes   int64     `json:"size_bytes"`   // Total bundle size (excluding .bundle/)
}
```

#### tag Package

Manage searchable tags.

```go
import "github.com/jvzantvoort/bundle/tag"

// Load tags
tags, err := tag.Load("/path/to/bundle")

// Create new tags
tags := &tag.Tags{Tags: []string{"travel", "photos", "2024"}}

// Add tags (deduplicates automatically)
tags.Add("vacation", "europe")

// Remove tags
tags.Remove("2024")

// Get sorted tag list
tagList := tags.List()

// Save tags
err := tags.Save("/path/to/bundle")
```

**Tags Type:**
```go
type Tags struct {
    Tags []string // Unique, lowercase tag names (1-64 chars, alphanumeric, ., _, -)
}
```

**Tag Validation:**
- Converted to lowercase for case-insensitive matching
- Must match pattern: `^[a-z0-9._-]{1,64}$`
- Automatically deduplicated

#### lock Package

File-based locking for concurrent write operations.

```go
import "github.com/jvzantvoort/bundle/lock"

// Acquire lock
bundleLock, err := lock.AcquireLock("/path/to/bundle")
if err != nil {
    // Lock is held by another process
}
defer bundleLock.Release()

// Perform write operations...
```

### CLI Commands

All CLI commands support `--json` output for programmatic use.

#### create

Create a new bundle from a directory.

```bash
bundle create <path> --title "My Bundle"
```

**JSON Output:**
```json
{
  "status": "created",
  "path": "/path/to/bundle",
  "checksum": "abc123...",
  "files": 42,
  "size_bytes": 1024000,
  "title": "My Bundle",
  "created_at": "2024-01-15T10:30:00Z"
}
```

#### info

Display bundle information.

```bash
bundle info <path> [--json]
```

**JSON Output:**
```json
{
  "path": "/path/to/bundle",
  "title": "My Bundle",
  "checksum": "abc123...",
  "files": 42,
  "size_bytes": 1024000,
  "created_at": "2024-01-15T10:30:00Z",
  "author": "username",
  "verified": true,
  "tags": ["travel", "photos"],
  "replicas": ["s3://bucket/path"]
}
```

#### list

List all files in a bundle.

```bash
bundle list <path> [--json]
```

**JSON Output:**
```json
{
  "path": "/path/to/bundle",
  "files": [
    {
      "path": "photo1.jpg",
      "checksum": "abc123...",
      "size_bytes": 2048000
    }
  ],
  "total_files": 42,
  "total_size": 1024000
}
```

#### verify

Verify bundle integrity by recomputing checksums.

```bash
bundle verify <path> [--json]
```

**JSON Output:**
```json
{
  "status": "valid",
  "files_checked": 42,
  "last_verified": "2024-01-15T10:30:00Z",
  "corrupted_files": []
}
```

Or on failure:
```json
{
  "status": "invalid",
  "files_checked": 42,
  "last_verified": "2024-01-15T10:30:00Z",
  "corrupted_files": ["photo1.jpg", "document.pdf"]
}
```

#### tag add

Add tags to a bundle.

```bash
bundle tag add <path> <tag1> [<tag2>...]
```

**JSON Output:**
```json
{
  "status": "added",
  "path": "/path/to/bundle",
  "tags": ["travel", "photos", "vacation"]
}
```

#### tag remove

Remove tags from a bundle.

```bash
bundle tag remove <path> <tag1> [<tag2>...]
```

**JSON Output:**
```json
{
  "status": "removed",
  "path": "/path/to/bundle",
  "tags": ["vacation"]
}
```

#### tag list

List all tags on a bundle.

```bash
bundle tag list <path> [--json]
```

**JSON Output:**
```json
{
  "path": "/path/to/bundle",
  "tags": ["travel", "photos"]
}
```

### Bundle Structure

A bundle is a directory with the following structure:

```
my-bundle/
├── .bundle/
│   ├── META.json      # Metadata (title, author, checksum)
│   ├── STATE.json     # Operational state (verified, replicas)
│   ├── TAGS.txt       # Searchable tags (one per line)
│   ├── SHA256SUM.txt  # File checksums
│   └── .lock          # Lock file (temporary)
├── file1.jpg
├── file2.pdf
└── documents/
    └── file3.txt
```

### Exit Codes

- `0` - Success
- `1` - User error (invalid input, path not found, etc.)
- `2` - System error (I/O error, JSON marshal error, etc.)

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

# Data Model: Bundle Library Core

**Feature**: 001-bundle-core  
**Date**: 2025-10-30  
**Status**: Design Complete

## Overview

This document defines the Go struct definitions and validation rules for Bundle Library entities, as specified in the feature requirements.

---

## Core Entities

### 1. Metadata (META.json)

**Purpose**: Stores human-readable bundle information and content-derived identity.

**Go Struct**:
```go
package metadata

import "time"

// Metadata represents the bundle metadata stored in .bundle/META.json
type Metadata struct {
    Title          string    `json:"title"`            // Human-readable name
    CreatedAt      time.Time `json:"created_at"`       // ISO 8601 timestamp
    BundleChecksum string    `json:"bundle_checksum"`  // SHA256 of sorted file checksums
    Author         string    `json:"author"`           // System username
    Version        int       `json:"version"`          // Metadata version (starts at 1)
}
```

**Validation Rules**:
- `Title`: Optional, max 256 characters, UTF-8
- `CreatedAt`: Required, RFC3339 format (`time.RFC3339`)
- `BundleChecksum`: Required, exactly 64 hex characters (SHA256)
- `Author`: Required, non-empty string (from `os.Getenv("USER")` or equivalent)
- `Version`: Required, positive integer starting at 1

**File Format Example**:
```json
{
  "title": "Iceland Vacation 2024",
  "created_at": "2025-10-30T10:48:42Z",
  "bundle_checksum": "a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890",
  "author": "jvzantvoort",
  "version": 1
}
```

**Operations**:
- `Load(bundlePath string) (*Metadata, error)` - Read from `.bundle/META.json`
- `Save(bundlePath string) error` - Write to `.bundle/META.json`
- `Validate() error` - Check all validation rules

---

### 2. State (STATE.json)

**Purpose**: Tracks operational state (verification status, replicas, size).

**Go Struct**:
```go
package state

import "time"

// State represents the bundle operational state stored in .bundle/STATE.json
type State struct {
    Verified    bool      `json:"verified"`      // Last verification result
    LastChecked time.Time `json:"last_checked"`  // Last verification timestamp
    Replicas    []string  `json:"replicas"`      // Known replica locations
    SizeBytes   int64     `json:"size_bytes"`    // Total bundle size (excluding .bundle/)
}
```

**Validation Rules**:
- `Verified`: Boolean, default `false`
- `LastChecked`: Optional (zero value if never verified), RFC3339 format
- `Replicas`: Array of valid URIs (file://, s3://, ssh://), can be empty
- `SizeBytes`: Non-negative integer

**File Format Example**:
```json
{
  "verified": true,
  "last_checked": "2025-10-30T14:02:00Z",
  "replicas": [
    "file:///mnt/nas/bundles/iceland-2024",
    "s3://backup-bucket/iceland-2024"
  ],
  "size_bytes": 12439123456
}
```

**Operations**:
- `Load(bundlePath string) (*State, error)` - Read from `.bundle/STATE.json`
- `Save(bundlePath string) error` - Write to `.bundle/STATE.json`
- `MarkVerified(timestamp time.Time)` - Update verification status
- `AddReplica(uri string) error` - Append replica location
- `UpdateSize(bytes int64)` - Set total size

---

### 3. ChecksumRecord (SHA256SUM.txt entry)

**Purpose**: Represents a single file checksum entry in the SHA256SUM format.

**Go Struct**:
```go
package checksum

// ChecksumRecord represents a single file checksum entry
type ChecksumRecord struct {
    Checksum string // SHA256 hash (64 hex characters)
    FilePath string // Relative path from bundle root
}

// ChecksumFile represents the entire SHA256SUM.txt file
type ChecksumFile struct {
    Records []ChecksumRecord
}
```

**Validation Rules**:
- `Checksum`: Required, exactly 64 hex characters (lowercase)
- `FilePath`: Required, relative path, UTF-8, no `.bundle/` prefix

**File Format** (SHA256SUM.txt):
```text
0f343b0931126a20f133d67c2b018a3b49ecf84816be857c8b95e23c8e0a3d10  ./IMG_001.jpg
f3bbbd66a63d4bf1747940578ec3d010c09e5c2e7dd5e57c14c47f9b6c9a8b21  ./IMG_002.jpg
a7d5c8f9e1234567890abcdef1234567890abcdef1234567890abcdef123456  ./video.mp4
```

**Format Details**:
- Two spaces between checksum and filepath
- Filepath starts with `./` prefix
- Sorted lexicographically by checksum (for determinism)

**Operations**:
- `Load(bundlePath string) (*ChecksumFile, error)` - Parse SHA256SUM.txt
- `Save(bundlePath string) error` - Write SHA256SUM.txt in sorted order
- `Compute(bundlePath string) (*ChecksumFile, error)` - Scan directory and compute all checksums
- `Verify(bundlePath string) ([]string, error)` - Return list of corrupted file paths

---

### 4. Tags (TAGS.txt)

**Purpose**: User-assigned classification labels for bundle organization.

**Go Struct**:
```go
package tag

// Tags represents the collection of tags associated with a bundle
type Tags struct {
    Tags []string // Unique, case-sensitive tag names
}
```

**Validation Rules**:
- Each tag: Non-empty, max 64 characters, UTF-8, no newlines
- Duplicates automatically removed
- Tags stored in sorted order (for consistency)

**File Format** (TAGS.txt):
```text
travel
iceland
vacation
2024
photos
```

**Format Details**:
- One tag per line
- UTF-8 encoding
- No empty lines
- Sorted alphabetically (case-sensitive)

**Operations**:
- `Load(bundlePath string) (*Tags, error)` - Read from `.bundle/TAGS.txt`
- `Save(bundlePath string) error` - Write to `.bundle/TAGS.txt` in sorted order
- `Add(tags ...string) error` - Append tags (deduplicate)
- `Remove(tags ...string) error` - Remove tags
- `List() []string` - Return sorted tag list

---

### 5. Bundle (High-level aggregate)

**Purpose**: Unified interface for bundle operations.

**Go Struct**:
```go
package bundle

import (
    "github.com/jvzantvoort/bundle/checksum"
    "github.com/jvzantvoort/bundle/metadata"
    "github.com/jvzantvoort/bundle/state"
    "github.com/jvzantvoort/bundle/tag"
)

// Bundle represents a complete bundle with all metadata and state
type Bundle struct {
    Path     string                  // Absolute path to bundle directory
    Metadata *metadata.Metadata      // Loaded from META.json
    State    *state.State            // Loaded from STATE.json
    Tags     *tag.Tags               // Loaded from TAGS.txt
    Files    *checksum.ChecksumFile  // Loaded from SHA256SUM.txt
}
```

**Operations**:
- `Load(path string) (*Bundle, error)` - Load all bundle metadata
- `Create(path string, title string) (*Bundle, error)` - Initialize new bundle
- `Verify() (bool, []string, error)` - Verify integrity, return (success, corrupted_files, error)
- `Info() (*BundleInfo, error)` - Return summary information
- `Reload() error` - Re-read all metadata files

---

### 6. Pool Configuration

**Purpose**: Defines centralized storage locations for bundle collections.

**Go Struct**:
```go
package pool

// Pool represents a centralized bundle storage location
type Pool struct {
    Name  string // Pool identifier (from config)
    Root  string // Absolute path to pool root directory
    Title string // Human-readable pool name
}

// Config represents the application configuration
type Config struct {
    Pools    map[string]PoolConfig `yaml:"pools"`
    LogLevel string                `yaml:"log_level"`
}

// PoolConfig represents a pool configuration in config.yaml
type PoolConfig struct {
    Root  string `yaml:"root"`
    Title string `yaml:"title"`
}
```

**Validation Rules**:
- `Name`: Required, valid identifier (from config key)
- `Root`: Required, absolute path, must be a directory
- `Title`: Optional, human-readable description

**Configuration File** (`~/.config/bundle/config.yaml`):
```yaml
pools:
  default:
    root: /mnt/bundles
    title: Default Bundle Pool
  
  backup:
    root: /backup/bundles
    title: Backup Pool
  
  archive:
    root: /archive/bundles
    title: Archive Pool

log_level: info
```

**Storage Layout**:
```text
/mnt/bundles/                                           # Pool root
├── a1b2c3d4e5f67890.../                               # Bundle (by checksum)
│   ├── file1.jpg
│   ├── file2.mp4
│   └── .bundle/
│       ├── META.json
│       ├── STATE.json
│       ├── TAGS.txt
│       └── SHA256SUM.txt
└── f7e8d9c0b1a2345.../                                # Another bundle
    ├── data.txt
    └── .bundle/
        └── ...
```

**Operations**:
- `GetPool(name string) (*Pool, error)` - Load pool from config
- `ListPools() ([]Pool, error)` - List all configured pools
- `Import(bundlePath string, move bool) error` - Import bundle to pool
- `ListBundles() ([]*metadata.Metadata, error)` - List bundles in pool

**Configuration Loading**:
1. Search paths (in order):
   - `./config.yaml`
   - `~/.config/bundle/config.yaml`
   - `/etc/bundle/config.yaml`
2. First file found is loaded
3. Missing pools section is allowed (pools optional)

---

## Relationships

```text
Bundle
├── Metadata (1:1) - Stored in .bundle/META.json
├── State (1:1) - Stored in .bundle/STATE.json
├── Tags (1:1) - Stored in .bundle/TAGS.txt
└── Files (1:N) - Stored in .bundle/SHA256SUM.txt
    └── ChecksumRecord (N) - One per file in bundle

Pool
└── Bundles (1:N) - Stored in <pool_root>/<checksum>/
    └── Bundle (N) - Each bundle stored by checksum
```

---

## File System Layout

```text
bundle-directory/
├── file1.jpg                  # Actual files (user data)
├── file2.mp4
├── subdir/
│   └── file3.txt
└── .bundle/                   # Metadata subdirectory
    ├── META.json              # Metadata entity
    ├── STATE.json             # State entity
    ├── TAGS.txt               # Tags entity
    ├── SHA256SUM.txt          # ChecksumFile entity
    └── .lock                  # Lock file (temporary, not persisted)
```

---

## Validation Summary

| Entity | Required Fields | Constraints |
|--------|-----------------|-------------|
| Metadata | CreatedAt, BundleChecksum, Author, Version | Checksum 64 hex chars, Version ≥ 1 |
| State | - (all optional) | SizeBytes ≥ 0, Replicas valid URIs |
| ChecksumRecord | Checksum, FilePath | Checksum 64 hex chars, FilePath relative |
| Tags | - (can be empty) | Each tag ≤ 64 chars, no newlines |
| Bundle | Path | Path must exist and contain .bundle/ subdir |

---

## State Transitions

### Bundle Lifecycle

```text
[Directory] 
    ↓ bundle create
[Bundle (unverified)]
    ↓ bundle verify (success)
[Bundle (verified)]
    ↓ file modification
[Bundle (unverified, corrupted)]
    ↓ bundle verify (failure)
[Bundle (unverified, corrupted)] → User must fix manually
```

### Metadata Version Transitions

```text
Version 1 (initial creation)
    ↓ Title change
Version 2 (metadata updated, checksum unchanged)
    ↓ Tag addition
Version 2 (no version change, tags not in metadata)
```

**Note**: Only metadata changes (title, author) increment version. Tag changes do NOT increment metadata version (tags are separate entity).

---

## Error Conditions

| Condition | Error Type | Recovery |
|-----------|------------|----------|
| Missing .bundle/ directory | `ErrNotABundle` | Run `bundle create` |
| Invalid JSON in META.json | System error | Restore from backup or recreate |
| Checksum mismatch | `ErrCorruptedBundle` | User must identify and fix corrupted files |
| Lock file exists | `ErrBundleLocked` | Wait for operation to complete or run `bundle unlock` |
| Missing SHA256SUM.txt | `ErrIncompleteBundle` | Run `bundle rebuild` (future command) |

---

## Serialization Notes

### JSON Encoding
- Use `json.MarshalIndent()` with 2-space indentation for human readability
- Use `time.RFC3339` for timestamp serialization
- Ensure UTF-8 encoding for all files

### Checksum File Encoding
- Plain text, UTF-8
- Two spaces between checksum and path (standard sha256sum format)
- Sort by checksum (not filepath) for determinism

### Tags File Encoding
- Plain text, UTF-8
- One tag per line
- Sort alphabetically (case-sensitive)
- Trim whitespace from each tag

---

## Future Enhancements

- **Bundle Version**: Add `bundle_version` to track file content changes (new checksum)
- **Signatures**: Add `.bundle/SIGNATURE` for cryptographic verification
- **Compression**: Add `.bundle/COMPRESSED` flag for storage optimization
- **Deduplication**: Add `dedup_id` to identify identical files across bundles

---

## References

- Spec: [spec.md](./spec.md)
- Research: [research.md](./research.md)
- Plan: [plan.md](./plan.md)

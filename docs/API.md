# Bundle Library API Documentation

Complete API reference for the Bundle Library Go packages.

---

## Table of Contents

- [Overview](#overview)
- [Package: bundle](#package-bundle)
- [Package: metadata](#package-metadata)
- [Package: state](#package-state)
- [Package: checksum](#package-checksum)
- [Package: tag](#package-tag)
- [Package: pool](#package-pool)
- [Package: config](#package-config)
- [Package: lock](#package-lock)
- [Package: messages](#package-messages)
- [Error Handling](#error-handling)
- [Examples](#examples)

---

## Overview

The Bundle Library provides a content-addressable digital asset management system with the following key features:

- **Content Addressing**: Bundles identified by SHA256 checksum of contents
- **Integrity Verification**: Detect file corruption or tampering
- **Pool Management**: Centralized storage with automatic deduplication
- **Metadata Management**: Human-readable titles and tags
- **CLI & API**: Both command-line and programmatic access

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    CLI Commands                          │
│  (create, verify, info, list, tag, rename, import, etc) │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                  Bundle Package                          │
│         High-level bundle operations                     │
└──┬──────┬──────┬──────┬──────┬──────┬──────┬───────────┘
   │      │      │      │      │      │      │
   ▼      ▼      ▼      ▼      ▼      ▼      ▼
┌──────┬──────┬──────┬──────┬──────┬──────┬──────────┐
│Meta  │State │Check │Tags  │Pool  │Lock  │Messages  │
│data  │      │sum   │      │      │      │          │
└──────┴──────┴──────┴──────┴──────┴──────┴──────────┘
```

---

## Package: bundle

High-level API for bundle operations.

### Types

#### Bundle
```go
type Bundle struct {
    Path     string
    Metadata *metadata.Metadata
    State    *state.State
    Tags     *tag.Tags
    Files    *checksum.ChecksumFile
}
```

Represents a complete bundle with all metadata and state.

**Fields:**
- `Path` - Absolute path to bundle directory
- `Metadata` - Bundle metadata (title, checksum, author, etc.)
- `State` - Operational state (verified, replicas, size)
- `Tags` - User-assigned tags
- `Files` - File checksums

### Functions

#### Load
```go
func Load(path string) (*Bundle, error)
```

Load an existing bundle from disk.

**Parameters:**
- `path` - Path to bundle directory

**Returns:**
- `*Bundle` - Loaded bundle
- `error` - Error if not a bundle or load fails

**Example:**
```go
b, err := bundle.Load("./my-photos")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Title: %s\n", b.Metadata.Title)
```

#### Create
```go
func Create(path string, title string) (*Bundle, error)
```

Create a new bundle from an existing directory.

**Parameters:**
- `path` - Directory to convert to bundle
- `title` - Human-readable title

**Returns:**
- `*Bundle` - Newly created bundle
- `error` - Error on failure

**Example:**
```go
b, err := bundle.Create("./vacation-photos", "Iceland 2024")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Created: %s\n", b.Metadata.BundleChecksum)
```

#### Verify
```go
func (b *Bundle) Verify() (bool, []string, error)
```

Verify bundle integrity by checking all file checksums.

**Returns:**
- `bool` - True if all checksums valid
- `[]string` - List of corrupted file paths (empty if valid)
- `error` - Error on I/O failure

**Example:**
```go
valid, corrupted, err := b.Verify()
if err != nil {
    log.Fatal(err)
}
if !valid {
    fmt.Printf("Corrupted files: %v\n", corrupted)
}
```

#### Info
```go
func (b *Bundle) Info() (*BundleInfo, error)
```

Get bundle summary information.

**Returns:**
- `*BundleInfo` - Bundle summary
- `error` - Error on failure

**Example:**
```go
info, err := b.Info()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Size: %d bytes\n", info.SizeBytes)
```

---

## Package: metadata

Bundle metadata management.

### Types

#### Metadata
```go
type Metadata struct {
    Title          string    `json:"title"`
    CreatedAt      time.Time `json:"created_at"`
    BundleChecksum string    `json:"bundle_checksum"`
    Author         string    `json:"author"`
    Version        int       `json:"version"`
}
```

Bundle metadata stored in `.bundle/META.json`.

**Fields:**
- `Title` - Human-readable name
- `CreatedAt` - Creation timestamp (RFC3339)
- `BundleChecksum` - SHA256 of file checksums (64 hex chars)
- `Author` - Creator username
- `Version` - Metadata version (starts at 1)

### Functions

#### Load
```go
func Load(bundlePath string) (*Metadata, error)
```

Load metadata from `.bundle/META.json`.

**Example:**
```go
meta, err := metadata.Load("./my-bundle")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Title: %s\n", meta.Title)
```

#### Save
```go
func (m *Metadata) Save(bundlePath string) error
```

Save metadata to `.bundle/META.json`.

**Example:**
```go
meta.Title = "Updated Title"
err := meta.Save("./my-bundle")
```

#### UpdateTitle
```go
func UpdateTitle(bundlePath string, newTitle string) (string, error)
```

Update bundle title and return old title.

**Parameters:**
- `bundlePath` - Path to bundle
- `newTitle` - New title

**Returns:**
- `string` - Previous title
- `error` - Error on failure

**Example:**
```go
oldTitle, err := metadata.UpdateTitle("./my-bundle", "New Title")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Updated: %s → New Title\n", oldTitle)
```

#### Validate
```go
func (m *Metadata) Validate() error
```

Validate metadata fields.

**Returns:**
- `error` - Validation error or nil

---

## Package: state

Bundle operational state management.

### Types

#### State
```go
type State struct {
    Verified    bool      `json:"verified"`
    LastChecked time.Time `json:"last_checked"`
    Replicas    []string  `json:"replicas"`
    SizeBytes   int64     `json:"size_bytes"`
}
```

Operational state stored in `.bundle/STATE.json`.

**Fields:**
- `Verified` - Last verification result
- `LastChecked` - Last verification timestamp
- `Replicas` - Known replica locations (URIs)
- `SizeBytes` - Total bundle size (excluding .bundle/)

### Functions

#### Load
```go
func Load(bundlePath string) (*State, error)
```

Load state from `.bundle/STATE.json`.

#### Save
```go
func (s *State) Save(bundlePath string) error
```

Save state to `.bundle/STATE.json`.

#### MarkVerified
```go
func (s *State) MarkVerified(timestamp time.Time)
```

Update verification status.

**Example:**
```go
s.MarkVerified(time.Now())
err := s.Save("./my-bundle")
```

#### AddReplica
```go
func (s *State) AddReplica(uri string) error
```

Add replica location.

**Parameters:**
- `uri` - Replica URI (file://, s3://, ssh://)

#### UpdateSize
```go
func (s *State) UpdateSize(bytes int64)
```

Set total bundle size.

---

## Package: checksum

File checksum computation and verification.

### Types

#### ChecksumRecord
```go
type ChecksumRecord struct {
    Checksum string
    FilePath string
}
```

Single file checksum entry.

**Fields:**
- `Checksum` - SHA256 hash (64 hex chars)
- `FilePath` - Relative path from bundle root

#### ChecksumFile
```go
type ChecksumFile struct {
    Records []ChecksumRecord
}
```

Collection of file checksums.

### Functions

#### Load
```go
func Load(bundlePath string) (*ChecksumFile, error)
```

Parse `SHA256SUM.txt`.

#### Save
```go
func (cf *ChecksumFile) Save(bundlePath string) error
```

Write `SHA256SUM.txt` in sorted order.

#### Compute
```go
func Compute(bundlePath string) (*ChecksumFile, error)
```

Scan directory and compute all checksums.

**Example:**
```go
checksums, err := checksum.Compute("./my-bundle")
if err != nil {
    log.Fatal(err)
}
err = checksums.Save("./my-bundle")
```

#### Verify
```go
func (cf *ChecksumFile) Verify(bundlePath string) ([]string, error)
```

Verify all checksums, return corrupted file paths.

**Example:**
```go
corrupted, err := checksums.Verify("./my-bundle")
if len(corrupted) > 0 {
    fmt.Printf("Corrupted: %v\n", corrupted)
}
```

---

## Package: tag

Tag management for bundle organization.

### Types

#### Tags
```go
type Tags struct {
    Tags []string
}
```

Collection of user-assigned tags.

### Functions

#### Load
```go
func Load(bundlePath string) (*Tags, error)
```

Read tags from `.bundle/TAGS.txt`.

#### Save
```go
func (t *Tags) Save(bundlePath string) error
```

Write tags to `.bundle/TAGS.txt` (sorted).

#### Add
```go
func (t *Tags) Add(tags ...string) error
```

Add tags (deduplicate).

**Example:**
```go
t := &tag.Tags{}
t.Add("vacation", "travel", "iceland")
err := t.Save("./my-bundle")
```

#### Remove
```go
func (t *Tags) Remove(tags ...string) error
```

Remove tags.

#### List
```go
func (t *Tags) List() []string
```

Return sorted tag list.

---

## Package: pool

Centralized bundle storage management.

### Types

#### Pool
```go
type Pool struct {
    Name  string
    Root  string
    Title string
}
```

Centralized storage location.

**Fields:**
- `Name` - Pool identifier
- `Root` - Absolute path to pool root
- `Title` - Human-readable description

#### PoolConfig
```go
type PoolConfig struct {
    Root  string `yaml:"root"`
    Title string `yaml:"title"`
}
```

Pool configuration in YAML.

### Functions

#### GetPool
```go
func GetPool(name string) (*Pool, error)
```

Load pool from configuration.

**Example:**
```go
pool, err := pool.GetPool("default")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Pool root: %s\n", pool.Root)
```

#### ListPools
```go
func ListPools() ([]Pool, error)
```

List all configured pools.

#### Import
```go
func (p *Pool) Import(bundlePath string, move bool) error
```

Import bundle to pool.

**Parameters:**
- `bundlePath` - Source bundle path
- `move` - True to move, false to copy

**Example:**
```go
pool, _ := pool.GetPool("default")
err := pool.Import("./my-bundle", false)
```

#### ListBundles
```go
func (p *Pool) ListBundles() ([]*metadata.Metadata, error)
```

List all bundles in pool.

**Example:**
```go
bundles, err := pool.ListBundles()
for _, meta := range bundles {
    fmt.Printf("%s: %s\n", meta.BundleChecksum[:12], meta.Title)
}
```

---

## Package: config

Configuration management.

### Types

#### Config
```go
type Config struct {
    Pools    map[string]PoolConfig `yaml:"pools"`
    LogLevel string                `yaml:"log_level"`
}
```

Application configuration.

### Functions

#### Load
```go
func Load() (*Config, error)
```

Load configuration from search paths.

**Search Order:**
1. `./config.yaml`
2. `~/.config/bundle/config.yaml`
3. `/etc/bundle/config.yaml`

**Example:**
```go
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}
for name, pool := range cfg.Pools {
    fmt.Printf("Pool %s: %s\n", name, pool.Root)
}
```

#### GetConfigPath
```go
func GetConfigPath() string
```

Return path to loaded config file.

---

## Package: lock

Bundle locking for concurrent access.

### Functions

#### Acquire
```go
func Acquire(bundlePath string) error
```

Acquire exclusive lock on bundle.

**Example:**
```go
err := lock.Acquire("./my-bundle")
if err != nil {
    log.Fatal("Bundle is locked")
}
defer lock.Release("./my-bundle")
```

#### Release
```go
func Release(bundlePath string) error
```

Release bundle lock.

#### IsLocked
```go
func IsLocked(bundlePath string) bool
```

Check if bundle is locked.

---

## Package: messages

Structured output for CLI commands.

### Types

#### BundleInfo
```go
type BundleInfo struct {
    Path           string    `json:"path"`
    Title          string    `json:"title"`
    BundleChecksum string    `json:"bundle_checksum"`
    Author         string    `json:"author"`
    CreatedAt      time.Time `json:"created_at"`
    Verified       bool      `json:"verified"`
    LastChecked    time.Time `json:"last_checked"`
    SizeBytes      int64     `json:"size_bytes"`
    FileCount      int       `json:"file_count"`
    TagCount       int       `json:"tag_count"`
}
```

Bundle information for display.

#### ImportResult
```go
type ImportResult struct {
    Status      string `json:"status"`
    Operation   string `json:"operation"`
    Pool        string `json:"pool"`
    PoolRoot    string `json:"pool_root"`
    Source      string `json:"source"`
    Destination string `json:"destination"`
    Checksum    string `json:"checksum"`
}
```

Import operation result.

#### RenameResult
```go
type RenameResult struct {
    Status   string `json:"status"`
    Path     string `json:"path"`
    OldTitle string `json:"old_title"`
    NewTitle string `json:"new_title"`
    Title    string `json:"title"`
}
```

Rename operation result.

---

## Error Handling

### Standard Errors

#### ErrNotABundle
```go
var ErrNotABundle = errors.New("not a bundle: missing .bundle directory")
```

Directory is not a bundle.

#### ErrBundleLocked
```go
var ErrBundleLocked = errors.New("bundle is locked by another process")
```

Bundle is currently locked.

#### ErrCorruptedBundle
```go
var ErrCorruptedBundle = errors.New("bundle integrity check failed")
```

Bundle has corrupted files.

#### ErrPoolNotFound
```go
var ErrPoolNotFound = errors.New("pool not found in configuration")
```

Pool name not in config.

### Error Checking

```go
b, err := bundle.Load("./path")
if err != nil {
    if errors.Is(err, bundle.ErrNotABundle) {
        // Not a bundle
    } else if errors.Is(err, bundle.ErrBundleLocked) {
        // Locked
    } else {
        // Other error
    }
}
```

---

## Examples

### Create and Verify Bundle

```go
package main

import (
    "fmt"
    "log"
    "github.com/jvzantvoort/bundle/bundle"
)

func main() {
    // Create bundle
    b, err := bundle.Create("./photos", "Vacation Photos")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Created bundle: %s\n", b.Metadata.BundleChecksum)
    
    // Verify integrity
    valid, corrupted, err := b.Verify()
    if err != nil {
        log.Fatal(err)
    }
    
    if valid {
        fmt.Println("Bundle is valid")
    } else {
        fmt.Printf("Corrupted files: %v\n", corrupted)
    }
}
```

### Import to Pool

```go
package main

import (
    "fmt"
    "log"
    "github.com/jvzantvoort/bundle/pool"
)

func main() {
    // Get pool
    p, err := pool.GetPool("default")
    if err != nil {
        log.Fatal(err)
    }
    
    // Import bundle
    err = p.Import("./my-bundle", false)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Bundle imported successfully")
    
    // List all bundles
    bundles, err := p.ListBundles()
    if err != nil {
        log.Fatal(err)
    }
    
    for _, meta := range bundles {
        fmt.Printf("%s: %s\n", meta.BundleChecksum[:12], meta.Title)
    }
}
```

### Update Metadata

```go
package main

import (
    "fmt"
    "log"
    "github.com/jvzantvoort/bundle/metadata"
    "github.com/jvzantvoort/bundle/tag"
)

func main() {
    // Update title
    oldTitle, err := metadata.UpdateTitle("./my-bundle", "New Title")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Renamed: %s → New Title\n", oldTitle)
    
    // Add tags
    tags := &tag.Tags{}
    err = tags.Load("./my-bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    tags.Add("vacation", "2024", "iceland")
    err = tags.Save("./my-bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Tags: %v\n", tags.List())
}
```

### Custom Checksum Verification

```go
package main

import (
    "fmt"
    "log"
    "github.com/jvzantvoort/bundle/checksum"
)

func main() {
    // Compute checksums
    checksums, err := checksum.Compute("./my-bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    // Save to file
    err = checksums.Save("./my-bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    // Verify
    corrupted, err := checksums.Verify("./my-bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    if len(corrupted) == 0 {
        fmt.Println("All files verified")
    } else {
        fmt.Printf("Corrupted: %v\n", corrupted)
    }
}
```

---

## See Also

- [Examples Guide](EXAMPLES.md) - Common usage patterns
- [Pool Management](POOLS.md) - Pool operations
- [Configuration](CONFIG_DEBUGGING.md) - Configuration details
- [Specifications](../specs/001-bundle-core/) - Formal specifications

---

**Last Updated**: 2025-10-30  
**API Version**: 1.0.0

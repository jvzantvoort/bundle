# Bundle Library - Code Examples

This document provides comprehensive examples for using the Bundle Library both as a Go package and via the CLI.

## Table of Contents

- [Go Package Examples](#go-package-examples)
  - [Bundle Operations](#bundle-operations)
  - [Checksum Operations](#checksum-operations)
  - [Metadata Management](#metadata-management)
  - [State Management](#state-management)
  - [Tag Management](#tag-management)
  - [Locking](#locking)
  - [Directory Scanning](#directory-scanning)
- [CLI Examples](#cli-examples)
  - [Creating Bundles](#creating-bundles)
  - [Verifying Bundles](#verifying-bundles)
  - [Managing Tags](#managing-tags)
  - [Bundle Information](#bundle-information)
  - [JSON Output](#json-output)

## Go Package Examples

### Bundle Operations

#### Create a Bundle

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/jvzantvoort/bundle/bundle"
)

func main() {
    // Create a new bundle from a directory
    b, err := bundle.Create("/path/to/photos", "Vacation 2024")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Created bundle:\n")
    fmt.Printf("  Path: %s\n", b.Path)
    fmt.Printf("  Title: %s\n", b.Metadata.Title)
    fmt.Printf("  Checksum: %s\n", b.Metadata.BundleChecksum)
    fmt.Printf("  Files: %d\n", len(b.Files.Records))
    fmt.Printf("  Size: %d bytes\n", b.State.SizeBytes)
}
```

#### Load and Inspect a Bundle

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/jvzantvoort/bundle/bundle"
)

func main() {
    // Load existing bundle
    b, err := bundle.Load("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    // Display bundle information
    fmt.Printf("Bundle: %s\n", b.Metadata.Title)
    fmt.Printf("Author: %s\n", b.Metadata.Author)
    fmt.Printf("Created: %s\n", b.Metadata.CreatedAt.Format("2006-01-02 15:04:05"))
    fmt.Printf("Checksum: %s\n", b.Metadata.BundleChecksum)
    fmt.Printf("Files: %d\n", len(b.Files.Records))
    fmt.Printf("Tags: %v\n", b.Tags.List())
    
    // List all files
    fmt.Println("\nFiles:")
    for _, record := range b.Files.Records {
        fmt.Printf("  %s (%s)\n", record.FilePath, record.Checksum[:8])
    }
}
```

#### Verify Bundle Integrity

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/jvzantvoort/bundle/bundle"
)

func main() {
    // Verify bundle integrity
    verified, corrupted, err := bundle.Verify("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    if verified {
        fmt.Println("✓ Bundle integrity verified")
    } else {
        fmt.Println("✗ Bundle integrity check failed")
        fmt.Println("\nCorrupted files:")
        for _, file := range corrupted {
            fmt.Printf("  - %s\n", file)
        }
    }
}
```

### Checksum Operations

#### Compute Directory Checksums

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/jvzantvoort/bundle/checksum"
)

func main() {
    // Create checksum file and compute checksums
    files := &checksum.ChecksumFile{}
    err := files.Compute("/path/to/directory")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Computed %d checksums:\n", len(files.Records))
    for _, record := range files.Records {
        fmt.Printf("  %s  %s\n", record.Checksum[:16], record.FilePath)
    }
    
    // Save to bundle
    err = files.Save("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
}
```

#### Compute Bundle Checksum

```go
package main

import (
    "fmt"
    
    "github.com/jvzantvoort/bundle/checksum"
)

func main() {
    // File checksums (in any order)
    checksums := []string{
        "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
        "d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f",
        "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e8",
    }
    
    // Compute deterministic bundle checksum
    bundleChecksum := checksum.ComputeBundleChecksum(checksums)
    fmt.Printf("Bundle checksum: %s\n", bundleChecksum)
}
```

#### Verify File Checksums

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/jvzantvoort/bundle/checksum"
)

func main() {
    // Load checksums
    files := &checksum.ChecksumFile{}
    err := files.Load("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    // Verify all files
    corrupted, err := files.Verify("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    if len(corrupted) == 0 {
        fmt.Println("All files verified successfully")
    } else {
        fmt.Printf("Found %d corrupted files:\n", len(corrupted))
        for _, file := range corrupted {
            fmt.Printf("  - %s\n", file)
        }
    }
}
```

### Metadata Management

#### Create and Save Metadata

```go
package main

import (
    "log"
    "time"
    
    "github.com/jvzantvoort/bundle/metadata"
)

func main() {
    // Create metadata
    meta := &metadata.Metadata{
        Title:          "Project Documentation",
        CreatedAt:      time.Now(),
        BundleChecksum: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
        Author:         "username",
        Version:        1,
    }
    
    // Validate before saving
    if err := meta.Validate(); err != nil {
        log.Fatal("Invalid metadata:", err)
    }
    
    // Save to bundle
    if err := meta.Save("/path/to/bundle"); err != nil {
        log.Fatal(err)
    }
}
```

#### Load and Update Metadata

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/jvzantvoort/bundle/metadata"
)

func main() {
    // Load metadata
    meta, err := metadata.Load("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Current title: %s\n", meta.Title)
    
    // Update title
    meta.Title = "Updated Title"
    
    // Save changes
    if err := meta.Save("/path/to/bundle"); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("New title: %s\n", meta.Title)
}
```

### State Management

#### Track Verification Status

```go
package main

import (
    "log"
    "time"
    
    "github.com/jvzantvoort/bundle/state"
)

func main() {
    // Load state
    st, err := state.Load("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    // Mark as verified
    st.MarkVerified(true, time.Now())
    
    // Update size
    st.UpdateSize(1024000)
    
    // Save changes
    if err := st.Save("/path/to/bundle"); err != nil {
        log.Fatal(err)
    }
}
```

#### Manage Replicas

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/jvzantvoort/bundle/state"
)

func main() {
    // Load state
    st, err := state.Load("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    // Add replica locations
    st.AddReplica("s3://bucket/backup/bundle")
    st.AddReplica("/mnt/backup/bundle")
    st.AddReplica("rsync://server/bundles/bundle")
    
    // Save changes
    if err := st.Save("/path/to/bundle"); err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Replicas:")
    for _, replica := range st.Replicas {
        fmt.Printf("  - %s\n", replica)
    }
}
```

### Tag Management

#### Add and Remove Tags

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/jvzantvoort/bundle/tag"
)

func main() {
    // Load tags
    tags, err := tag.Load("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    // Add tags
    tags.Add("vacation", "europe", "2024")
    fmt.Printf("Added tags: %v\n", tags.List())
    
    // Remove a tag
    tags.Remove("2024")
    fmt.Printf("After removal: %v\n", tags.List())
    
    // Save changes
    if err := tags.Save("/path/to/bundle"); err != nil {
        log.Fatal(err)
    }
}
```

#### Tag Normalization

```go
package main

import (
    "fmt"
    
    "github.com/jvzantvoort/bundle/tag"
)

func main() {
    tags := &tag.Tags{}
    
    // Tags are normalized to lowercase
    tags.Add("Vacation", "EUROPE", "Travel")
    
    // Duplicates are filtered
    tags.Add("vacation", "europe")
    
    // Invalid tags are ignored
    tags.Add("my tag")  // Contains space - invalid
    tags.Add("")        // Empty - invalid
    
    fmt.Printf("Tags: %v\n", tags.List())
    // Output: Tags: [europe travel vacation]
}
```

### Locking

#### Safe Concurrent Operations

```go
package main

import (
    "log"
    
    "github.com/jvzantvoort/bundle/bundle"
    "github.com/jvzantvoort/bundle/lock"
)

func main() {
    // Acquire exclusive lock
    bundleLock, err := lock.AcquireLock("/path/to/bundle")
    if err != nil {
        log.Fatal("Bundle is locked:", err)
    }
    defer bundleLock.Release()
    
    // Perform operations safely
    b, err := bundle.Load("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    b.Metadata.Title = "Updated Title"
    if err := b.Metadata.Save("/path/to/bundle"); err != nil {
        log.Fatal(err)
    }
    
    // Lock is automatically released via defer
}
```

### Directory Scanning

#### Scan Directory

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/jvzantvoort/bundle/scanner"
)

func main() {
    // Scan directory (excludes .bundle/)
    files, err := scanner.ScanDirectory("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d files:\n", len(files))
    for _, file := range files {
        fmt.Printf("  %s\n", file)
    }
}
```

#### Scan with Symlinks

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/jvzantvoort/bundle/scanner"
)

func main() {
    // Scan and follow symlinks
    files, err := scanner.ScanWithSymlinks("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d files (following symlinks):\n", len(files))
    for _, file := range files {
        fmt.Printf("  %s\n", file)
    }
}
```

## CLI Examples

### Creating Bundles

#### Basic Bundle Creation

```bash
# Create bundle with title
bundle create ./photos --title "Vacation 2024"

# Create bundle with verbose output
bundle create ./documents --title "Project Docs" --verbose

# Create bundle with JSON output
bundle create ./backup --title "Backup" --json
```

Output:
```
Bundle Created
--------------
Path:     /home/user/photos
Checksum: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
Title:    Vacation 2024
Created:  2024-01-15 10:30:00
Files:    42
Size:     1024000 bytes
```

### Verifying Bundles

#### Verify Bundle Integrity

```bash
# Verify bundle
bundle verify ./photos

# Verify with verbose output
bundle verify ./photos --verbose

# Get JSON verification result
bundle verify ./photos --json
```

Output:
```
Bundle Integrity: VALID
```

### Managing Tags

#### Add Tags

```bash
# Add single tag
bundle tag add ./photos vacation

# Add multiple tags
bundle tag add ./photos travel europe 2024

# Add tags with JSON output
bundle tag add ./photos summer beach --json
```

#### Remove Tags

```bash
# Remove single tag
bundle tag remove ./photos 2024

# Remove multiple tags
bundle tag remove ./photos summer beach
```

#### List Tags

```bash
# List all tags
bundle tag list ./photos

# List with JSON output
bundle tag list ./photos --json
```

Output:
```
europe
travel
vacation
```

### Bundle Information

#### Display Bundle Info

```bash
# Show bundle information
bundle info ./photos

# Show with JSON output
bundle info ./photos --json

# Show with verbose logging
bundle info ./photos --verbose
```

Output:
```
Bundle Information
------------------
Path:     /home/user/photos
Title:    Vacation 2024
Checksum: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
Author:   username
Created:  2024-01-15 10:30:00
Files:    42
Size:     1024000
```

#### List Bundle Files

```bash
# List all files in bundle
bundle list ./photos

# List with JSON output
bundle list ./photos --json
```

Output:
```
+------------------+------------------------------------------------------------------+--------+
|    FILENAME      |                             CHECKSUM                             |  SIZE  |
+------------------+------------------------------------------------------------------+--------+
| photo1.jpg       | e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7... | 2.0 MB |
| photo2.jpg       | d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f... | 1.5 MB |
+------------------+------------------------------------------------------------------+--------+

Total: 42 files, 1.0 GB
```

#### Rename Bundle

```bash
# Update bundle title
bundle rename ./photos "Summer Vacation 2024"

# Rename with JSON output
bundle rename ./photos "New Title" --json
```

### JSON Output

#### JSON Output Examples

All commands support `--json` flag for machine-readable output:

```bash
# Create bundle
$ bundle create ./photos --title "Vacation" --json
{
  "status": "created",
  "path": "/home/user/photos",
  "checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "files": 42,
  "size_bytes": 1024000,
  "title": "Vacation",
  "created_at": "2024-01-15T10:30:00Z"
}

# Verify bundle
$ bundle verify ./photos --json
{
  "status": "valid",
  "files_checked": 42,
  "last_verified": "2024-01-15T10:30:00Z",
  "corrupted_files": []
}

# Bundle info
$ bundle info ./photos --json
{
  "path": "/home/user/photos",
  "title": "Vacation",
  "checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "files": 42,
  "size_bytes": 1024000,
  "created_at": "2024-01-15T10:30:00Z",
  "author": "username",
  "verified": true,
  "tags": ["travel", "vacation"],
  "replicas": []
}

# List files
$ bundle list ./photos --json
{
  "path": "/home/user/photos",
  "files": [
    {
      "path": "photo1.jpg",
      "checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      "size_bytes": 2048000
    },
    {
      "path": "photo2.jpg",
      "checksum": "d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f",
      "size_bytes": 1536000
    }
  ],
  "total_files": 42,
  "total_size": 1024000
}
```

## Error Handling

### Exit Codes

The CLI uses consistent exit codes:

- `0` - Success
- `1` - User error (invalid input, bundle not found, corrupted bundle)
- `2` - System error (I/O error, permissions, etc.)

Example:
```bash
#!/bin/bash

bundle verify ./photos
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo "Success"
elif [ $EXIT_CODE -eq 1 ]; then
    echo "User error"
elif [ $EXIT_CODE -eq 2 ]; then
    echo "System error"
fi
```

## Advanced Usage

### Scripting with Bundle CLI

```bash
#!/bin/bash

# Create bundle
bundle create ./data --title "Daily Backup $(date +%Y-%m-%d)" --json > create.json

# Extract checksum
CHECKSUM=$(jq -r '.checksum' create.json)
echo "Bundle checksum: $CHECKSUM"

# Add tags
bundle tag add ./data backup "$(date +%Y)" "$(date +%m)"

# Verify
if bundle verify ./data --json | jq -e '.status == "valid"' > /dev/null; then
    echo "Backup verified successfully"
    # Copy to remote location
    rsync -av ./data/ /mnt/backup/
else
    echo "Backup verification failed!"
    exit 1
fi
```

### Integration with CI/CD

```yaml
# .github/workflows/bundle.yml
name: Create and Verify Bundle

on: [push]

jobs:
  bundle:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install bundle CLI
        run: go install github.com/jvzantvoort/bundle/cmd/bundle@latest
      
      - name: Create bundle
        run: bundle create ./artifacts --title "Build ${{ github.sha }}" --json
      
      - name: Verify bundle
        run: bundle verify ./artifacts
      
      - name: Upload bundle
        uses: actions/upload-artifact@v3
        with:
          name: bundle
          path: ./artifacts
```

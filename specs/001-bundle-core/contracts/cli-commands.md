# CLI Command Contracts: Bundle Library

**Feature**: 001-bundle-core  
**Date**: 2025-10-30  
**Status**: Contract Specification Complete

## Overview

This document specifies the command-line interface contracts for the Bundle CLI tool. All commands follow Constitution Principle III (CLI-First Interface): text-based I/O, JSON support, and standard exit codes.

---

## Global Conventions

### Exit Codes
- **0**: Success
- **1**: User error (invalid arguments, bundle not found, corrupted bundle)
- **2**: System error (I/O failure, permission denied, out of disk space)

### Output Modes
- **Table mode** (default): Human-readable output using tablewriter
- **JSON mode** (`--json` flag): Machine-parseable JSON output

### Logging
- **Stdout**: User-facing data (tables, JSON)
- **Stderr**: Operational logs (via logrus), error messages

### Common Flags
- `--json`: Output JSON instead of table format
- `--verbose` / `-v`: Enable DEBUG level logging
- `--quiet` / `-q`: Suppress INFO level logs (errors only)

---

## Commands

### 1. `bundle create <path> [flags]`

**Description**: Create a new bundle from a directory.

**Arguments**:
- `<path>`: Directory to bundle (required)

**Flags**:
- `--title <title>`: Human-readable bundle name (optional)
- `--json`: Output JSON instead of table

**Behavior**:
1. Scan directory for all files (excluding `.bundle/`)
2. Compute SHA256 checksum for each file
3. Generate deterministic bundle checksum from sorted file checksums
4. Create `.bundle/` subdirectory
5. Write `META.json`, `SHA256SUM.txt`, `STATE.json`, `TAGS.txt` (empty)

**Exit Codes**:
- **0**: Bundle created successfully
- **1**: Path does not exist or is not a directory
- **2**: I/O error (permission denied, disk full)

**Output (Table Mode)**:
```text
Bundle Created
--------------
Path:     /home/user/photos/iceland-2024
Checksum: a1b2c3d4e5f67890...
Files:    42
Size:     1.2 GB
Title:    Iceland Vacation 2024
```

**Output (JSON Mode)**:
```json
{
  "status": "created",
  "path": "/home/user/photos/iceland-2024",
  "checksum": "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678",
  "files": 42,
  "size_bytes": 1288490188,
  "title": "Iceland Vacation 2024",
  "created_at": "2025-10-30T10:48:42Z"
}
```

**Error Examples**:
```bash
# Exit code 1 (user error)
$ bundle create /nonexistent
Error: directory does not exist: /nonexistent

# Exit code 2 (system error)
$ bundle create /root/protected
System error: permission denied: /root/protected
```

---

### 2. `bundle verify <path> [flags]`

**Description**: Verify bundle integrity by recomputing checksums.

**Arguments**:
- `<path>`: Bundle directory (required)

**Flags**:
- `--json`: Output JSON instead of table

**Behavior**:
1. Check if `.bundle/` subdirectory exists
2. Load `SHA256SUM.txt`
3. Recompute checksum for each file
4. Compare against stored checksums
5. Update `STATE.json` with verification result and timestamp

**Exit Codes**:
- **0**: Bundle integrity valid (all checksums match)
- **1**: Bundle corrupted (checksum mismatch) OR not a bundle
- **2**: I/O error during verification

**Output (Success, Table Mode)**:
```text
Bundle Integrity: VALID
---------------------
Files Checked: 42
Last Verified: 2025-10-30 10:48:42
```

**Output (Success, JSON Mode)**:
```json
{
  "status": "valid",
  "files_checked": 42,
  "last_verified": "2025-10-30T10:48:42Z",
  "corrupted_files": []
}
```

**Output (Failure, Table Mode)**:
```text
Bundle Integrity: INVALID
------------------------
Files Checked: 42
Corrupted Files:
- IMG_001.jpg (expected: 0f343b09..., found: deadbeef...)
- video.mp4 (file missing)
```

**Output (Failure, JSON Mode)**:
```json
{
  "status": "invalid",
  "files_checked": 42,
  "last_verified": "2025-10-30T10:48:42Z",
  "corrupted_files": [
    {
      "path": "IMG_001.jpg",
      "expected": "0f343b0931126a20f133d67c2b018a3b49ecf84816be857c8b95e23c8e0a3d10",
      "found": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
    },
    {
      "path": "video.mp4",
      "expected": "a7d5c8f9e1234567890abcdef1234567890abcdef1234567890abcdef123456",
      "found": null,
      "error": "file missing"
    }
  ]
}
```

---

### 3. `bundle info <path> [flags]`

**Description**: Display bundle metadata and statistics.

**Arguments**:
- `<path>`: Bundle directory (required)

**Flags**:
- `--json`: Output JSON instead of table

**Behavior**:
1. Load `META.json`, `STATE.json`, `TAGS.txt`, `SHA256SUM.txt`
2. Display summary information

**Exit Codes**:
- **0**: Successfully displayed info
- **1**: Not a bundle
- **2**: I/O error reading metadata

**Output (Table Mode)**:
```text
Bundle Information
------------------
Path:          /home/user/photos/iceland-2024
Title:         Iceland Vacation 2024
Checksum:      a1b2c3d4e5f67890...
Files:         42
Size:          1.2 GB
Created:       2025-10-30 10:48:42
Author:        jvzantvoort
Last Verified: 2025-10-30 14:02:00
Status:        Verified
Tags:          travel, iceland, vacation, 2024, photos (5)
```

**Output (JSON Mode)**:
```json
{
  "path": "/home/user/photos/iceland-2024",
  "title": "Iceland Vacation 2024",
  "checksum": "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678",
  "files": 42,
  "size_bytes": 1288490188,
  "created_at": "2025-10-30T10:48:42Z",
  "author": "jvzantvoort",
  "last_verified": "2025-10-30T14:02:00Z",
  "verified": true,
  "tags": ["travel", "iceland", "vacation", "2024", "photos"],
  "replicas": ["file:///mnt/nas/bundles/iceland-2024"]
}
```

---

### 4. `bundle list <path> [flags]`

**Description**: List all files in a bundle with checksums and sizes.

**Arguments**:
- `<path>`: Bundle directory (required)

**Flags**:
- `--json`: Output JSON instead of table

**Behavior**:
1. Load `SHA256SUM.txt`
2. Get file sizes via filesystem stat
3. Display file inventory

**Exit Codes**:
- **0**: Successfully listed files
- **1**: Not a bundle
- **2**: I/O error

**Output (Table Mode)**:
```text
Files in Bundle
---------------
Filename         Checksum                                                          Size
--------         --------                                                          ----
IMG_001.jpg      0f343b0931126a20f133d67c2b018a3b49ecf84816be857c8b95e23c8e0a3d10  2.4 MB
IMG_002.jpg      f3bbbd66a63d4bf1747940578ec3d010c09e5c2e7dd5e57c14c47f9b6c9a8b21  3.1 MB
video.mp4        a7d5c8f9e1234567890abcdef1234567890abcdef1234567890abcdef123456  150 MB
...

Total: 42 files, 1.2 GB
```

**Output (JSON Mode)**:
```json
{
  "files": [
    {
      "path": "IMG_001.jpg",
      "checksum": "0f343b0931126a20f133d67c2b018a3b49ecf84816be857c8b95e23c8e0a3d10",
      "size_bytes": 2516582
    },
    {
      "path": "IMG_002.jpg",
      "checksum": "f3bbbd66a63d4bf1747940578ec3d010c09e5c2e7dd5e57c14c47f9b6c9a8b21",
      "size_bytes": 3251200
    },
    {
      "path": "video.mp4",
      "checksum": "a7d5c8f9e1234567890abcdef1234567890abcdef1234567890abcdef123456",
      "size_bytes": 157286400
    }
  ],
  "total_files": 42,
  "total_bytes": 1288490188
}
```

---

### 5. `bundle tag add <path> <tag> [<tag>...]`

**Description**: Add one or more tags to a bundle.

**Arguments**:
- `<path>`: Bundle directory (required)
- `<tag>`: Tag name(s) (required, multiple allowed)

**Behavior**:
1. Load existing `TAGS.txt`
2. Append new tags (deduplicate)
3. Save sorted tag list back to `TAGS.txt`

**Exit Codes**:
- **0**: Tags added successfully
- **1**: Not a bundle OR invalid tag (empty, contains newline)
- **2**: I/O error

**Output (Table Mode)**:
```text
Tags Added
----------
Added 3 tags: travel, iceland, photos
Total tags: 5
```

**Output (JSON Mode)**:
```json
{
  "status": "added",
  "added_tags": ["travel", "iceland", "photos"],
  "total_tags": 5,
  "all_tags": ["2024", "iceland", "photos", "travel", "vacation"]
}
```

---

### 6. `bundle tag remove <path> <tag> [<tag>...]`

**Description**: Remove one or more tags from a bundle.

**Arguments**:
- `<path>`: Bundle directory (required)
- `<tag>`: Tag name(s) (required, multiple allowed)

**Behavior**:
1. Load existing `TAGS.txt`
2. Remove specified tags
3. Save updated tag list

**Exit Codes**:
- **0**: Tags removed successfully (even if tag didn't exist)
- **1**: Not a bundle
- **2**: I/O error

**Output (Table Mode)**:
```text
Tags Removed
------------
Removed 2 tags: old, archived
Total tags: 3
```

**Output (JSON Mode)**:
```json
{
  "status": "removed",
  "removed_tags": ["old", "archived"],
  "total_tags": 3,
  "all_tags": ["iceland", "photos", "travel"]
}
```

---

### 7. `bundle tag list <path> [flags]`

**Description**: List all tags associated with a bundle.

**Arguments**:
- `<path>`: Bundle directory (required)

**Flags**:
- `--json`: Output JSON instead of table

**Behavior**:
1. Load `TAGS.txt`
2. Display sorted tag list

**Exit Codes**:
- **0**: Successfully listed tags
- **1**: Not a bundle
- **2**: I/O error

**Output (Table Mode)**:
```text
Tags
----
travel
iceland
vacation
2024
photos

Total: 5 tags
```

**Output (JSON Mode)**:
```json
{
  "tags": ["travel", "iceland", "vacation", "2024", "photos"],
  "total": 5
}
```

---

## Future Commands (Out of Scope for MVP)

### `bundle rebuild <path>`
Rebuild `SHA256SUM.txt` from current files (if accidentally deleted).

### `bundle unlock <path>`
Manually remove stale lock file.

### `bundle rename <path> <new_title>`
Update bundle title in `META.json`.

### `bundle replicate <path> <destination>`
Copy bundle to remote storage and update `STATE.json` replicas.

---

## Testing Contract

### CLI Contract Tests

All commands MUST be tested for:
1. **Exit codes**: Verify correct 0/1/2 for success/user error/system error
2. **JSON output**: Verify valid JSON structure with expected fields
3. **Table output**: Verify human-readable formatting (visual inspection)
4. **Error messages**: Verify clear, actionable error messages on stderr

### Example Test (Pseudocode)

```go
func TestCLI_CreateCommand_ExitCodes(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        wantExit int
    }{
        {"valid path", []string{"create", validPath}, 0},
        {"nonexistent path", []string{"create", "/nonexistent"}, 1},
        {"permission denied", []string{"create", "/root/test"}, 2},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := exec.Command("bundle", tt.args...)
            err := cmd.Run()
            assert.Equal(t, tt.wantExit, getExitCode(err))
        })
    }
}
```

---

## References

- Spec: [spec.md](../spec.md)
- Data Model: [data-model.md](../data-model.md)
- Research: [research.md](../research.md)
- Plan: [plan.md](../plan.md)

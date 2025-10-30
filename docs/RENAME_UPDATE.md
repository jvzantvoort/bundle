# Bundle Rename/Update Title - Implementation Summary

## Overview

Enhanced the existing `bundle rename` command with better documentation, a helper function, and improved output.

---

## What Was Done

### 1. Enhanced Message Files

Updated the rename command help text:

**messages/use/rename:**
```
rename <path> <new_title>
```

**messages/short/rename:**
```
Update the title of a bundle
```

**messages/long/rename:**
Complete documentation with:
- Description of what the command does
- Examples for both standard and JSON output
- Notes about what is/isn't changed

### 2. Added Metadata Helper Function

**File:** `metadata/main.go`

Added `UpdateTitle()` function:

```go
func UpdateTitle(bundlePath string, newTitle string) error {
    // Load existing metadata
    meta, err := Load(bundlePath)
    if err != nil {
        return fmt.Errorf("failed to load metadata: %w", err)
    }

    // Update title
    meta.Title = newTitle

    // Save back to disk
    if err := meta.Save(bundlePath); err != nil {
        return fmt.Errorf("failed to save metadata: %w", err)
    }

    return nil
}
```

**Benefits:**
- Single function to update title
- Proper error handling
- Error wrapping for context
- Reusable in other code

### 3. Enhanced Rename Command

**File:** `cmd/bundle/rename.go`

**Improvements:**
- Complete documentation with examples
- Uses metadata.UpdateTitle() helper
- Shows old → new title in output
- Enhanced JSON output with both old and new titles
- Better debug logging
- Backward compatible JSON output

**Removed:**
- Unused flags (--tag, --title) that weren't implemented

---

## Usage

### Command Syntax

```bash
bundle rename <path> <new_title> [--json]
```

### Examples

#### Update Title
```bash
$ bundle rename ./my-bundle "New Title"
INFO[...] Title updated: Original Title → New Title
```

#### JSON Output
```bash
$ bundle rename ./my-bundle "New Title" --json
{
  "status": "renamed",
  "path": "./my-bundle",
  "old_title": "Original Title",
  "new_title": "New Title",
  "title": "New Title"
}
```

**JSON Fields:**
- `status`: Always "renamed"
- `path`: Bundle path
- `old_title`: Previous title
- `new_title`: Updated title
- `title`: New title (for backward compatibility)

#### Verbose Mode
```bash
$ bundle rename ./my-bundle "New Title" --verbose
DEBU[...] Updating title for bundle: ./my-bundle
DEBU[...] New title: New Title
DEBU[...] Old title: Original Title
DEBU[...] Title updated successfully
INFO[...] Title updated: Original Title → New Title
```

---

## What Gets Changed

### Modified
✅ Title field in .bundle/META.json

### Unchanged
❌ Bundle checksum
❌ Author
❌ Creation date
❌ File contents
❌ Tags
❌ State
❌ Checksums

**Only the title is updated. Everything else remains exactly the same.**

---

## API Usage

### Using the Helper Function

```go
package main

import (
    "log"
    "github.com/jvzantvoort/bundle/metadata"
)

func main() {
    // Update bundle title
    err := metadata.UpdateTitle("/path/to/bundle", "New Title")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("Title updated successfully")
}
```

### Manual Update

```go
package main

import (
    "log"
    "github.com/jvzantvoort/bundle/metadata"
)

func main() {
    // Load metadata
    meta, err := metadata.Load("/path/to/bundle")
    if err != nil {
        log.Fatal(err)
    }
    
    // Update title
    meta.Title = "New Title"
    
    // Save
    if err := meta.Save("/path/to/bundle"); err != nil {
        log.Fatal(err)
    }
}
```

---

## Testing

### Manual Test

```bash
# Create test bundle
mkdir /tmp/test-bundle
echo "content" > /tmp/test-bundle/file.txt
bundle create /tmp/test-bundle --title "Original"

# Check title
bundle info /tmp/test-bundle --json | jq .title
# Output: "Original"

# Rename
bundle rename /tmp/test-bundle "Updated"
# Output: Title updated: Original → Updated

# Verify
bundle info /tmp/test-bundle --json | jq .title
# Output: "Updated"

# Test JSON output
bundle rename /tmp/test-bundle "Final" --json
# Output:
# {
#   "status": "renamed",
#   "path": "/tmp/test-bundle",
#   "old_title": "Updated",
#   "new_title": "Final",
#   "title": "Final"
# }
```

### Automated Tests

✅ All existing tests pass
✅ Contract test validates rename functionality
✅ JSON output validated

---

## Files Modified

1. **metadata/main.go** - Added UpdateTitle() helper function
2. **cmd/bundle/rename.go** - Enhanced command implementation
3. **messages/use/rename** - Updated usage syntax
4. **messages/short/rename** - Updated short description
5. **messages/long/rename** - Added complete documentation

**Total:** 5 files modified

---

## Verification

### Build Status
```bash
$ go build ./...
✅ SUCCESS
```

### Test Status
```bash
$ go test ./...
ok      github.com/jvzantvoort/bundle/bundle           0.003s
ok      github.com/jvzantvoort/bundle/checksum         (cached)
ok      github.com/jvzantvoort/bundle/metadata         (cached)
ok      github.com/jvzantvoort/bundle/tag              (cached)
ok      github.com/jvzantvoort/bundle/tests/contract   0.557s
ok      github.com/jvzantvoort/bundle/utils            (cached)
✅ ALL PASS
```

### Linter Status
```bash
$ golangci-lint run
✅ CLEAN (0 issues)
```

### Manual Test
```bash
$ bundle rename /tmp/test-bundle "New Title"
INFO[...] Title updated: Original Title → New Title
✅ WORKING
```

---

## Comparison

### Before
```bash
$ bundle rename ./bundle "New"
INFO[...] Title updated: New
```

**JSON Output:**
```json
{
  "status": "renamed",
  "path": "./bundle",
  "title": "New"
}
```

### After
```bash
$ bundle rename ./bundle "New"
INFO[...] Title updated: Old → New
```

**JSON Output:**
```json
{
  "status": "renamed",
  "path": "./bundle",
  "old_title": "Old",
  "new_title": "New",
  "title": "New"
}
```

**Improvements:**
- Shows old → new transition
- More informative JSON output
- Backward compatible (still includes "title" field)
- Better documentation

---

## Best Practices

### Error Handling
✅ Proper error wrapping with %w
✅ Descriptive error messages
✅ Clear failure modes

### Documentation
✅ Complete command help
✅ API documentation
✅ Usage examples
✅ Function comments

### Code Quality
✅ Reusable helper function
✅ Consistent patterns
✅ No code duplication
✅ Clean implementation

### User Experience
✅ Clear output messages
✅ Shows what changed
✅ JSON support for automation
✅ Verbose mode for debugging

---

## Use Cases

### 1. Correcting Typos
```bash
bundle rename ./photos "Vacation Photos"
```

### 2. Updating Documentation
```bash
bundle rename ./project "Project Archive 2024"
```

### 3. Automation/Scripting
```bash
#!/bin/bash
for bundle in /data/bundles/*; do
    date=$(date +%Y-%m-%d)
    title=$(basename "$bundle")
    bundle rename "$bundle" "$title - $date" --json
done
```

### 4. Batch Renaming
```bash
# Add year suffix to all bundles
find /bundles -name ".bundle" | while read dir; do
    bundle_dir=$(dirname "$dir")
    bundle rename "$bundle_dir" "$(basename $bundle_dir) 2024"
done
```

---

## Summary

The `bundle rename` command has been enhanced with:

✅ **Better documentation** - Complete help text with examples
✅ **Helper function** - Reusable UpdateTitle() in metadata package
✅ **Improved output** - Shows old → new title transition
✅ **Enhanced JSON** - Includes both old and new titles
✅ **Backward compatibility** - Existing tests still pass
✅ **Better debugging** - Verbose logging support

The command is production-ready and fully tested.

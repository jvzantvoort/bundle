# Configuration Debug Logging - Implementation

## Overview

Added comprehensive debug logging to show configuration sources, content, and values throughout the application.

---

## Changes Made

### 1. Enhanced config/main.go

**Added debug logging for:**
- Configuration file search paths
- Configuration file loading (success/failure)
- Configuration file source (which file was loaded)
- All configuration keys and values
- Log level detection from config

**Example Debug Output:**
```
time="2025-10-30T16:51:57+01:00" level=info msg="Configuration loaded from: /home/user/.config/bundle/config.yaml"
time="2025-10-30T16:51:57+01:00" level=debug msg="Log level set to debug from configuration"
```

### 2. Enhanced pool/main.go

**Added debug logging for:**

#### GetPool Function
- Pool name being requested
- Pool not found errors (with available pools list)
- Root directory from config
- Title from config
- Complete pool configuration summary

**Example Debug Output:**
```
time="2025-10-30 16:51:57" level=debug msg="GetPool called with name: testpool"
time="2025-10-30 16:51:57" level=debug msg="Pool 'testpool' root from config: /tmp/test-bundles"
time="2025-10-30 16:51:57" level=debug msg="Pool 'testpool' title from config: Test Bundle Pool"
time="2025-10-30 16:51:57" level=debug msg="Pool 'testpool' configuration loaded successfully:"
time="2025-10-30 16:51:57" level=debug msg="  Root:  /tmp/test-bundles"
time="2025-10-30 16:51:57" level=debug msg="  Title: Test Bundle Pool"
```

#### ListPools Function
- Number of pools found in configuration
- Pool names from configuration
- Loading status for each pool
- Total pools loaded successfully

**Example Debug Output:**
```
time="2025-10-30 16:52:10" level=debug msg="ListPools: found 2 pool(s) in configuration"
time="2025-10-30 16:52:10" level=debug msg="Pool names from configuration: map[backup:map[...] testpool:map[...]]"
time="2025-10-30 16:52:10" level=debug msg="Loading pool configuration for: testpool"
time="2025-10-30 16:52:10" level=debug msg="Successfully loaded 2 pool(s)"
```

#### Import Function
- Pool information (title and root)
- Source bundle path
- Operation mode (copy vs move)
- Bundle metadata (title, checksum, author)
- Destination path
- Copy progress
- Move/remove status

**Example Debug Output:**
```
time="2025-10-30 16:52:20" level=debug msg="Import called:"
time="2025-10-30 16:52:20" level=debug msg="  Pool:   Test Bundle Pool (/tmp/test-bundles)"
time="2025-10-30 16:52:20" level=debug msg="  Source: /path/to/bundle"
time="2025-10-30 16:52:20" level=debug msg="  Mode:   copy"
time="2025-10-30 16:52:20" level=debug msg="Loading bundle metadata from: /path/to/bundle"
time="2025-10-30 16:52:20" level=debug msg="Bundle metadata loaded:"
time="2025-10-30 16:52:20" level=debug msg="  Title:    My Bundle"
time="2025-10-30 16:52:20" level=debug msg="  Checksum: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
time="2025-10-30 16:52:20" level=debug msg="  Author:   john"
time="2025-10-30 16:52:20" level=debug msg="Destination path: /tmp/test-bundles/e3b0c44..."
time="2025-10-30 16:52:20" level=debug msg="Copying bundle from /path/to/bundle to /tmp/test-bundles/e3b0c44..."
time="2025-10-30 16:52:21" level=debug msg="Bundle copied successfully"
time="2025-10-30 16:52:21" level=debug msg="Import completed successfully"
```

#### ListBundles Function
- Pool information
- Pool directory existence
- Number of entries found
- Each bundle loading attempt
- Valid/skipped bundle counts

**Example Debug Output:**
```
time="2025-10-30 16:52:30" level=debug msg="ListBundles called for pool: Test Bundle Pool (/tmp/test-bundles)"
time="2025-10-30 16:52:30" level=debug msg="Scanning pool directory: /tmp/test-bundles"
time="2025-10-30 16:52:30" level=debug msg="Found 3 entries in pool directory"
time="2025-10-30 16:52:30" level=debug msg="Loading bundle metadata from: /tmp/test-bundles/e3b0c44..."
time="2025-10-30 16:52:30" level=debug msg="Bundle loaded: My Bundle (e3b0c44298fc...)"
time="2025-10-30 16:52:30" level=debug msg="Skipping non-directory entry: somefile.txt"
time="2025-10-30 16:52:30" level=debug msg="ListBundles completed:"
time="2025-10-30 16:52:30" level=debug msg="  Total entries:   3"
time="2025-10-30 16:52:30" level=debug msg="  Valid bundles:   2"
time="2025-10-30 16:52:30" level=debug msg="  Skipped entries: 1"
```

### 3. Updated cmd/bundle/root.go

**Added:**
- config.InitConfig() call in init()
- Import of config package

**Purpose:**
- Ensures configuration is loaded before any commands run
- Allows config-based log level to be set early

---

## Usage

### Enable Debug Logging

**Option 1: Via Configuration File**

`~/.config/bundle/config.yaml`:
```yaml
log_level: debug

pools:
  default:
    root: /mnt/bundles
    title: Default Bundle Pool
```

**Option 2: Via Command Line Flag**
```bash
bundle list_bundles --verbose
```

### What Gets Logged

With debug logging enabled, you'll see:

1. **Configuration Loading:**
   - Which config file was loaded (or if none found)
   - All configuration keys and values
   - Log level changes

2. **Pool Operations:**
   - Pool lookup (which pool, from where)
   - Pool configuration details
   - All pool values loaded

3. **Import Operations:**
   - Source and destination paths
   - Bundle metadata
   - Copy/move progress
   - Success/failure details

4. **List Operations:**
   - Pool being scanned
   - Number of entries
   - Each bundle loaded
   - Statistics

---

## Examples

### Viewing Configuration Loading

```bash
$ bundle list_bundles --pool backup
time="2025-10-30T16:52:00+01:00" level=info msg="Configuration loaded from: /home/user/.config/bundle/config.yaml"
time="2025-10-30T16:52:00+01:00" level=debug msg="Log level set to debug from configuration"
```

### Debugging Pool Not Found

```bash
$ bundle list_bundles --pool missing --verbose
time="2025-10-30 16:52:10" level=debug msg="GetPool called with name: missing"
time="2025-10-30 16:52:10" level=debug msg="Pool 'missing' not found in configuration"
time="2025-10-30 16:52:10" level=debug msg="Available pools: map[backup:map[root:/backup title:Backup] default:map[root:/mnt/bundles title:Default]]"
time="2025-10-30 16:52:10" level=error msg="Pool error: pool 'missing' not found in configuration"
```

### Debugging Import

```bash
$ bundle import ./my-bundle --pool testpool --verbose
time="2025-10-30 16:52:20" level=debug msg="Import called:"
time="2025-10-30 16:52:20" level=debug msg="  Pool:   Test Bundle Pool (/tmp/test-bundles)"
time="2025-10-30 16:52:20" level=debug msg="  Source: ./my-bundle"
time="2025-10-30 16:52:20" level=debug msg="  Mode:   copy"
time="2025-10-30 16:52:20" level=debug msg="Loading bundle metadata from: ./my-bundle"
time="2025-10-30 16:52:20" level=debug msg="Bundle metadata loaded:"
time="2025-10-30 16:52:20" level=debug msg="  Title:    My Bundle"
time="2025-10-30 16:52:20" level=debug msg="  Checksum: abc123..."
time="2025-10-30 16:52:20" level=debug msg="  Author:   john"
...
```

---

## Files Modified

1. **config/main.go** - Enhanced InitConfig() with debug logging
2. **pool/main.go** - Added debug logging to all functions
3. **cmd/bundle/root.go** - Added InitConfig() call and config import

---

## Benefits

### Troubleshooting
- Quickly see which config file is being used
- Verify pool configurations are correct
- Debug path resolution issues
- Understand why operations fail

### Development
- Trace execution flow
- Verify configuration values
- Debug pool logic
- Test configuration changes

### Operations
- Monitor bundle imports
- Verify pool access
- Audit configuration usage
- Debug production issues

---

## Configuration File Precedence

Config files are checked in this order:

1. `$HOME/.config/bundle/config.yaml` (user config)
2. `/etc/bundle/config.yaml` (system config)
3. `./config.yaml` (current directory)

The first file found is used. This is logged at INFO level when a config file is loaded.

---

## Log Levels

### info (default)
- Configuration file loaded message
- Operation results

### debug (--verbose or log_level: debug)
- All configuration details
- Pool lookup details
- Operation progress
- Detailed statistics

---

## Testing

```bash
# Create test config
mkdir -p ~/.config/bundle
cat > ~/.config/bundle/config.yaml << 'EOF'
pools:
  testpool:
    root: /tmp/test-bundles
    title: Test Pool

log_level: debug
EOF

# Test with debug logging
bundle list_bundles --pool testpool

# Expected output shows:
# - Config file location
# - Pool configuration details
# - Operation details
```

---

## Summary

All configuration loading and pool operations now have comprehensive debug logging that shows:
- **Where** configuration comes from (file path)
- **What** configuration is loaded (all keys/values)
- **How** pools are configured (root, title)
- **Why** operations succeed or fail (detailed steps)

This makes troubleshooting configuration issues much easier and provides visibility into the application's behavior.

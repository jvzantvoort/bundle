# Bundle Pools - Centralized Storage

## Overview

Bundle pools provide centralized storage for bundles, enabling:
- Content-addressable storage (bundles stored by checksum)
- Automatic deduplication
- Multiple storage locations (pools)
- Easy import/export workflow

## Configuration

### Config File Location

Pools are configured in `~/.config/bundle/config.yaml`:

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
```

### Pool Structure

Each pool has:
- **root**: Directory path where bundles are stored
- **title**: Human-readable name for the pool

Bundles are stored as: `{root}/{checksum}/`

## Commands

### import - Import Bundle to Pool

Import (copy or move) a bundle to a centralized pool.

#### Syntax

```bash
bundle import <path> [flags]
```

#### Flags

- `-p, --pool <name>` - Pool name (default: "default")
- `-m, --move` - Move bundle instead of copy
- `--json` - Output in JSON format

#### Examples

```bash
# Copy bundle to default pool
bundle import /path/to/bundle

# Move bundle to backup pool
bundle import /path/to/bundle --pool backup --move

# Import with JSON output
bundle import /path/to/bundle --json
```

#### JSON Output

```json
{
  "status": "imported",
  "operation": "copied",
  "pool": "default",
  "pool_root": "/mnt/bundles",
  "source": "/path/to/bundle"
}
```

### list_bundles - List Bundles in Pool

Display all bundles stored in a pool.

#### Syntax

```bash
bundle list_bundles [flags]
```

#### Flags

- `-p, --pool <name>` - Pool name (default: "default")
- `--json` - Output in JSON format

#### Examples

```bash
# List bundles in default pool
bundle list_bundles

# List bundles in backup pool
bundle list_bundles --pool backup

# List with JSON output
bundle list_bundles --json
```

#### Table Output

```
Pool: Default Bundle Pool (/mnt/bundles)

+----------------+------------------+----------+------------------+
|   CHECKSUM     |      TITLE       |  AUTHOR  |     CREATED      |
+----------------+------------------+----------+------------------+
| e3b0c44298fc...| Vacation Photos  | john     | 2024-01-15 10:30 |
| d14a028c2a3a...| Project Docs     | john     | 2024-01-16 14:20 |
+----------------+------------------+----------+------------------+

Total: 2 bundles
```

#### JSON Output

```json
{
  "pool": "default",
  "root": "/mnt/bundles",
  "bundles": [
    {
      "checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      "title": "Vacation Photos",
      "author": "john",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 1
}
```

## Workflow Examples

### Basic Import Workflow

```bash
# 1. Create a local bundle
bundle create ./photos --title "Vacation 2024"

# 2. Import to pool (copy)
bundle import ./photos

# 3. List bundles in pool
bundle list_bundles

# 4. Original bundle still exists locally
ls ./photos
```

### Archive Workflow (Move)

```bash
# 1. Create bundle
bundle create ./old-project --title "Old Project Archive"

# 2. Move to archive pool (removes local copy)
bundle import ./old-project --pool archive --move

# 3. Verify it's in archive
bundle list_bundles --pool archive

# 4. Local copy is gone
ls ./old-project  # No such file or directory
```

### Multi-Pool Strategy

```bash
# Work pool for active bundles
bundle import ./active-data --pool work

# Backup pool for redundancy
bundle import ./important-data --pool backup

# Archive pool for long-term storage
bundle import ./completed-project --pool archive --move

# List each pool
bundle list_bundles --pool work
bundle list_bundles --pool backup
bundle list_bundles --pool archive
```

## Use Cases

### 1. Centralized Team Storage

```yaml
# ~/.config/bundle/config.yaml
pools:
  team:
    root: /mnt/nfs/team-bundles
    title: Team Shared Storage
```

```bash
# Team members import bundles
bundle import ./my-work --pool team
bundle list_bundles --pool team
```

### 2. Tiered Storage

```yaml
pools:
  hot:
    root: /fast-ssd/bundles
    title: Hot Storage (SSD)
  
  cold:
    root: /slow-hdd/bundles
    title: Cold Storage (HDD)
  
  glacier:
    root: /archive/bundles
    title: Glacier Archive
```

```bash
# Active data on fast storage
bundle import ./current-project --pool hot

# Old data on slow storage
bundle import ./last-year --pool cold --move

# Ancient data on archive
bundle import ./2020-backup --pool glacier --move
```

### 3. Backup Strategy

```yaml
pools:
  local:
    root: /mnt/local/bundles
    title: Local Storage
  
  remote:
    root: /mnt/remote-backup/bundles
    title: Remote Backup
```

```bash
# Store locally
bundle import ./data --pool local

# Replicate to remote
bundle import ./data --pool remote

# Both pools now have the bundle
bundle list_bundles --pool local
bundle list_bundles --pool remote
```

## Go API Usage

### Import Bundle Programmatically

```go
package main

import (
    "log"
    
    "github.com/jvzantvoort/bundle/pool"
)

func main() {
    // Get pool configuration
    p, err := pool.GetPool("default")
    if err != nil {
        log.Fatal(err)
    }
    
    // Import bundle (copy)
    err = p.Import("/path/to/bundle", false)
    if err != nil {
        log.Fatal(err)
    }
    
    // List bundles
    bundles, err := p.ListBundles()
    if err != nil {
        log.Fatal(err)
    }
    
    for _, meta := range bundles {
        log.Printf("%s: %s\n", meta.BundleChecksum[:8], meta.Title)
    }
}
```

### List All Pools

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/jvzantvoort/bundle/pool"
)

func main() {
    pools, err := pool.ListPools()
    if err != nil {
        log.Fatal(err)
    }
    
    for name, p := range pools {
        fmt.Printf("%s: %s (%s)\n", name, p.Title, p.Root)
    }
}
```

## Storage Format

Bundles in pools use content-addressable storage:

```
/mnt/bundles/
├── e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855/
│   ├── .bundle/
│   │   ├── META.json
│   │   ├── STATE.json
│   │   ├── TAGS.txt
│   │   └── SHA256SUM.txt
│   ├── file1.txt
│   └── file2.txt
└── d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f/
    ├── .bundle/
    └── ...
```

### Benefits

1. **Deduplication**: Same content = same checksum = single copy
2. **Verification**: Bundle path is its checksum
3. **Discovery**: Can verify any bundle by recomputing checksum
4. **Integrity**: Path mismatch indicates corruption

## Best Practices

### 1. Pool Naming

- Use descriptive names: `production`, `staging`, `archive`
- Keep names lowercase
- Avoid special characters

### 2. Storage Layout

- Separate pools by:
  - Performance tier (SSD vs HDD)
  - Lifecycle (active vs archive)
  - Team/project
  - Backup vs primary

### 3. Workflow

- **Copy** for: Backup, replication, sharing
- **Move** for: Archival, cleanup, one-way transfer

### 4. Verification

```bash
# Regularly verify pool bundles
for bundle in /mnt/bundles/*; do
    bundle verify "$bundle"
done
```

## Troubleshooting

### Pool Not Found

```bash
$ bundle import ./bundle --pool missing
Error: pool 'missing' not found in configuration
```

**Solution**: Check `~/.config/bundle/config.yaml`

### Permission Denied

```bash
$ bundle import ./bundle
Error: failed to create pool directory: permission denied
```

**Solution**: Ensure pool root is writable:
```bash
sudo chown $USER /mnt/bundles
# or
sudo chmod 755 /mnt/bundles
```

### Bundle Already Exists

```bash
$ bundle import ./bundle
Error: bundle already exists in pool: e3b0c44...
```

**Solution**: Bundle with same checksum already imported (deduplication working!)

## Migration

### From Local to Pool

```bash
# Find all local bundles
find ~/bundles -name ".bundle" -type d | while read dir; do
    bundle=$(dirname "$dir")
    echo "Importing $bundle"
    bundle import "$bundle" --pool archive --move
done
```

### Between Pools

```bash
# Get bundle checksum
checksum=$(bundle info ./bundle --json | jq -r '.checksum')

# Copy from one pool to another
cp -r "/pool1/$checksum" "/pool2/$checksum"
```

## See Also

- [Bundle Creation](../README.md#creating-bundles)
- [Configuration](../config.yaml.example)
- [API Documentation](../README.md#api-documentation)

# GitHub Copilot Instructions for Bundle Library

## Project Overview

Bundle Library is a **content-addressable digital asset management (DAM) system** written in Go. It organizes files into immutable bundles identified by SHA256 checksums, enabling data integrity verification, deduplication, and centralized storage management.

**Key Concepts:**
- **Bundle**: An immutable directory of files with metadata in `.bundle/` subdirectory
- **Checksum**: SHA256-based content addressing for integrity verification
- **Pool**: Centralized storage location for managing multiple bundles
- **Metadata**: Title, tags, timestamps stored as YAML
- **State**: Tracks file information (size, timestamps, permissions)

## Code Style & Conventions

### Go Standards

1. **Package Documentation**: Every package must have comprehensive godoc comments
   ```go
   // Package bundle provides high-level operations for managing content-addressable
   // file bundles with SHA256-based integrity verification.
   //
   // Example usage:
   //
   //	b, err := bundle.Create("/path/to/files", "My Photos")
   //	if err != nil {
   //	    log.Fatal(err)
   //	}
   ```

2. **Function Documentation**: Include purpose, parameters, return values, and examples
   ```go
   // ComputeFileSHA256 computes the SHA256 checksum of a file using streaming I/O
   // to handle files of any size efficiently.
   //
   // Parameters:
   //   - path: Absolute or relative path to the file
   //
   // Returns:
   //   - Lowercase hexadecimal SHA256 checksum (64 characters)
   //   - Error if file cannot be read or hashed
   //
   // Example:
   //
   //	checksum, err := ComputeFileSHA256("/path/to/file.txt")
   //	if err != nil {
   //	    log.Fatal(err)
   //	}
   ```

3. **Error Handling**: Always use `fmt.Errorf` with context
   ```go
   if err != nil {
       return fmt.Errorf("failed to compute checksum for %s: %w", path, err)
   }
   ```

4. **Logging**: Use structured logging with `log.Debugf` for configuration tracing
   ```go
   log.Debugf("Configuration loaded from: %s", configPath)
   log.Debugf("Pool configuration: %+v", poolConfig)
   ```

### Project-Specific Patterns

1. **Cobra Commands**: All CLI commands use cobra with consistent structure
   ```go
   var exampleCmd = &cobra.Command{
       Use:   "example <arg>",
       Short: "Brief description",
       Long:  `Detailed description with examples`,
       Args:  cobra.ExactArgs(1),
       Run:   runExample,
   }
   ```

2. **Bundle Operations**: Always verify bundle validity before operations
   ```go
   b, err := bundle.Load(bundlePath)
   if err != nil {
       return fmt.Errorf("invalid bundle: %w", err)
   }
   ```

3. **Checksum Files**: Use `checksum.ChecksumFile` for integrity operations
   ```go
   files := &checksum.ChecksumFile{}
   if err := files.Compute(bundlePath); err != nil {
       return err
   }
   ```

4. **Pool Management**: Use `pool.Pool` for centralized storage operations
   ```go
   p, err := pool.NewPool(poolConfig)
   if err != nil {
       return err
   }
   ```

## Architecture Guidelines

### Package Organization

- **`bundle/`**: Core bundle operations (Create, Load, Verify)
- **`checksum/`**: SHA256 computation and verification
- **`metadata/`**: YAML-based metadata management (title, tags, dates)
- **`state/`**: File state tracking (size, timestamps, permissions)
- **`pool/`**: Centralized storage management (import, list, organize)
- **`cmd/`**: CLI command implementations using Cobra
- **`config/`**: Configuration file parsing and management
- **`utils/`**: Shared utilities (path handling, file operations)
- **`scanner/`**: Directory scanning and file discovery
- **`lock/`**: File locking for concurrent operations
- **`messages/`**: Consistent output formatting (table, JSON)
- **`tag/`**: Tag management and validation

### Dependency Rules

1. **Library-first design**: Packages should be usable independently
2. **No circular dependencies**: Enforce strict dependency hierarchy
3. **Minimal external dependencies**: Prefer standard library when possible
4. **Vendor dependencies**: Use `go mod vendor` for reproducibility

### File Structure Patterns

```
bundle-directory/
├── .bundle/
│   ├── metadata.yaml      # Title, tags, timestamps
│   ├── SHA256SUM.txt      # File checksums (sorted)
│   ├── state.yaml         # File metadata (size, mode, times)
│   └── lock               # Lock file for concurrent operations
└── [files...]             # Actual bundle content
```

## Testing Standards

1. **Test Coverage**: Aim for >80% coverage on core packages
2. **Use `t.TempDir()`**: For all file system operations in tests
3. **Table-Driven Tests**: For multiple test cases
4. **Integration Tests**: Test complete workflows end-to-end
5. **Error Cases**: Always test error paths and edge cases

Example test structure:
```go
func TestBundleCreate(t *testing.T) {
    tmpDir := t.TempDir()
    
    tests := []struct {
        name    string
        files   map[string]string
        title   string
        wantErr bool
    }{
        {"valid bundle", map[string]string{"a.txt": "hello"}, "Test", false},
        {"empty directory", map[string]string{}, "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Common Operations

### Creating a Bundle
```go
b, err := bundle.Create("/path/to/files", "Bundle Title")
if err != nil {
    return fmt.Errorf("create failed: %w", err)
}
```

### Loading a Bundle
```go
b, err := bundle.Load("/path/to/bundle")
if err != nil {
    return fmt.Errorf("load failed: %w", err)
}
```

### Verifying Integrity
```go
corrupted, err := b.Verify()
if err != nil {
    return fmt.Errorf("verify failed: %w", err)
}
if len(corrupted) > 0 {
    // Handle corrupted files
}
```

### Working with Pools
```go
// Import bundle to pool
err := poolInstance.Import("/path/to/bundle", pool.ImportOptions{
    Copy: true,
    VerifyAfter: true,
})

// List bundles in pool
bundles, err := poolInstance.List()
for _, b := range bundles {
    fmt.Printf("%s: %s\n", b.Checksum, b.Metadata.Title)
}
```

### Updating Metadata
```go
b.Metadata.Title = "New Title"
if err := b.Metadata.Save(bundlePath); err != nil {
    return err
}
```

## Configuration

Configuration is read from `~/.config/bundle/config.yaml`:

```yaml
pools:
  default:
    root: "/data/bundles"
    title: "Default Storage Pool"
  backup:
    root: "/backup/bundles"
    title: "Backup Pool"
```

Always log configuration loading:
```go
log.Debugf("Loading config from: %s", configPath)
log.Debugf("Pools configured: %+v", config.Pools)
```

## CLI Command Patterns

### Command Registration
```go
func init() {
    rootCmd.AddCommand(exampleCmd)
    exampleCmd.Flags().StringP("format", "f", "table", "Output format (table|json)")
}
```

### Output Formatting
```go
if format == "json" {
    messages.PrintJSON(result)
} else {
    messages.PrintTable(headers, rows)
}
```

## Performance Considerations

1. **Streaming I/O**: Use `io.Copy` for large files, never read entire file into memory
2. **Concurrent Operations**: Use goroutines for independent file operations
3. **Deterministic Sorting**: Always sort checksums before computing bundle checksum
4. **File Locking**: Use `lock.Lock()` to prevent concurrent modifications
5. **Lazy Loading**: Load metadata only when needed

## Code Quality

### Linting Rules

The project uses `golangci-lint`. Common issues to avoid:

1. **Exported symbols**: Must have documentation comments
2. **Error wrapping**: Use `%w` verb with `fmt.Errorf`
3. **Context usage**: Pass `context.Context` for cancellable operations
4. **Unused code**: Remove commented-out code and unused variables
5. **Magic numbers**: Define constants for repeated values

### Pre-commit Checks
```bash
go fmt ./...
golangci-lint run
go test ./...
go build ./cmd/bundle
```

## Documentation Standards

### Required Documentation

1. **Every package**: Package-level godoc with examples
2. **Every exported function**: Full documentation with examples
3. **README.md**: Quick start and basic usage
4. **docs/**: Comprehensive guides organized by audience
   - User guides (EXAMPLES.md, POOLS.md)
   - Developer guides (API.md, POOLS_IMPLEMENTATION.md)
   - Project docs (SPEC_UPDATES.md, IMPROVEMENTS.md)

### Documentation Organization
```
docs/
├── README.md              # Documentation index
├── NAVIGATION.md          # Navigation guide
├── API.md                 # Complete API reference
├── EXAMPLES.md            # Usage examples
├── POOLS.md               # Pool management guide
└── [other guides]
```

## AI-Assisted Development Notes

This project was developed with significant AI assistance. Key learnings:

1. **Start with specs**: Define specifications before implementation
2. **Incremental development**: Build features one at a time
3. **Test-driven**: Write tests alongside implementation
4. **Documentation-first**: Document as you code, not after
5. **Code review**: Always review AI-generated code for correctness
6. **Performance**: AI may choose simple solutions; optimize critical paths
7. **Error handling**: AI sometimes misses edge cases; add defensive checks

## References

- **Main README**: [README.md](../README.md)
- **API Documentation**: [docs/API.md](../docs/API.md)
- **Examples**: [docs/EXAMPLES.md](../docs/EXAMPLES.md)
- **Specifications**: [specs/001-bundle-core/](../specs/001-bundle-core/)
- **Pool Implementation**: [docs/POOLS_IMPLEMENTATION.md](../docs/POOLS_IMPLEMENTATION.md)

## Quick Reference

### File Locations
- Bundle metadata: `.bundle/metadata.yaml`
- Bundle checksums: `.bundle/SHA256SUM.txt`
- Bundle state: `.bundle/state.yaml`
- Config file: `~/.config/bundle/config.yaml`
- Specs: `specs/001-bundle-core/spec.md`
- Documentation: `docs/`

### Common Commands
```bash
# Create bundle
bundle create /path/to/files "Title"

# Verify bundle
bundle verify /path/to/bundle

# Import to pool
bundle import /path/to/bundle pool-name

# List bundles
bundle list-bundles pool-name

# Update title
bundle rename /path/to/bundle "New Title"

# Add tags
bundle tag-add /path/to/bundle tag1 tag2
```

### Environment Variables
```bash
BUNDLE_CONFIG_PATH   # Override config file location
BUNDLE_LOG_LEVEL     # Set logging level (debug, info, warn, error)
BUNDLE_NO_COLOR      # Disable colored output
```

---

**Remember**: This is a library-first design. Every feature should work as both a Go package and a CLI command. Always maintain data integrity and immutability of bundles.

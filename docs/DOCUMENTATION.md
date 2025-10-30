# Bundle Library - Documentation Guide

Quick reference for accessing and using the Bundle Library documentation.

## Quick Links

- **[README.md](README.md)** - Project overview and API reference
- **[EXAMPLES.md](EXAMPLES.md)** - Comprehensive code examples
- **[Quickstart](specs/001-bundle-core/quickstart.md)** - Developer setup
- **[CLI Commands](specs/001-bundle-core/contracts/cli-commands.md)** - CLI reference

## Viewing Documentation

### Online (pkg.go.dev)

Once published, documentation will be available at:
```
https://pkg.go.dev/github.com/jvzantvoort/bundle
```

### Local godoc

Start local documentation server:
```bash
godoc -http=:6060
```

Then visit:
- http://localhost:6060/pkg/github.com/jvzantvoort/bundle/
- http://localhost:6060/pkg/github.com/jvzantvoort/bundle/bundle/
- http://localhost:6060/pkg/github.com/jvzantvoort/bundle/checksum/
- http://localhost:6060/pkg/github.com/jvzantvoort/bundle/metadata/
- http://localhost:6060/pkg/github.com/jvzantvoort/bundle/state/
- http://localhost:6060/pkg/github.com/jvzantvoort/bundle/tag/

### Command Line

```bash
# View package documentation
go doc github.com/jvzantvoort/bundle/bundle
go doc github.com/jvzantvoort/bundle/checksum
go doc github.com/jvzantvoort/bundle/metadata

# View specific function
go doc bundle.Create
go doc checksum.ComputeBundleChecksum
go doc tag.Tags.Add

# View all documentation for a package
go doc -all bundle
```

### IDE Integration

Most Go IDEs (VSCode with Go extension, GoLand, etc.) will automatically show
inline documentation from the godoc comments when you hover over functions
or use autocomplete.

## Documentation Structure

### Package-Level Documentation

Each package has comprehensive documentation at the top of its main file:
- Purpose and overview
- Key concepts
- Usage examples
- Related packages

### Function/Method Documentation

Each exported function includes:
- What it does
- Usage examples
- Parameter descriptions
- Return value descriptions
- Error conditions

### Type Documentation

Each exported type includes:
- Purpose and usage
- Field descriptions
- Example JSON (where applicable)
- Usage examples

## Example Categories

### EXAMPLES.md Contains

1. **Go Package Examples** (50+ examples)
   - Bundle operations
   - Checksum computation
   - Metadata management
   - State tracking
   - Tag management
   - Locking
   - Directory scanning

2. **CLI Examples**
   - Creating bundles
   - Verifying integrity
   - Managing tags
   - Getting information
   - JSON output

3. **Advanced Usage**
   - Shell scripting
   - CI/CD integration
   - Error handling
   - Automation

## Common Tasks

### Creating a Bundle (Go)

```go
import "github.com/jvzantvoort/bundle/bundle"

b, err := bundle.Create("/path/to/files", "My Bundle")
```

See: [EXAMPLES.md - Bundle Operations](#bundle-operations)

### Creating a Bundle (CLI)

```bash
bundle create /path/to/files --title "My Bundle"
```

See: [EXAMPLES.md - Creating Bundles](#creating-bundles)

### Verifying Integrity (Go)

```go
import "github.com/jvzantvoort/bundle/bundle"

verified, corrupted, err := bundle.Verify("/path/to/bundle")
```

See: [EXAMPLES.md - Bundle Operations](#bundle-operations)

### Verifying Integrity (CLI)

```bash
bundle verify /path/to/bundle
```

See: [EXAMPLES.md - Verifying Bundles](#verifying-bundles)

## API Reference

Complete API documentation is available in:
- [README.md - API Documentation](README.md#api-documentation)

This includes:
- All package APIs
- Type definitions
- JSON schemas
- CLI command reference
- Exit codes

## Contributing

When adding new code, please:
1. Add godoc comments to all exported types and functions
2. Include usage examples in the comments
3. Document parameters and return values
4. Add examples to EXAMPLES.md if applicable
5. Update README.md API section if adding new packages

## Documentation Standards

Follow these standards when writing documentation:

### Package Documentation
```go
// Package mypackage provides functionality for X.
//
// It does Y by using Z approach. This ensures...
//
// Example usage:
//
//     result := mypackage.DoSomething()
//     fmt.Println(result)
package mypackage
```

### Function Documentation
```go
// DoSomething performs action X on Y.
//
// It does this by... Returns error if...
//
// Example:
//
//     result, err := DoSomething(input)
//     if err != nil {
//         log.Fatal(err)
//     }
//
// Parameters:
//   - input: description
//
// Returns:
//   - type: description
//   - error: when X happens
func DoSomething(input string) (Result, error) {
```

### Type Documentation
```go
// MyType represents a thing that does X.
//
// It contains Y and Z fields which...
//
// Example:
//
//     t := &MyType{Field: "value"}
//     t.DoSomething()
type MyType struct {
    Field string // Description
}
```

## Getting Help

- Check [EXAMPLES.md](EXAMPLES.md) for code examples
- Check [README.md](README.md) for API reference
- Use `go doc <package>.<function>` for quick reference
- Start `godoc -http=:6060` for browseable documentation
- Read package-level documentation for overview

## Updates

Documentation is kept up-to-date with the code. When:
- Adding new functions → Add godoc comments
- Changing APIs → Update function documentation
- Adding features → Add examples to EXAMPLES.md
- Changing behavior → Update README.md

## Version

This documentation is current as of the addition of comprehensive documentation
in October 2024. See git history for updates.

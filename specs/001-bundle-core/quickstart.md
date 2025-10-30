# Quickstart Guide: Bundle Library Development

**Feature**: 001-bundle-core  
**Date**: 2025-10-30  
**Audience**: Developers building or contributing to the Bundle Library

## Overview

This guide helps you quickly set up, build, test, and understand the Bundle Library codebase. The project implements a content-addressable file bundle system in Go with CLI interface.

---

## Prerequisites

### Required
- **Go 1.21+**: [Download](https://go.dev/dl/)
- **Git**: For version control
- **Unix-like environment**: Linux, macOS, or WSL2 on Windows

### Verify Installation

```bash
$ go version
go version go1.21.0 linux/amd64

$ git --version
git version 2.40.0
```

---

## Getting Started

### 1. Clone Repository

```bash
$ git clone https://github.com/jvzantvoort/bundle.git
$ cd bundle
$ git checkout 001-bundle-core  # Development branch
```

### 2. Initialize Go Module

```bash
$ go mod init github.com/jvzantvoort/bundle
$ go mod tidy
```

### 3. Install Dependencies

```bash
$ go get github.com/spf13/cobra@latest
$ go get github.com/spf13/viper@latest
$ go get github.com/sirupsen/logrus@latest
$ go get github.com/olekukonko/tablewriter@latest
$ go get github.com/fatih/color@latest
```

---

## Building

### Build CLI Binary

```bash
# Build main binary
$ go build -o bundle ./cmd/bundle

# Verify build
$ ./bundle --version
bundle version 0.1.0
```

### Build All Commands

```bash
# Build and install to $GOPATH/bin
$ go install ./cmd/...

# Verify installation
$ which bundle
/home/user/go/bin/bundle
```

---

## Running Tests

### Run All Tests

```bash
# Run all unit and integration tests
$ go test ./...
ok      github.com/jvzantvoort/bundle/checksum      0.123s
ok      github.com/jvzantvoort/bundle/metadata      0.089s
ok      github.com/jvzantvoort/bundle/state         0.045s
ok      github.com/jvzantvoort/bundle/tag           0.067s
ok      github.com/jvzantvoort/bundle/scanner       0.102s
ok      github.com/jvzantvoort/bundle/lock          0.078s
ok      github.com/jvzantvoort/bundle/bundle        0.234s
ok      github.com/jvzantvoort/bundle/tests/integration  1.456s
ok      github.com/jvzantvoort/bundle/tests/contract     0.567s
```

### Run Specific Test Suite

```bash
# Unit tests only
$ go test ./checksum ./metadata ./state ./tag ./scanner ./lock

# Integration tests
$ go test ./tests/integration -v

# Contract tests
$ go test ./tests/contract -v
```

### Run with Coverage

```bash
$ go test -cover ./...
ok      github.com/jvzantvoort/bundle/checksum      0.123s  coverage: 92.3% of statements
ok      github.com/jvzantvoort/bundle/metadata      0.089s  coverage: 94.1% of statements
...

# Generate HTML coverage report
$ go test -coverprofile=coverage.out ./...
$ go tool cover -html=coverage.out -o coverage.html
$ open coverage.html  # View in browser
```

---

## Example Workflow

### Create a Bundle

```bash
# Create test directory
$ mkdir -p ~/test-bundle
$ echo "Hello World" > ~/test-bundle/file1.txt
$ echo "Goodbye World" > ~/test-bundle/file2.txt

# Create bundle
$ bundle create ~/test-bundle --title "My Test Bundle"
Bundle Created
--------------
Path:     /home/user/test-bundle
Checksum: 7a8f3b2c1d...
Files:    2
Size:     26 bytes
Title:    My Test Bundle

# Verify bundle structure
$ ls -la ~/test-bundle/.bundle/
drwxr-xr-x  2 user user 4096 Oct 30 10:48 .
drwxr-xr-x  3 user user 4096 Oct 30 10:48 ..
-rw-r--r--  1 user user  123 Oct 30 10:48 META.json
-rw-r--r--  1 user user  187 Oct 30 10:48 SHA256SUM.txt
-rw-r--r--  1 user user   98 Oct 30 10:48 STATE.json
-rw-r--r--  1 user user    0 Oct 30 10:48 TAGS.txt
```

### Verify Bundle Integrity

```bash
$ bundle verify ~/test-bundle
Bundle Integrity: VALID
---------------------
Files Checked: 2
Last Verified: 2025-10-30 10:48:42

# Modify a file to test corruption detection
$ echo "Modified" > ~/test-bundle/file1.txt

$ bundle verify ~/test-bundle
Bundle Integrity: INVALID
------------------------
Files Checked: 2
Corrupted Files:
- file1.txt (expected: 7a8f3b2c..., found: deadbeef...)
```

### Add Tags

```bash
$ bundle tag add ~/test-bundle testing demo example
Tags Added
----------
Added 3 tags: testing, demo, example
Total tags: 3

$ bundle tag list ~/test-bundle
Tags
----
demo
example
testing

Total: 3 tags
```

### Inspect Bundle

```bash
$ bundle info ~/test-bundle --json
{
  "path": "/home/user/test-bundle",
  "title": "My Test Bundle",
  "checksum": "7a8f3b2c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a",
  "files": 2,
  "size_bytes": 26,
  "created_at": "2025-10-30T10:48:42Z",
  "author": "user",
  "last_verified": "2025-10-30T10:48:42Z",
  "verified": true,
  "tags": ["demo", "example", "testing"]
}
```

---

## Project Structure

### Key Directories

```text
bundle/
├── cmd/                  # CLI entry points (Cobra commands)
│   ├── bundle/          # Root command
│   ├── create/          # bundle create
│   ├── verify/          # bundle verify
│   ├── info/            # bundle info
│   ├── list/            # bundle list
│   └── tag/             # bundle tag
├── checksum/            # SHA256 computation library
├── metadata/            # META.json handling
├── state/               # STATE.json handling
├── tag/                 # TAGS.txt handling
├── scanner/             # Directory traversal
├── lock/                # Concurrency control
├── bundle/              # High-level bundle operations
├── config/              # Viper configuration
├── utils/               # Shared utilities
├── messages/            # Help text (go:embed)
└── tests/               # Integration and contract tests
```

### Important Files

- `go.mod` - Go module dependencies
- `specs/001-bundle-core/spec.md` - Feature specification
- `specs/001-bundle-core/plan.md` - Implementation plan
- `specs/001-bundle-core/data-model.md` - Data structures
- `specs/001-bundle-core/contracts/cli-commands.md` - CLI specifications

---

## Development Tips

### Code Style

Follow Go conventions:
- `gofmt` for formatting (run before commit)
- `go vet` for static analysis
- Exported functions must have doc comments
- Use table-driven tests for multiple test cases

```bash
# Format all code
$ gofmt -s -w .

# Run static analysis
$ go vet ./...

# Run linter (if installed)
$ golangci-lint run
```

### Logging

Use logrus for structured logging:

```go
import log "github.com/sirupsen/logrus"

// Set log level based on --verbose flag
if verbose {
    log.SetLevel(log.DebugLevel)
} else {
    log.SetLevel(log.InfoLevel)
}

// Use structured logging
log.WithFields(log.Fields{
    "path": bundlePath,
    "files": fileCount,
}).Info("Bundle created")
```

### Testing Best Practices

```go
func TestComputeChecksum(t *testing.T) {
    // Use t.TempDir() for temporary test directories
    tmpDir := t.TempDir()
    
    // Create test files
    testFile := filepath.Join(tmpDir, "test.txt")
    err := os.WriteFile(testFile, []byte("content"), 0644)
    require.NoError(t, err)
    
    // Test the function
    checksum, err := ComputeFileSHA256(testFile)
    require.NoError(t, err)
    assert.Len(t, checksum, 64) // SHA256 is 64 hex chars
}
```

---

## Common Tasks

### Add a New CLI Command

1. Create command file: `cmd/mycommand/main.go`
2. Implement Cobra command structure
3. Call library functions (dependency injection)
4. Add help messages to `messages/{long,usage,short}/mycommand`
5. Write contract tests in `tests/contract/`

### Add a New Library Component

1. Create package directory: `mycomponent/`
2. Implement functionality in `mycomponent/main.go`
3. Write unit tests in `mycomponent/main_test.go`
4. Document exported functions with Go doc comments
5. Update integration tests if needed

---

## Troubleshooting

### Build Errors

```bash
# Clear Go module cache
$ go clean -modcache
$ go mod tidy

# Re-download dependencies
$ go get -u ./...
```

### Test Failures

```bash
# Run tests with verbose output
$ go test -v ./...

# Run specific test
$ go test -v -run TestComputeChecksum ./checksum
```

### Bundle Lock Issues

```bash
# Manually unlock bundle (if process crashed)
$ rm ~/test-bundle/.bundle/.lock
```

---

## Next Steps

1. **Read the spec**: [spec.md](./spec.md) - Understand requirements
2. **Review data model**: [data-model.md](./data-model.md) - Understand entities
3. **Study contracts**: [contracts/cli-commands.md](./contracts/cli-commands.md) - CLI behavior
4. **Explore code**: Start with `bundle/main.go` for high-level operations
5. **Run tests**: Ensure everything works before making changes

---

## Resources

- **Go Documentation**: https://go.dev/doc/
- **Cobra Guide**: https://github.com/spf13/cobra/blob/main/user_guide.md
- **Logrus Examples**: https://github.com/sirupsen/logrus#example
- **Project Constitution**: `bundle/.specify/memory/constitution.md`

---

## Getting Help

- **Spec Questions**: See [spec.md](./spec.md) for requirements
- **Implementation Questions**: See [plan.md](./plan.md) for design decisions
- **Technical Decisions**: See [research.md](./research.md) for rationale

---

**Happy coding! 🚀**

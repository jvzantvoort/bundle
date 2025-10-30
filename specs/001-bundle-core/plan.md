# Implementation Plan: Bundle Library Core Implementation

**Branch**: `001-bundle-core` | **Date**: 2025-10-30 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/001-bundle-core/spec.md`

**Note**: This plan follows the constitution v1.0.0 principles and user-specified Go project structure preferences.

## Summary

Build a Go library and CLI for content-addressable, immutable file bundles with SHA256-based integrity verification. Implements Library-First Architecture (Principle II) with independent library components, then exposes via CLI-First Interface (Principle III) using Cobra. Ensures Content Integrity (Principle I) through deterministic checksum computation and Observability (Principle V) via structured logrus logging.

## Technical Context

**Language/Version**: Go 1.21+  
**Primary Dependencies**: 
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/sirupsen/logrus` - Structured logging
- `github.com/olekukonko/tablewriter` - Human-readable table output
- `github.com/fatih/color` - Terminal colorization

**Storage**: Local filesystem (POSIX-compliant), `.bundle/` subdirectory for metadata (META.json, SHA256SUM.txt, STATE.json, TAGS.txt)  
**Testing**: Go standard `testing` package with `go test ./...`, integration tests using temporary directories  
**Target Platform**: Linux/macOS/Windows (cross-platform via Go standard library)  
**Project Type**: Single project (library + CLI)  
**Performance Goals**: 
- Create bundle from 100 files (1GB) in <30 seconds
- Streaming checksum computation for files up to 10GB
- Support 1000+ files per bundle without degradation

**Constraints**: 
- Deterministic checksum computation (order-independent)
- Streaming I/O to avoid memory exhaustion on large files
- Lock-based concurrency control for write operations
- Exit codes: 0 (success), 1 (user error), 2 (system error)

**Scale/Scope**: 
- MVP: 7 CLI commands (create, verify, info, list, tag add/remove/list)
- 6 library components (checksum, metadata, state, tag, scanner, lock managers)
- 15 functional requirements from spec
- 3 prioritized user stories (P1-P3)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

### ✅ I. Content Integrity (NON-NEGOTIABLE)
- SHA256 checksums for all files (FR-001, FR-002)
- Deterministic bundle checksum from sorted file hashes (FR-002)
- Integrity verification before operations (FR-013, FR-014)
- **Status**: PASS - Core requirement, design enforces immutability

### ✅ II. Library-First Architecture
- Standalone library components (checksum, metadata, state, tag, scanner, lock)
- CLI depends on library (dependency injection), not vice versa
- Each component independently testable with Go doc comments
- **Status**: PASS - Project structure separates library from CLI

### ✅ III. CLI-First Interface
- Cobra CLI with text-based I/O (stdin/args → stdout/stderr)
- JSON flag support for all commands (FR-008)
- Tablewriter for human-readable output (FR-015)
- Exit codes: 0/1/2 (FR-009)
- **Status**: PASS - CLI wraps library, follows contract

### ✅ IV. Immutability & Determinism
- Lexicographic sort of checksums ensures determinism (FR-002)
- Metadata changes don't affect bundle checksum
- Physical names derived from content (FR-002)
- **Status**: PASS - Deterministic checksum algorithm specified

### ✅ V. Observability & Debugging
- Logrus structured logging with ERROR/WARN/INFO/DEBUG levels (FR-010)
- Stdout for data, stderr for logs (FR-011)
- **Status**: PASS - Logging architecture defined

### Technology Standards Compliance
- ✅ Go 1.21+ with standard library
- ✅ Required libraries: Cobra, Viper, Logrus, Tablewriter, Color
- ✅ Code structure follows user preference (cmd/, config/, functional areas, messages/, utils/)
- ✅ Go modules with semantic versioning
- ✅ Standard `go build`, `go test`, `go vet` workflow

**Overall**: ✅ ALL GATES PASSED - No violations, no complexity justification needed

## Project Structure

### Documentation (this feature)

```text
specs/001-bundle-core/
├── spec.md               # Feature specification (completed)
├── plan.md               # This file (current)
├── research.md           # Phase 0 output (next step)
├── data-model.md         # Phase 1 output
├── quickstart.md         # Phase 1 output
├── contracts/            # Phase 1 output (CLI contracts)
│   └── cli-commands.md   # Command signatures and examples
└── tasks.md              # Phase 2 output (/speckit.tasks - not created yet)
```

### Source Code (repository root)

```text
# Go module root
go.mod
go.sum

# CLI entry points (Cobra commands)
cmd/
├── bundle/
│   └── main.go           # Root command and version
├── create/
│   └── main.go           # bundle create
├── verify/
│   └── main.go           # bundle verify
├── info/
│   └── main.go           # bundle info
├── list/
│   └── main.go           # bundle list
└── tag/
    └── main.go           # bundle tag (with add/remove/list subcommands)

# Configuration handling (Viper)
config/
└── main.go               # Config file loader, defaults, environment variables

# Library components (core functionality)
checksum/
├── main.go               # SHA256 computation, deterministic bundle checksum
├── main_test.go          # Unit tests for checksum logic
└── stream.go             # Streaming checksum for large files

metadata/
├── main.go               # META.json read/write operations
├── main_test.go          # Unit tests for metadata persistence
└── types.go              # Metadata struct definitions

state/
├── main.go               # STATE.json read/write operations
└── main_test.go          # Unit tests for state management

tag/
├── main.go               # TAGS.txt operations (add, remove, list)
└── main_test.go          # Unit tests for tag management

scanner/
├── main.go               # Directory traversal, file discovery (excludes .bundle/)
└── main_test.go          # Unit tests for file scanning

lock/
├── main.go               # Lock file management for concurrent operations
└── main_test.go          # Unit tests for lock behavior

bundle/
├── main.go               # High-level bundle operations (Create, Verify, Info)
└── main_test.go          # Integration tests for bundle operations

# Help messages (go:embed)
messages/
├── long/
│   ├── bundle            # Long help for root command
│   ├── create            # Long help for create command
│   ├── verify            # Long help for verify command
│   ├── info              # Long help for info command
│   ├── list              # Long help for list command
│   └── tag               # Long help for tag command
├── usage/
│   ├── bundle
│   ├── create
│   ├── verify
│   ├── info
│   ├── list
│   └── tag
└── short/
    ├── bundle
    ├── create
    ├── verify
    ├── info
    ├── list
    └── tag

# Shared utilities
utils/
├── filepath.go           # Path normalization, .bundle/ exclusion logic
├── exit.go               # Exit code helpers (0/1/2)
└── output.go             # Stdout/stderr helpers, JSON vs table output

# Tests
tests/
├── integration/
│   ├── create_test.go    # End-to-end bundle creation tests
│   ├── verify_test.go    # End-to-end integrity verification tests
│   ├── metadata_test.go  # End-to-end metadata/tag tests
│   └── testdata/         # Test fixtures (sample files for bundles)
└── contract/
    └── cli_test.go       # CLI contract tests (exit codes, JSON output)
```

**Structure Decision**: Single project layout following user-specified preferences from COPILOT.md. Library components in functional directories (`checksum/`, `metadata/`, etc.) provide standalone, testable functionality. CLI commands in `cmd/` use dependency injection to call library functions. No web/mobile components needed for this feature.

## Complexity Tracking

> **Not applicable** - All Constitution Check gates passed without violations.

---

## Phase 0: Research & Design Decisions

**Status**: PENDING - Execute research tasks below

### Research Tasks

No unresolved NEEDS CLARIFICATION markers from Technical Context. All technology choices specified by user (Go, Cobra, Viper, Logrus, Tablewriter). Proceed to document design decisions:

#### Task R-001: Deterministic Checksum Algorithm
**Question**: How to ensure deterministic bundle checksums across different platforms and file orderings?

**Research Areas**:
- Lexicographic sorting of SHA256 hashes (ensure consistent collation)
- Newline handling in concatenated hash strings (Unix vs Windows line endings)
- Floating-point timestamp exclusion (use only content, not metadata)

**Deliverable**: Algorithm pseudocode in `research.md` with test cases for determinism

#### Task R-002: Lock File Strategy
**Question**: How to prevent concurrent modification races without deadlocks?

**Research Areas**:
- Filesystem-based lock files (`.bundle/.lock`) vs in-memory locks
- Lock timeout strategies (fail-fast vs retry)
- Cross-platform lock file compatibility (Windows vs Unix)

**Deliverable**: Lock acquisition/release protocol in `research.md`

#### Task R-003: Streaming Checksum for Large Files
**Question**: How to compute checksums for 10GB+ files without memory exhaustion?

**Research Areas**:
- Go `io.Reader` interface for streaming SHA256
- Chunk size optimization for I/O performance
- Progress indicator implementation (for files >100MB)

**Deliverable**: Code pattern for streaming checksum in `research.md`

#### Task R-004: Error Handling Patterns
**Question**: How to map library errors to CLI exit codes (0/1/2)?

**Research Areas**:
- Custom error types for user vs system errors
- Logrus integration for error logging before exit
- Error message formatting (stdout vs stderr)

**Deliverable**: Error handling guidelines in `research.md`

#### Task R-005: Testing Strategy
**Question**: What test coverage is needed to meet constitution requirements?

**Research Areas**:
- Unit tests for each library component (checksum, metadata, state, tag, scanner, lock)
- Integration tests using temporary directories and `t.TempDir()`
- Contract tests for CLI exit codes and JSON output format
- Determinism tests (same input → same checksum, run 100 times)

**Deliverable**: Test plan with coverage targets in `research.md`

**Output**: `research.md` with all design decisions documented

---

## Phase 1: Data Model & Contracts

**Status**: PENDING - Awaits Phase 0 completion

### Deliverables Overview
1. `data-model.md` - Go struct definitions and validation rules
2. `contracts/cli-commands.md` - CLI command specifications
3. `quickstart.md` - Developer onboarding guide
4. Agent context update via script

---

## Phase 2: Task Breakdown

**Status**: NOT STARTED - Use `/speckit.tasks` command after Phase 1 completes

**Next Command**: `/speckit.tasks` to generate prioritized task list based on this plan and data model

---

## Notes

- Constitution compliance verified: All 5 principles satisfied
- No complexity violations to justify
- Library-first design enables future backend integration
- CLI provides both human (table) and machine (JSON) interfaces
- Deterministic checksums critical for distributed deduplication (future)
- Lock mechanism prevents corruption, not needed for read operations
- User preferences from COPILOT.md followed (cmd/ structure, go:embed for messages, utils/)

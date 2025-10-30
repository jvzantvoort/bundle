<!--
Sync Impact Report:
- Version: 0.0.0 → 1.0.0
- Initial constitution ratification
- Principles established: Content Integrity, Library-First Architecture, CLI-First Interface, Immutability, Observability & Debugging
- Templates requiring updates: ⚠ Pending validation of .specify/templates/* alignment
-->

# Bundle Library Constitution

## Core Principles

### I. Content Integrity (NON-NEGOTIABLE)
Bundle identity MUST be derived solely from content using SHA256 checksums. All file changes result in new bundle identities. Integrity verification MUST succeed before any bundle operation (sync, replicate, or export).

**Rationale**: Content-addressable design ensures immutability, deduplication, and tamper detection across distributed storage.

### II. Library-First Architecture
Every feature MUST start as a standalone library component with clear interfaces. Libraries MUST be:
- Self-contained and independently testable
- Documented with Go doc comments
- Free of CLI dependencies (use dependency injection)

**Rationale**: Enables reuse by backend services, CLI tools, and future integrations without coupling.

### III. CLI-First Interface
Every library component MUST expose functionality via Cobra CLI commands following this contract:
- Text-based I/O: stdin/args → stdout (data), stderr (errors/logs)
- Support both JSON (`--json`) and human-readable table formats (using tablewriter)
- Exit codes: 0 (success), 1 (user error), 2 (system error)

**Rationale**: Ensures scriptability, testability, and composability with Unix pipelines.

### IV. Immutability & Determinism
Bundle operations MUST be deterministic and idempotent:
- Identical file sets produce identical bundle checksums regardless of order or timestamps
- Metadata changes (title, tags) MUST NOT alter bundle checksum
- Physical bundle names MUST be derived from content checksum (no user-specified names)

**Rationale**: Enables reliable deduplication, replication validation, and distributed synchronization.

### V. Observability & Debugging
All operations MUST emit structured logs using logrus with these levels:
- `ERROR`: Integrity failures, I/O errors, unrecoverable states
- `WARN`: Missing metadata, checksum mismatches, deprecated features
- `INFO`: Bundle operations (create, verify, sync), state changes
- `DEBUG`: Checksum computations, file scans, storage adapter calls

CLI output MUST separate user-facing results (stdout) from operational logs (stderr).

**Rationale**: Enables post-mortem debugging and operational monitoring in distributed DAM deployments.

## Technology Standards

### Language & Toolchain
- **Language**: Go 1.21+ (use generics where appropriate)
- **Build**: Standard `go build`, no custom build scripts
- **Modules**: Go modules with semantic versioning

### Required Libraries
- `github.com/spf13/cobra` — CLI framework
- `github.com/spf13/viper` — Configuration management
- `github.com/sirupsen/logrus` — Structured logging
- `github.com/olekukonko/tablewriter` — Human-readable output
- `github.com/fatih/color` — Terminal colorization

### Code Structure
Follow this canonical layout:
- `cmd/<command>/main.go` — CLI entry points
- `config/` — Viper-based configuration handling
- `<functional-area>/` — Core library components (e.g., `checksum/`, `metadata/`, `storage/`)
- `messages/{long,usage,short}/<command>` — Help text using `go:embed`
- `utils/` — Shared utilities (avoid dumping grounds; keep focused)

## Development Workflow

### Testing Requirements
- Unit tests MUST cover all library functions (`*_test.go` alongside source)
- Integration tests REQUIRED for: bundle creation, integrity checks, storage adapters
- Checksum computation MUST have determinism tests (same input → same output)
- CLI commands MUST have end-to-end tests using `exec.Command` or test harnesses

### Code Review Gates
Before merge, all changes MUST:
1. Pass `go test ./...` with no failures
2. Pass `go vet ./...` with no warnings
3. Pass `gofmt -s` (code must be formatted)
4. Update relevant documentation (README, BUNDLE.md, or inline Go docs)
5. Verify no hardcoded paths, credentials, or non-portable assumptions

### Breaking Changes
API/CLI changes that break backward compatibility REQUIRE:
- MAJOR version bump (per semantic versioning)
- Migration guide in CHANGELOG
- Deprecation warning period (1 minor version minimum) before removal

## Governance

This constitution supersedes all informal practices. All pull requests, code reviews, and design decisions MUST comply with these principles.

**Amendment Process**:
1. Propose change with rationale and impact assessment
2. Update constitution version (MAJOR for principle changes, MINOR for additions, PATCH for clarifications)
3. Propagate changes to `.specify/templates/*` and update dependent documentation
4. Commit with message: `docs: amend constitution to vX.Y.Z (<summary>)`

**Compliance**: Maintainers MUST verify constitutional compliance during code review. Complexity or deviations MUST be justified in PR descriptions.

**Runtime Guidance**: See `COPILOT.md` for detailed implementation patterns and operational context.

**Version**: 1.0.0 | **Ratified**: 2025-10-30 | **Last Amended**: 2025-10-30

# Feature Specification: Bundle Library Core Implementation

**Feature Branch**: `001-bundle-core`  
**Created**: 2025-10-30  
**Status**: Draft  
**Input**: User description: "Build the application code for the bundle library, add a cli and test code"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create and Verify Bundle Integrity (Priority: P1)

As a digital asset manager, I need to create a bundle from a collection of files and verify its integrity so that I can ensure content authenticity and detect corruption across distributed storage systems.

**Why this priority**: Core value proposition - content-addressable, immutable bundles are the foundation of the entire DAM system. Without this, no other functionality is possible.

**Independent Test**: Can be fully tested by creating a bundle from a directory of test files, computing its checksum, modifying a file, and verifying that integrity check fails. Delivers immediate value for basic asset protection.

**Acceptance Scenarios**:

1. **Given** a directory containing multiple files, **When** user runs `bundle create <path>`, **Then** the system creates a `.bundle/` subdirectory with META.json, SHA256SUM.txt, and STATE.json files, and generates a deterministic bundle checksum
2. **Given** an existing bundle with valid checksums, **When** user runs `bundle verify <path>`, **Then** the system validates all file checksums and reports success
3. **Given** a bundle where a file has been modified, **When** user runs `bundle verify <path>`, **Then** the system detects the checksum mismatch and reports which file(s) are corrupted
4. **Given** two bundles with identical file content but different creation times, **When** checksums are computed, **Then** both bundles produce identical checksums (deterministic naming)

---

### User Story 2 - Manage Bundle Metadata and Tags (Priority: P2)

As a digital asset manager, I need to add human-readable titles and searchable tags to bundles so that I can organize and find assets without relying solely on content hashes.

**Why this priority**: Essential for usability - users need human-friendly names and organization. However, the bundle is still functional without metadata (P1 works first).

**Independent Test**: Can be tested by creating a bundle, setting a title, adding tags, and retrieving bundle information to confirm metadata persistence. Tags enable basic search/filter workflows.

**Acceptance Scenarios**:

1. **Given** a newly created bundle, **When** user runs `bundle create <path> --title "Iceland Vacation 2024"`, **Then** the META.json file contains the title and creation timestamp
2. **Given** an existing bundle, **When** user runs `bundle tag add <path> travel iceland photos`, **Then** the TAGS.txt file contains all three tags on separate lines
3. **Given** a bundle with tags, **When** user runs `bundle tag list <path>`, **Then** the system displays all tags associated with the bundle
4. **Given** a bundle, **When** user runs `bundle info <path>`, **Then** the system displays title, checksum, size, tag count, creation date, and last verification time in both human-readable table format and JSON format (with --json flag)

---

### User Story 3 - Query and Inspect Bundle Structure (Priority: P3)

As a digital asset manager, I need to inspect bundle contents and understand their structure so that I can audit what files are included and troubleshoot issues.

**Why this priority**: Operational visibility is important for debugging and auditing, but not required for basic bundle operations. Can be partially achieved with filesystem tools.

**Independent Test**: Can be tested by creating bundles with known file structures and verifying the CLI displays accurate file listings, checksums, and metadata. Useful for support and operations teams.

**Acceptance Scenarios**:

1. **Given** an existing bundle, **When** user runs `bundle list <path>`, **Then** the system displays all files in the bundle with their individual checksums and sizes
2. **Given** a bundle, **When** user runs `bundle info <path> --json`, **Then** the system outputs structured JSON containing all metadata, checksums, tags, and file inventory for programmatic consumption
3. **Given** a corrupted bundle with missing SHA256SUM.txt, **When** user runs `bundle info <path>`, **Then** the system reports the bundle state as "incomplete" and suggests running `bundle rebuild`

---

### Edge Cases

- What happens when a bundle directory is empty (no files to checksum)?
  - System should create minimal `.bundle/` structure with empty SHA256SUM.txt and warn user
- How does the system handle concurrent modifications to the same bundle?
  - Lock mechanism prevents race conditions during checksum computation and metadata updates
- What happens when `.bundle/` subdirectory is deleted but files remain?
  - System treats it as uninitialized directory; `bundle verify` fails with "not a bundle" error
- How does the system handle symbolic links in bundle directories?
  - Follow symlinks during checksum computation, but document that moving bundles may break links (warn in CLI)
- What happens when file permissions prevent reading files during checksum computation?
  - Skip unreadable files, log warnings, and mark bundle as "incomplete" in STATE.json
- How does the system handle very large files (>10GB)?
  - Stream-based checksum computation to avoid memory exhaustion; show progress indicator for files >100MB
- What happens when file names contain special characters or non-UTF8 encodings?
  - Support UTF-8 filenames; document limitations and fail gracefully with clear error messages for unsupported encodings

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST compute SHA256 checksums for all files in the bundle directory (excluding `.bundle/` subdirectory)
- **FR-002**: System MUST generate a deterministic bundle checksum by sorting file checksums lexicographically, concatenating them, and computing SHA256 of the concatenated string
- **FR-003**: System MUST create a `.bundle/META.json` file containing title, creation timestamp, bundle checksum, author (from system user), and version number
- **FR-004**: System MUST create a `.bundle/SHA256SUM.txt` file following standard `sha256sum` output format (checksum, space, space, filepath)
- **FR-005**: System MUST create a `.bundle/STATE.json` file containing verification status, last checked timestamp, replica locations, and total size in bytes
- **FR-006**: System MUST persist tags in `.bundle/TAGS.txt` as newline-separated plain text entries
- **FR-007**: CLI MUST provide commands: `create`, `verify`, `info`, `list`, `tag add`, `tag remove`, `tag list`
- **FR-008**: All CLI commands MUST support `--json` flag to output structured JSON for programmatic consumption
- **FR-009**: All CLI commands MUST exit with code 0 on success, 1 for user errors (invalid arguments), 2 for system errors (I/O failures)
- **FR-010**: System MUST emit structured logs using logrus with levels: ERROR (integrity failures, I/O errors), WARN (missing metadata), INFO (operations), DEBUG (checksums)
- **FR-011**: CLI MUST separate user-facing output (stdout) from operational logs (stderr)
- **FR-012**: System MUST support concurrent-safe operations using lock files in `.bundle/.lock` during write operations
- **FR-013**: System MUST validate bundle integrity by recomputing file checksums and comparing against SHA256SUM.txt
- **FR-014**: System MUST detect and report which specific files fail integrity checks
- **FR-015**: CLI commands MUST display human-readable tables using tablewriter for non-JSON output

### Key Entities

- **Bundle**: A directory containing files and a `.bundle/` subdirectory with metadata; identified by content-derived checksum; represents an immutable collection of assets
- **Asset/File**: Individual files within the bundle; each has SHA256 checksum, size, and path relative to bundle root; excludes `.bundle/` directory contents
- **Metadata**: Title (human-readable name), creation timestamp, author, version number, bundle checksum; stored in META.json; title changes do not affect bundle checksum
- **Tag**: User-assigned classification label; stored in TAGS.txt; used for search and organization; multiple tags allowed per bundle
- **Checksum Record**: Entry in SHA256SUM.txt containing file path and SHA256 hash; sorted lexicographically by hash for deterministic bundle checksum computation
- **Bundle State**: Verification status, last checked timestamp, known replica locations, total size; stored in STATE.json; updated by verify operations

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create a bundle from 100 files (totaling 1GB) in under 30 seconds on standard hardware
- **SC-002**: Identical file collections produce identical bundle checksums regardless of creation time or directory order (100% determinism)
- **SC-003**: Integrity verification correctly detects 100% of file modifications, deletions, or corruptions
- **SC-004**: All CLI commands complete successfully with exit code 0 for valid inputs and provide clear error messages (with exit codes 1 or 2) for invalid inputs
- **SC-005**: Bundle operations support files up to 10GB without memory exhaustion (streaming checksum computation)
- **SC-006**: CLI provides both human-readable table output and machine-parseable JSON output for all query commands
- **SC-007**: System handles 1000+ files per bundle without performance degradation
- **SC-008**: Concurrent bundle operations on different bundles execute without conflicts (lock isolation works correctly)

## Assumptions

- Users have read/write permissions to directories where bundles are created
- File systems support standard POSIX file operations (create, read, stat)
- System user environment provides username for author field in metadata
- Default behavior follows Unix philosophy: text-based I/O, composability, exit codes
- Bundle directories are stored on local or network-mounted filesystems (not object storage like S3 - that's handled by future storage adapters)
- File timestamps are available but not used for checksum computation (content-only hashing)
- System has sufficient temporary space for lock files and atomic write operations
- Go standard library crypto/sha256 provides adequate performance for checksum computation

## Out of Scope

- Replication to remote storage backends (future: storage adapters for NAS, S3, rsync)
- Deduplication engine (future: block-level hashing and cross-bundle deduplication)
- Backend API integration (this spec covers library + CLI; backend is separate)
- User authentication/authorization (single-user, filesystem-based permissions only)
- Bundle compression or encryption (future enhancement)
- GUI or web interface (CLI only for this phase)
- Migration tools for existing asset collections (future: import utilities)
- Automated integrity verification scheduling (future: daemon/cron integration)

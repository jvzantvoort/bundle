# Tasks: Bundle Library Core Implementation

**Input**: Design documents from `/specs/001-bundle-core/`  
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅

**Tests**: Test tasks included per constitution requirement (Development Workflow section mandates unit, integration, and contract tests)

**Organization**: Tasks grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Project uses single-project Go structure following user preferences:
- Library components: `checksum/`, `metadata/`, `state/`, `tag/`, `scanner/`, `lock/`, `bundle/`
- CLI commands: `cmd/bundle/`, `cmd/create/`, `cmd/verify/`, etc.
- Tests: `tests/integration/`, `tests/contract/`
- Help messages: `messages/long/`, `messages/usage/`, `messages/short/`
- Utilities: `utils/`, `config/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and Go module setup

- [X] T001 Initialize Go module at repository root with `go mod init github.com/jvzantvoort/bundle`
- [X] T002 Install dependencies: cobra, viper, logrus, tablewriter, color using `go get`
- [X] T003 [P] Create project directory structure (cmd/, config/, utils/, messages/, tests/)
- [X] T004 [P] Create README.md with project overview and build instructions
- [X] T005 [P] Create .gitignore for Go projects (ignore binaries, test artifacts)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core utilities and error handling infrastructure that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T006 Implement custom error types in utils/errors.go (ErrNotABundle, ErrBundleLocked, ErrCorruptedBundle, ErrInvalidPath)
- [ ] T007 Implement exit code mapping in utils/exit.go (ExitCodeFromError function)
- [ ] T008 [P] Implement filepath utilities in utils/filepath.go (path normalization, .bundle/ exclusion)
- [ ] T009 [P] Implement output helpers in utils/output.go (stdout/stderr, JSON vs table formatting)
- [ ] T010 Setup logrus configuration in config/main.go (log levels, structured fields)
- [ ] T011 [P] Create help message directory structure (messages/long/, messages/usage/, messages/short/)
- [ ] T012 Write unit tests for utils/errors.go in utils/errors_test.go
- [ ] T013 Write unit tests for utils/exit.go in utils/exit_test.go
- [ ] T014 [P] Write unit tests for utils/filepath.go in utils/filepath_test.go
- [ ] T015 [P] Write unit tests for utils/output.go in utils/output_test.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Create and Verify Bundle Integrity (Priority: P1) 🎯 MVP

**Goal**: Enable users to create content-addressable bundles and verify their integrity using SHA256 checksums

**Independent Test**: Create bundle from test files → modify a file → verify detects corruption

### Library Components for User Story 1

#### Checksum Library (Core integrity)

- [ ] T016 [P] [US1] Implement streaming SHA256 computation in checksum/stream.go (ComputeFileSHA256 using io.Copy)
- [ ] T017 [P] [US1] Implement deterministic bundle checksum in checksum/main.go (sort checksums, concatenate, hash)
- [ ] T018 [P] [US1] Implement ChecksumRecord and ChecksumFile structs in checksum/main.go
- [ ] T019 [P] [US1] Implement Load/Save for SHA256SUM.txt format in checksum/main.go
- [ ] T020 [US1] Implement Compute function (scan directory, compute all checksums) in checksum/main.go
- [ ] T021 [US1] Implement Verify function (recompute checksums, compare) in checksum/main.go
- [ ] T022 [P] [US1] Write unit tests for streaming checksum in checksum/stream_test.go
- [ ] T023 [P] [US1] Write unit tests for deterministic bundle checksum in checksum/main_test.go (100 iterations with shuffled order)
- [ ] T024 [P] [US1] Write unit tests for SHA256SUM.txt parsing in checksum/main_test.go

#### Scanner Library (File discovery)

- [ ] T025 [P] [US1] Implement directory scanner in scanner/main.go (walk directory, exclude .bundle/)
- [ ] T026 [P] [US1] Implement symlink handling logic in scanner/main.go
- [ ] T027 [P] [US1] Write unit tests for scanner in scanner/main_test.go (test exclusion, symlinks)

#### Metadata Library (Bundle metadata)

- [ ] T028 [P] [US1] Define Metadata struct in metadata/types.go (title, created_at, bundle_checksum, author, version)
- [ ] T029 [P] [US1] Implement Load/Save for META.json in metadata/main.go (JSON marshaling with RFC3339 timestamps)
- [ ] T030 [P] [US1] Implement Validate function in metadata/main.go (check checksum format, version ≥ 1)
- [ ] T031 [P] [US1] Write unit tests for metadata serialization in metadata/main_test.go

#### State Library (Operational state)

- [ ] T032 [P] [US1] Define State struct in state/main.go (verified, last_checked, replicas, size_bytes)
- [ ] T033 [P] [US1] Implement Load/Save for STATE.json in state/main.go
- [ ] T034 [P] [US1] Implement MarkVerified and UpdateSize methods in state/main.go
- [ ] T035 [P] [US1] Write unit tests for state management in state/main_test.go

#### Lock Library (Concurrency control)

- [ ] T036 [P] [US1] Implement lock acquisition in lock/main.go (O_CREATE|O_EXCL for atomic creation)
- [ ] T037 [P] [US1] Implement lock release in lock/main.go (close file, remove .lock)
- [ ] T038 [P] [US1] Write PID to lock file for debugging in lock/main.go
- [ ] T039 [P] [US1] Write unit tests for lock behavior in lock/main_test.go (concurrent access tests)

#### Bundle High-Level Operations

- [ ] T040 [US1] Implement Bundle struct in bundle/main.go (aggregate all metadata entities)
- [ ] T041 [US1] Implement Create function in bundle/main.go (scan, checksum, create .bundle/, write metadata)
- [ ] T042 [US1] Implement Verify function in bundle/main.go (load checksums, verify, update state)
- [ ] T043 [US1] Implement Load function in bundle/main.go (read all metadata files)
- [ ] T044 [P] [US1] Write integration tests for Create in tests/integration/create_test.go (use t.TempDir())
- [ ] T045 [P] [US1] Write integration tests for Verify in tests/integration/verify_test.go (test corruption detection)

### CLI Commands for User Story 1

#### Root Command

- [ ] T046 [P] [US1] Implement root command in cmd/bundle/main.go (Cobra setup, version flag)
- [ ] T047 [P] [US1] Create help messages for bundle command in messages/{long,usage,short}/bundle
- [ ] T048 [P] [US1] Add global flags (--verbose, --quiet, --json) in cmd/bundle/main.go

#### Create Command

- [ ] T049 [US1] Implement create command in cmd/create/main.go (call bundle.Create, handle errors)
- [ ] T050 [US1] Add --title flag to create command in cmd/create/main.go
- [ ] T051 [US1] Implement table output for create in cmd/create/main.go (using tablewriter)
- [ ] T052 [US1] Implement JSON output for create in cmd/create/main.go (--json flag)
- [ ] T053 [P] [US1] Create help messages for create command in messages/{long,usage,short}/create

#### Verify Command

- [ ] T054 [US1] Implement verify command in cmd/verify/main.go (call bundle.Verify, report results)
- [ ] T055 [US1] Implement table output for verify success in cmd/verify/main.go
- [ ] T056 [US1] Implement table output for verify failure (show corrupted files) in cmd/verify/main.go
- [ ] T057 [US1] Implement JSON output for verify in cmd/verify/main.go (--json flag)
- [ ] T058 [P] [US1] Create help messages for verify command in messages/{long,usage,short}/verify

### Contract Tests for User Story 1

- [ ] T059 [P] [US1] Write CLI contract test for create exit codes in tests/contract/cli_test.go
- [ ] T060 [P] [US1] Write CLI contract test for verify exit codes in tests/contract/cli_test.go
- [ ] T061 [P] [US1] Write CLI contract test for JSON output format in tests/contract/cli_test.go

**Checkpoint**: User Story 1 complete - bundle creation and verification fully functional and independently testable

---

## Phase 4: User Story 2 - Manage Bundle Metadata and Tags (Priority: P2)

**Goal**: Enable users to add human-readable titles and searchable tags for bundle organization

**Independent Test**: Create bundle → set title → add tags → retrieve info → verify metadata persistence

### Library Components for User Story 2

#### Tag Library

- [ ] T062 [P] [US2] Define Tags struct in tag/main.go (slice of unique strings)
- [ ] T063 [P] [US2] Implement Load/Save for TAGS.txt in tag/main.go (one tag per line, sorted)
- [ ] T064 [P] [US2] Implement Add method in tag/main.go (append, deduplicate)
- [ ] T065 [P] [US2] Implement Remove method in tag/main.go (filter out tags)
- [ ] T066 [P] [US2] Implement List method in tag/main.go (return sorted tags)
- [ ] T067 [P] [US2] Write unit tests for tag operations in tag/main_test.go

#### Bundle Info Operation

- [ ] T068 [US2] Implement Info method in bundle/main.go (load all metadata, return summary)
- [ ] T069 [US2] Define BundleInfo struct in bundle/main.go (all displayable fields)
- [ ] T070 [P] [US2] Write integration tests for Info in tests/integration/metadata_test.go

### CLI Commands for User Story 2

#### Info Command

- [ ] T071 [US2] Implement info command in cmd/info/main.go (call bundle.Info)
- [ ] T072 [US2] Implement table output for info in cmd/info/main.go (tablewriter formatting)
- [ ] T073 [US2] Implement JSON output for info in cmd/info/main.go (--json flag)
- [ ] T074 [P] [US2] Create help messages for info command in messages/{long,usage,short}/info

#### Tag Command (add/remove/list subcommands)

- [ ] T075 [US2] Implement tag root command in cmd/tag/main.go (Cobra subcommands setup)
- [ ] T076 [US2] Implement tag add subcommand in cmd/tag/main.go (call tag.Add)
- [ ] T077 [US2] Implement tag remove subcommand in cmd/tag/main.go (call tag.Remove)
- [ ] T078 [US2] Implement tag list subcommand in cmd/tag/main.go (call tag.List)
- [ ] T079 [US2] Add table and JSON output for tag commands in cmd/tag/main.go
- [ ] T080 [P] [US2] Create help messages for tag commands in messages/{long,usage,short}/tag

### Contract Tests for User Story 2

- [ ] T081 [P] [US2] Write CLI contract test for info exit codes in tests/contract/cli_test.go
- [ ] T082 [P] [US2] Write CLI contract test for tag add/remove/list in tests/contract/cli_test.go
- [ ] T083 [P] [US2] Write CLI contract test for info JSON output in tests/contract/cli_test.go

**Checkpoint**: User Stories 1 AND 2 both work independently - bundles can be created, verified, tagged, and inspected

---

## Phase 5: User Story 3 - Query and Inspect Bundle Structure (Priority: P3)

**Goal**: Enable users to inspect bundle contents for auditing and troubleshooting

**Independent Test**: Create bundle with known files → list files → verify accurate display of checksums and sizes

### CLI Commands for User Story 3

#### List Command

- [ ] T084 [US3] Implement list command in cmd/list/main.go (load SHA256SUM.txt, stat files)
- [ ] T085 [US3] Implement table output for list in cmd/list/main.go (filename, checksum, size columns)
- [ ] T086 [US3] Implement JSON output for list in cmd/list/main.go (file inventory array)
- [ ] T087 [US3] Add human-readable size formatting (KB, MB, GB) in cmd/list/main.go
- [ ] T088 [P] [US3] Create help messages for list command in messages/{long,usage,short}/list

### Enhanced Info Output

- [ ] T089 [US3] Extend Info output to include file count breakdown in cmd/info/main.go
- [ ] T090 [US3] Add incomplete bundle detection to info command in cmd/info/main.go

### Contract Tests for User Story 3

- [ ] T091 [P] [US3] Write CLI contract test for list exit codes in tests/contract/cli_test.go
- [ ] T092 [P] [US3] Write CLI contract test for list JSON output in tests/contract/cli_test.go

**Checkpoint**: All user stories independently functional - complete bundle lifecycle from create to inspect

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories and finalize the implementation

### Documentation

- [ ] T093 [P] Update README.md with installation instructions and usage examples
- [ ] T094 [P] Add CLI usage examples to quickstart.md based on actual implementation
- [ ] T095 [P] Document edge cases and troubleshooting in README.md

### Code Quality

- [ ] T096 [P] Run `gofmt -s -w .` to format all code
- [ ] T097 [P] Run `go vet ./...` and fix all warnings
- [ ] T098 Run full test suite with coverage (`go test -cover ./...`) and verify >90% unit test coverage
- [ ] T099 [P] Add Go doc comments to all exported functions
- [ ] T100 [P] Review error messages for clarity and actionability

### Build & Validation

- [ ] T101 Build CLI binary with `go build -o bundle ./cmd/bundle`
- [ ] T102 Test quickstart.md workflow end-to-end (follow all examples)
- [ ] T103 Run constitution compliance check (verify all 5 principles satisfied)
- [ ] T104 [P] Create example bundles for testing (small, medium, large file counts)

### Performance Testing

- [ ] T105 Benchmark bundle creation with 100 files (1GB total) - verify <30 seconds
- [ ] T106 Benchmark large file checksums (10GB) - verify streaming works without OOM
- [ ] T107 Benchmark 1000+ file bundles - verify no performance degradation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - **BLOCKS all user stories**
- **User Story 1 (Phase 3)**: Depends on Foundational (Phase 2) - **MVP CRITICAL**
- **User Story 2 (Phase 4)**: Depends on Foundational (Phase 2) - Can run parallel to US1 if staffed
- **User Story 3 (Phase 5)**: Depends on Foundational (Phase 2) - Can run parallel to US1/US2 if staffed
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: No dependencies on other stories - foundational functionality
- **User Story 2 (P2)**: No dependencies on US1 - can start after Foundational phase
- **User Story 3 (P3)**: No dependencies on US1/US2 - can start after Foundational phase

### Within Each User Story

- Library components before CLI commands (library-first architecture)
- Tests can be written in parallel with implementation (TDD encouraged)
- Core functionality before extended features
- Story complete before moving to next priority

### Parallel Opportunities

**Setup Phase**:
- T003, T004, T005 can run in parallel

**Foundational Phase**:
- T008, T009, T011 can run in parallel (different files)
- T012-T015 tests can run in parallel

**User Story 1**:
- T016-T019 checksum library tasks can run in parallel
- T022-T024 checksum tests can run in parallel
- T025, T028, T032, T036 (different library components) can run in parallel
- T044, T045 integration tests can run in parallel
- T047, T048, T053, T058 help messages can run in parallel
- T059-T061 contract tests can run in parallel

**User Story 2**:
- T062-T066 tag library tasks can run in parallel
- T071-T074 info command can run in parallel with T075-T080 tag command
- T081-T083 contract tests can run in parallel

**User Story 3**:
- T091-T092 contract tests can run in parallel

**Polish Phase**:
- T093-T095, T096-T097, T099-T100, T104 can run in parallel

---

## Parallel Example: User Story 1 Library Components

```bash
# Launch all core library components together (different packages):
Task T016: Implement streaming SHA256 in checksum/stream.go
Task T025: Implement directory scanner in scanner/main.go
Task T028: Define Metadata struct in metadata/types.go
Task T032: Define State struct in state/main.go
Task T036: Implement lock acquisition in lock/main.go

# After core libraries, launch tests in parallel:
Task T022: Test streaming checksum
Task T027: Test scanner
Task T031: Test metadata
Task T035: Test state
Task T039: Test lock
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T005)
2. Complete Phase 2: Foundational (T006-T015) - **CRITICAL BLOCKING PHASE**
3. Complete Phase 3: User Story 1 (T016-T061)
4. **STOP and VALIDATE**: Test `bundle create` and `bundle verify` end-to-end
5. Run integration and contract tests
6. Build binary and validate with quickstart examples
7. Deploy/demo MVP if ready

**Estimated Tasks for MVP**: 61 tasks (T001-T061)

### Incremental Delivery

1. **Foundation** (T001-T015) → Project structure ready
2. **MVP Release** (+ T016-T061) → Core bundle creation and verification working
3. **Metadata Release** (+ T062-T083) → Tagging and info commands added
4. **Full Release** (+ T084-T107) → Complete feature set with polish

Each increment adds value without breaking previous functionality.

### Parallel Team Strategy

With multiple developers after Foundational phase completes:

1. **Team completes Setup + Foundational together** (T001-T015)
2. **Once T015 done, split work**:
   - **Developer A**: User Story 1 library components (T016-T043)
   - **Developer B**: User Story 1 CLI commands (T046-T058)
   - **Developer C**: User Story 1 tests (T022-T024, T027, T031, T035, T039, T044-T045, T059-T061)
3. **After US1 complete**:
   - **Developer A**: User Story 2 (T062-T083)
   - **Developer B**: User Story 3 (T084-T092)
   - **Developer C**: Polish & validation (T093-T107)

---

## Validation Checkpoints

### After Setup (T005)
- ✅ Go module initialized
- ✅ Dependencies installed
- ✅ Project structure matches plan.md

### After Foundational (T015)
- ✅ All utils tests pass
- ✅ Error handling framework complete
- ✅ Logging configured

### After User Story 1 (T061)
- ✅ `bundle create` works end-to-end
- ✅ `bundle verify` detects corruption
- ✅ Determinism test passes (100 iterations)
- ✅ All integration tests pass
- ✅ All contract tests pass
- ✅ CLI exit codes correct

### After User Story 2 (T083)
- ✅ Tags persist correctly
- ✅ `bundle info` displays all metadata
- ✅ JSON output validates

### After User Story 3 (T092)
- ✅ `bundle list` shows all files accurately
- ✅ File size formatting works

### Final Validation (T107)
- ✅ All tests pass (`go test ./...`)
- ✅ Coverage >90% on library components
- ✅ Performance benchmarks met
- ✅ quickstart.md workflow validated
- ✅ Constitution principles verified

---

## Notes

- [P] tasks = different files, no shared dependencies, safe to parallelize
- [Story] label enables tracking which user story each task serves
- Each user story is independently completable and testable
- Constitution requires tests (Development Workflow section), so test tasks included
- Go doc comments required for all exported functions (Principle II)
- Exit codes must be 0/1/2 per constitution (Principle III)
- Logrus structured logging required (Principle V)
- Commit after each logical group of related tasks
- Run `go test ./...` frequently to catch issues early
- Avoid: vague tasks, concurrent edits to same file, cross-story dependencies

---

## Task Count Summary

- **Total Tasks**: 107
- **Setup Phase**: 5 tasks
- **Foundational Phase**: 10 tasks (blocking)
- **User Story 1 (P1 - MVP)**: 46 tasks (T016-T061)
- **User Story 2 (P2)**: 22 tasks (T062-T083)
- **User Story 3 (P3)**: 9 tasks (T084-T092)
- **Polish Phase**: 15 tasks (T093-T107)
- **Parallelizable Tasks**: 58 tasks marked [P]

**MVP Scope**: 61 tasks (Setup + Foundational + User Story 1)  
**Full Feature**: 107 tasks

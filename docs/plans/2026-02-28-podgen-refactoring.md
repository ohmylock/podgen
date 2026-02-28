# Podgen Refactoring: Thread Safety, Error Handling & Testability

## Overview
Comprehensive refactoring of the podgen CLI to fix critical issues discovered during code review: BoltDB transactions passed to goroutines (thread-unsafe), 26 instances of log.Fatalf killing processes, zero interfaces preventing testing. Patterns adopted from gitlabenver (worker pools, error accumulation) and ralphex (domain errors, context support).

## Context
- **Files involved:**
  - `cmd/podgen/main.go` (entry point)
  - `internal/app/podgen/app.go` (orchestration)
  - `internal/app/podgen/proc/processor.go` (core logic)
  - `internal/app/podgen/proc/store.go` (BoltDB)
  - `internal/app/podgen/proc/s3.go` (S3 operations)
  - `internal/app/podgen/proc/files.go` (file scanning)
  - `internal/configs/config.go` (configuration)
- **Existing patterns:** go-flags for CLI, BoltDB for storage, minio for S3, go-pkgz/lgr for logging
- **Dependencies:** moq (mock generation), testify (assertions)

## Development Approach
- Testing approach: Regular (code then tests) for phases 1-4, TDD for phase 5
- Complete each task fully before moving to the next
- Every task includes writing/updating tests
- All tests must pass before starting the next task

## Implementation Tasks

### Task 1: Fix Test Infrastructure

---
model: haiku
priority: P0
complexity: Low
---

**Description:** Fix failing config test and create missing testdata file.

**Files:**
- Create: `internal/configs/testdata/config.yaml`
- Modify: `internal/configs/config_test.go` (if needed)

**Steps:**
- [x] Check current test failure reason
- [x] Create testdata/config.yaml with valid test config
- [x] Run tests to verify fix
- [x] Verify: `go test ./internal/configs/...`

### Task 2: Define Interfaces for Dependency Injection

---
model: sonnet
priority: P0
complexity: Medium
---

**Description:** Create interfaces for Storage, S3, and FileScanner to enable mocking.

**Files:**
- Create: `internal/app/podgen/proc/interfaces.go`
- Modify: `internal/app/podgen/proc/processor.go`

**Steps:**
- [x] Create EpisodeStore interface (6 methods)
- [x] Create ObjectStorage interface (5 methods)
- [x] Create FileScanner interface (1 method)
- [x] Create UploadResult and ObjectInfo types
- [x] Update Processor struct to use interfaces
- [x] Verify: `go build ./...`

### Task 3: Create Domain Error Types

---
model: sonnet
priority: P1
complexity: Medium
---

**Description:** Create structured error types for better error context and handling.

**Files:**
- Create: `internal/errors/errors.go`

**Steps:**
- [x] Create EpisodeError type with PodcastID, Filename, Op, Err fields
- [x] Implement Error() and Unwrap() methods
- [x] Define sentinel errors: ErrNoBucket, ErrEpisodeNotFound
- [x] Write tests for error types
- [x] Verify: `go test ./internal/errors/...`

### Task 4: Replace log.Fatalf in store.go

---
model: sonnet
priority: P0
complexity: Medium
---

**Description:** Replace 4 log.Fatalf calls with proper error returns in BoltDB store.

**Files:**
- Modify: `internal/app/podgen/proc/store.go`

**Steps:**
- [ ] Replace log.Fatalf at line 75 (FindEpisodesBySession)
- [ ] Replace log.Fatalf at line 100 (ChangeStatusEpisodes)
- [ ] Replace log.Fatalf at line 188 (GetLastEpisodeByStatus)
- [ ] Replace log.Fatalf at line 217 (GetLastEpisodeByNotStatus)
- [ ] Update callers to handle errors
- [ ] Verify: `go build ./...` and `go vet ./...`

### Task 5: Replace log.Fatalf in files.go and processor.go

---
model: sonnet
priority: P0
complexity: Medium
---

**Description:** Replace 10 log.Fatalf calls with error returns.

**Files:**
- Modify: `internal/app/podgen/proc/files.go` (2 calls)
- Modify: `internal/app/podgen/proc/processor.go` (8 calls)

**Steps:**
- [ ] files.go: Replace log.Fatalf at lines 26, 42
- [ ] processor.go: Replace log.Fatalf at lines 44, 63, 75, 222, 242, 268, 284, 355-365
- [ ] Remove unreachable code after fatalf replacements
- [ ] Update callers to propagate errors
- [ ] Verify: `go build ./...` and `go vet ./...`

### Task 6: Fix BoltDB Thread Safety - Store Wrapper

---
model: opus
priority: P0
complexity: High
---

**Description:** Encapsulate transaction management inside BoltDB wrapper to prevent goroutine leakage.

**Files:**
- Modify: `internal/app/podgen/proc/store.go`

**Steps:**
- [ ] Add sync.Mutex to BoltDB struct
- [ ] Create WithWriteTx(fn func(*bolt.Tx) error) method
- [ ] Create WithReadTx(fn func(*bolt.Tx) error) method
- [ ] Refactor all store methods to use internal transactions
- [ ] Remove tx parameter from public method signatures
- [ ] Verify: `go test -race ./...`

### Task 7: Fix BoltDB Thread Safety - App Refactoring

---
model: opus
priority: P0
complexity: High
---

**Description:** Refactor app.go to use sequential DB operations and parallel S3 only.

**Files:**
- Modify: `internal/app/podgen/app.go`
- Create: `internal/app/podgen/proc/worker.go`
- Modify: `cmd/podgen/main.go`

**Steps:**
- [ ] Create worker pool for S3 operations (worker.go)
- [ ] Refactor Update() - remove tx parameter, sequential iteration
- [ ] Refactor UploadEpisodes() - sequential DB, parallel S3
- [ ] Refactor DeleteOldEpisodes() - sequential DB, parallel S3 delete
- [ ] Refactor GenerateFeed(), RollbackEpisodes(), RollbackEpisodesBySession()
- [ ] Update main.go - remove CreateTransaction, direct method calls
- [ ] Verify: `go test -race ./...`

### Task 8: Add Testing Infrastructure with Mocks

---
model: sonnet
priority: P1
complexity: Medium
---

**Description:** Set up mock generation and write unit tests for core components.

**Files:**
- Create: `internal/app/podgen/proc/generate.go`
- Create: `internal/app/podgen/proc/mocks/` directory
- Create: `internal/app/podgen/proc/processor_test.go`
- Create: `internal/app/podgen/proc/store_test.go`

**Steps:**
- [ ] Add moq to go.mod dependencies
- [ ] Create generate.go with go:generate directives
- [ ] Run `go generate ./...` to create mocks
- [ ] Write TestProcessor_Update with table-driven tests
- [ ] Write TestBoltDB_Integration with temp database
- [ ] Achieve >60% coverage on processor.go
- [ ] Verify: `go test -v -cover ./...`

### Task 9: Add Context Support and Graceful Shutdown

---
model: sonnet
priority: P2
complexity: Medium
---

**Description:** Add context.Context to all public methods and implement graceful shutdown.

**Files:**
- Modify: `internal/app/podgen/app.go`
- Modify: `internal/app/podgen/proc/processor.go`
- Modify: `cmd/podgen/main.go`
- Create: `internal/configs/validate.go`

**Steps:**
- [ ] Add context.Context as first parameter to all App methods
- [ ] Add context.Context to Processor methods
- [ ] Check ctx.Done() before long operations
- [ ] Implement signal handling in main.go (SIGINT, SIGTERM)
- [ ] Create config validation (Validate() method)
- [ ] Test graceful shutdown with Ctrl+C
- [ ] Verify: `go build ./...` and manual testing

### Task 10: Final Verification

---
model: haiku
priority: P0
complexity: Low
---

**Description:** Complete verification of all changes.

**Steps:**
- [ ] Run full test suite: `go test -v -race -cover ./...`
- [ ] Run linting: `go vet ./...`
- [ ] Manual test: scan, upload, feed generation flow
- [ ] Manual test: Ctrl+C graceful shutdown
- [ ] Verify test coverage >= 60%
- [ ] Update README if CLI behavior changed

## File Summary

| Task | New Files | Modified Files |
|------|-----------|----------------|
| 1 | `internal/configs/testdata/config.yaml` | `internal/configs/config_test.go` |
| 2 | `internal/app/podgen/proc/interfaces.go` | `internal/app/podgen/proc/processor.go` |
| 3 | `internal/errors/errors.go` | — |
| 4 | — | `internal/app/podgen/proc/store.go` |
| 5 | — | `internal/app/podgen/proc/files.go`, `processor.go` |
| 6 | — | `internal/app/podgen/proc/store.go` |
| 7 | `internal/app/podgen/proc/worker.go` | `app.go`, `main.go` |
| 8 | `proc/generate.go`, `proc/mocks/*`, `proc/*_test.go` | — |
| 9 | `internal/configs/validate.go` | `app.go`, `processor.go`, `main.go` |

## Risks and Assumptions

| Risk | Impact | Mitigation |
|------|--------|------------|
| BoltDB refactoring breaks existing data | High | Test with copy of production DB |
| Race conditions after refactoring | High | Use -race flag in all tests |
| Breaking CLI interface | Medium | Maintain same flags/behavior |
| Performance regression from sequential DB | Low | S3 remains parallel (bottleneck) |

**Assumptions:**
- BoltDB database schema remains unchanged
- S3 API (minio) behavior unchanged
- CLI flags and output format preserved

**Open Questions:**
- Should we add structured logging (slog/zap)?
- Target test coverage percentage?

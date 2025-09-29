# Master Test Improvement Plan for Oar Project

## Current State Analysis

**43 test files** across multiple packages with several critical issues:
- Heavy duplication in test utilities and setup code
- Mixed unit and integration tests causing slow CI runs
- Inconsistent mocking patterns across packages
- Time-based flakiness in watcher tests
- Scattered test helpers limiting reusability

## Critical Issues Identified

### 1. Test Infrastructure Duplication
- `setupTestDB()` duplicated in `services/testutils_test.go` and `models/testutils_test.go`
- Mock implementations scattered across packages (`MockProjectManager`, `MockGitExecutor`)
- Test data builders repeated with slight variations
- Helper functions (`stringPtr`, `generateTestKey`) duplicated

### 2. Test Organization Anti-Patterns
- Integration tests without build tags slow down default test runs
- Large test files (600+ lines) mixing unrelated behaviors
- Test utilities in `_test.go` files aren't importable across packages
- Missing `t.Helper()` markers on utility functions

### 3. Reliability Issues
- Time-based tests using `time.Sleep` causing flakiness
- SQLite `:memory:` DSN issues with GORM connection pooling
- Docker/Git dependencies without proper skip conditions
- Inconsistent error testing patterns

## Master Improvement Plan

### Phase 1: Centralize Test Infrastructure (Critical Priority)

#### 1.1 Create Unified Test Support Package
**Location**: `internal/testutil/` (with `//go:build test` build constraint)

**Structure**:
```
internal/testutil/
├── db.go          # Unified database setup
├── crypto.go      # Shared encryption utilities
├── ptr.go         # Pointer helpers (stringPtr, etc.)
├── http.go        # HTTP/web test helpers
├── fixtures.go    # Golden file and test data utilities
├── builders/      # Test data builders
│   ├── project.go
│   └── deployment.go
└── clock.go       # Time abstraction for tests
```

**Benefits**: Single source of truth, eliminates all setup duplication

#### 1.2 Standardize Mock Generation
**Location**: `internal/mocks/` (generated via mockery)

**Implementation**:
```go
//go:generate mockery --name=GitExecutor --outpkg=mocks --with-expecter
//go:generate mockery --name=ProjectManager --outpkg=mocks --with-expecter
//go:generate mockery --name=ComposeProjectInterface --outpkg=mocks --with-expecter
```

**Interfaces to unify**:
- `services.GitExecutor`
- `services.ProjectManager`
- `services.ComposeProjectInterface`
- `services.EnvProvider`

### Phase 2: Test Stratification and Build Tags (Critical Priority)

#### 2.1 Separate Integration Tests
- Add `//go:build integration` tags to all Docker/Git dependent tests
- Rename integration files with `_integration_test.go` suffix
- Add environment capability checks with `t.Skipf` fallbacks

#### 2.2 Enhanced Makefile Targets
```makefile
test:
	gotestsum --format testname -- -race -short ./...

test_integration:
	gotestsum --format testname -- -race -tags=integration ./...

test_all:
	make test && make test_integration
```

#### 2.3 CI Pipeline Optimization
- Default CI runs fast unit tests only
- Separate integration test job with Docker/Git setup
- Parallel execution where possible

### Phase 3: Eliminate Time-Based Flakiness (Critical Priority)

#### 3.1 Clock Interface Injection
**Problem**: Watcher tests use real `time.Sleep` making them non-deterministic

**Solution**:
```go
type ClockInterface interface {
    Now() time.Time
    NewTicker(d time.Duration) TickerInterface
}

type TickerInterface interface {
    C() <-chan time.Time
    Stop()
}
```

**Implementation**: Use `benbjohnson/clock` or similar for deterministic tests

### Phase 4: CLI Test Harness (High Priority)

#### 4.1 Unified Command Testing
**Location**: Enhanced `cmd/test/` package

**Features**:
- App initialization with mock dependencies
- Stdout/stderr capture and validation
- Golden file comparison with update flags
- Environment variable and flag testing

**Benefits**: Eliminates 80% of CLI test boilerplate

### Phase 5: Database and Web Test Improvements (Medium Priority)

#### 5.1 Fix SQLite Connection Issues
- Replace `:memory:` with `file::memory:?cache=shared&_fk=1`
- Add transaction rollback patterns
- Test constraint violations and migration edge cases

#### 5.2 HTTP Router-Level Testing
- End-to-end tests mounting full chi router
- Middleware and template rendering validation
- Authentication flow testing

### Phase 6: Enhanced Coverage and Quality (Medium Priority)

#### 6.1 Split Overgrown Test Files
Break large files like `services/project_test.go` into focused files:
- `project_service_list_test.go`
- `project_service_create_test.go`
- `project_service_deploy_test.go`

#### 6.2 Add Missing Coverage Areas
- Concurrency and race condition testing
- Error path and recovery scenarios
- Config validation and precedence
- Logging output verification

### Phase 7: Golden Files and Fixture Management (Low Priority)

#### 7.1 Standardize Test Data
- Package-local `testdata/` directories with `*.golden` files
- Shared `-update` flag for golden file regeneration
- Consolidated fixture repositories under consistent structure

#### 7.2 Property-Based Testing
- Generate test inputs for data validation functions
- Test invariants across CRUD operations
- Edge case discovery for parsers and validators

## Implementation Roadmap

### Week 1: Foundation
1. Create `internal/testutil/` package with core utilities
2. Consolidate `setupTestDB` and crypto helpers
3. Update 2-3 packages to validate approach

### Week 2: Mock Standardization
1. Generate unified mocks with mockery
2. Replace hand-rolled mocks in services package
3. Update watcher and other packages

### Week 3: Test Separation
1. Tag integration tests with build constraints
2. Update Makefile and CI configuration
3. Convert Docker/Git tests to use local resources

### Week 4: Time Interface
1. Inject clock interface into WatcherService
2. Implement fake clock for tests
3. Remove all `time.Sleep` calls

### Week 5-6: CLI and Web
1. Build unified CLI test harness
2. Refactor command tests to use harness
3. Add HTTP router-level web tests

### Week 7-8: Coverage and Cleanup
1. Split large test files
2. Add concurrency testing
3. Enhance error path coverage

## Success Metrics

### Before Implementation
- 43 test files with significant duplication
- Flaky time-based tests causing CI failures
- Mixed unit/integration causing 30s+ test runs
- 4+ different mock implementations for same interfaces

### After Implementation
- Zero code duplication in test utilities
- Deterministic tests (no time.Sleep dependencies)
- Fast unit tests (<5s) separated from integration
- Consistent mocking pattern across all packages
- Comprehensive error and edge case coverage
- Race condition testing enabled in CI

## Risk Mitigation

**Low Risk**: Test utility consolidation and mock standardization
**Medium Risk**: Clock interface injection (requires service changes)
**High Risk**: Integration test separation (may break existing workflows)

**Mitigation Strategy**: Implement incrementally with backward compatibility during transition periods

## Files to be Created/Modified

### New Files
- `internal/testutil/*.go` (db, crypto, ptr, http, fixtures, clock, builders)
- `internal/mocks/*.go` (generated)
- Enhanced `cmd/test/` harness

### Major Refactors
- `services/testutils_test.go` → Move to shared package
- `models/testutils_test.go` → Move to shared package
- `services/mocks_test.go` → Replace with generated mocks
- All integration tests → Add build tags
- Large test files → Split by functionality

### Configuration Updates
- Makefile test targets
- CI pipeline separation
- Documentation in AGENTS.md

---

This master plan synthesizes the best insights from all four agent analyses while providing a practical, risk-managed implementation approach focused on eliminating duplication, improving reliability, and establishing scalable testing patterns.

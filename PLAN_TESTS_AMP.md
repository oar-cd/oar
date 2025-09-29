# Test Improvement Plan for Oar Project

## Current State Analysis

### Test Structure Overview
- **43 test files** across multiple packages
- **services/**: 14 test files (unit + integration + mocks)
- **cmd/**: 16 test files across CLI subcommands
- **models/**: 2 test files with testutils
- **web/**: 3 test files for handlers/routes/actions
- **watcher/, db/, logging/**: 1-2 test files each

### Major Issues Identified

#### 1. Critical Code Duplication
- **setupTestDB**: Duplicated in `services/testutils_test.go` and `models/testutils_test.go`
- **MockGitExecutor**: Two different implementations (hand-rolled vs testify/mock)
- **MockProjectManager**: Same duplication pattern as GitExecutor
- **Test builders**: Project creation helpers scattered across packages
- **Helper functions**: `stringPtr`, `generateTestKey` duplicated

#### 2. Testing Anti-Patterns
- **Mixed unit/integration**: Integration tests without build tags cause CI slowness
- **Time-based flakiness**: Watcher tests use `time.Sleep` making them non-deterministic
- **Inconsistent mocking**: Mix of hand-rolled fakes and testify/mock
- **SQLite `:memory:`**: Unsafe with GORM connection pooling

#### 3. Organizational Weaknesses
- Test utilities in `_test.go` files aren't importable across packages
- Missing `t.Helper()` markers on helper functions
- No separation between fast unit tests and slow integration tests
- Inconsistent logging setup (only in services package)

## Improvement Plan

### Phase 1: Centralize Test Infrastructure (High Priority)

#### 1.1 Create Shared Test Utilities
**Location**: `internal/testutil/` (with `//go:build test`)

**Files to create**:
- `db.go`: Centralized `SetupTestDB` with safe SQLite DSN
- `crypto.go`: Shared `GenerateTestKey` function
- `ptr.go`: Common pointer helpers (`StringPtr`, etc.)
- `builders/`: Test data builders for all entities

**Benefits**: Eliminates duplication, ensures consistent test DB setup

#### 1.2 Standardize Mocking
**Location**: `internal/mocks/` (with `//go:build test`)

**Approach**: Use mockery for consistent mock generation
```go
//go:generate mockery --name=GitExecutor --outpkg=mocks --output=internal/mocks --with-expecter
//go:generate mockery --name=ProjectManager --outpkg=mocks --output=internal/mocks --with-expecter
```

**Interfaces to mock**:
- `services.GitExecutor`
- `services.ProjectManager`
- `services.ComposeProjectInterface`
- `services.EnvProvider`

### Phase 2: Separate Unit from Integration Tests (High Priority)

#### 2.1 Integration Test Isolation
- Add `//go:build integration` tags to integration tests
- Rename files with `_integration_test.go` suffix
- Use local Git repos (file:// URLs) instead of network calls

#### 2.2 Makefile Targets
```makefile
test:
	go test ./... -race -short

test-integration:
	go test ./... -race -tags=integration

test-all:
	make test && make test-integration
```

### Phase 3: Fix Time-Based Test Flakiness (High Priority)

#### 3.1 Inject Clock Interface
**Problem**: Watcher tests use real `time.Sleep` making them flaky

**Solution**: Dependency injection for time
```go
type TickerFactory func(d time.Duration) Ticker
type Ticker interface {
    C() <-chan time.Time
    Stop()
}
```

**Implementation**: Use `benbjohnson/clock` or similar fake clock library

### Phase 4: Database Test Improvements (Medium Priority)

#### 4.1 Fix SQLite Connection Issues
**Current**: `:memory:` DSN causes issues with GORM pooling
**Solution**: Use `file::memory:?cache=shared&_fk=1` or temp file DBs

#### 4.2 Transaction Test Patterns
- Add tests for rollback scenarios
- Test constraint violations
- Verify migration edge cases

### Phase 5: Enhance Test Coverage (Medium Priority)

#### 5.1 Handler Integration Tests
**Current**: Only unit tests for individual functions
**Add**: End-to-end HTTP tests with router wiring
```go
func TestProjectHandlerIntegration(t *testing.T) {
    // Setup router + handlers
    // Test HTTP requests/responses
    // Assert status codes and bodies
}
```

#### 5.2 CLI Behavior Tests
**Current**: Only structure validation tests
**Add**: Flag parsing, command execution, output validation

#### 5.3 Config Edge Cases
- Invalid duration parsing errors
- Invalid boolean values
- Environment variable precedence

### Phase 6: Code Quality Improvements (Low Priority)

#### 6.1 Test Hygiene
- Add `t.Helper()` to all helper functions
- Use `t.Parallel()` for safe unit tests
- Standardize `require` vs `assert` usage
- Add table-driven tests for input matrices

#### 6.2 Race Condition Testing
- Add concurrency tests for services
- Run CI with `-race` flag consistently
- Test watcher/service interactions under load

## Implementation Order

### Week 1: Critical Duplications
1. Create `internal/testutil/` package
2. Consolidate `setupTestDB` function
3. Move shared crypto/ptr helpers
4. Update all packages to use shared utilities

### Week 2: Mocking Standardization
1. Generate mocks with mockery
2. Replace hand-rolled fakes in services
3. Replace duplicated mocks in watcher
4. Update all tests to use standard mocks

### Week 3: Test Separation
1. Tag integration tests with build constraints
2. Update Makefile with separate test targets
3. Convert Git integration to use local repos
4. Verify CI runs fast unit tests by default

### Week 4: Time Flakiness
1. Inject ticker interface into WatcherService
2. Implement fake clock for tests
3. Remove all `time.Sleep` calls from tests
4. Verify tests run deterministically

### Week 5-6: Coverage & Quality
1. Add handler integration tests
2. Enhance CLI behavior tests
3. Add config edge case coverage
4. Implement race condition tests

## Success Metrics

### Before Implementation
- 43 test files with significant duplication
- Flaky time-based tests causing CI failures
- Mixed unit/integration causing 30s+ test runs
- 4 different mock implementations for same interfaces

### After Implementation
- ✅ Zero code duplication in test utilities
- ✅ Deterministic tests (no time.Sleep)
- ✅ Fast unit tests (<5s) separated from integration
- ✅ Consistent mocking pattern across all packages
- ✅ Improved test coverage for edge cases
- ✅ Race-condition testing enabled

## Files That Will Be Modified/Created

### New Files
- `internal/testutil/db.go`
- `internal/testutil/crypto.go`
- `internal/testutil/ptr.go`
- `internal/testutil/builders/project.go`
- `internal/testutil/builders/deployment.go`
- `internal/mocks/*.go` (generated)

### Major Refactors
- `services/testutils_test.go` → Move to shared package
- `models/testutils_test.go` → Move to shared package
- `services/mocks_test.go` → Replace with generated mocks
- `watcher/watcher_test.go` → Use shared mocks + fake clock
- All integration tests → Add build tags

### Makefile Updates
- Separate test targets for unit vs integration
- Add race detection to CI pipeline
- Add coverage reporting

## Risk Assessment

**Low Risk**: Centralized test utilities and mock standardization
**Medium Risk**: Time interface injection (requires service refactoring)
**High Risk**: Integration test separation (may break existing CI)

**Mitigation**: Implement in phases with feature flags for gradual rollout

---

## Progress

This plan addresses the major testing issues identified through analysis of the 43 test files across the codebase. The focus is on eliminating duplication, improving reliability, and establishing consistent patterns that will scale as the project grows.

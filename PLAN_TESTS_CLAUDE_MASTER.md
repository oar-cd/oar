# Test Improvement Master Plan for Oar Project

This master plan consolidates insights from Claude Code, Codex, Amp, and Gemini agent analyses to create a comprehensive test refactoring strategy. The plan addresses identified code duplication, organizational issues, and coverage gaps across 40+ test files.

## Current State Analysis

### Test Structure Overview
- **43 test files** across multiple packages with ~150 distinct test functions
- **Test types**: Unit tests, integration tests, and mock-based tests with good separation
- **Test organization**: Generally follows Go conventions with proper `*_test.go` naming
- **Major strength**: Comprehensive mocking infrastructure and good use of testify

### Critical Issues Identified

#### 1. Code Duplication (High Priority)
- **Database setup**: `setupTestDB()` duplicated in `services/testutils_test.go:32-50` and `models/testutils_test.go:15-28`
- **Mock implementations**: `MockProjectRepository`, `MockGitExecutor` patterns repeated across packages
- **Test data creation**: `createTestProject()` vs `createTestProjectModel()` with similar but different patterns
- **Helper functions**: `stringPtr`, `generateTestKey` duplicated across test files
- **CLI command setup**: Repeated environment setup, temp directories, and stdout/stderr capture in `cmd/` tests

#### 2. Integration Test Organization (High Priority)
- Integration tests mixed with unit tests causing slow CI runs
- Missing build tags for Docker/git dependent tests
- No separation between fast unit tests (<5s) and slow integration tests (30s+)
- Time-based flakiness in watcher tests using `time.Sleep`

#### 3. Test Infrastructure Issues (Medium Priority)
- Test utilities in `_test.go` files aren't importable across packages
- Missing `t.Helper()` markers on helper functions
- Inconsistent mocking patterns (hand-rolled vs testify/mock)
- SQLite `:memory:` DSN causing issues with GORM connection pooling

#### 4. Coverage and Quality Gaps (Medium Priority)
- Web handlers tested individually but missing router-level integration tests
- CLI commands have basic tests but lack comprehensive scenario coverage
- Error handling tests are inconsistent (some check error messages, others just assert error existence)
- Missing concurrency and race condition testing

## Improvement Plan

### Phase 1: Test Infrastructure Consolidation (High Priority)

#### 1.1 Create Unified Test Utilities Package
**Location**: `internal/testutil/` with `//go:build test` build constraint

**New files to create**:
```
internal/testutil/
├── database.go     - Centralized DB setup with safe SQLite DSN
├── crypto.go       - Shared encryption key generation
├── builders/       - Test data builders for all entities
│   ├── project.go
│   └── deployment.go
├── helpers.go      - Common helpers (stringPtr, etc.)
├── http.go         - HTTP test utilities for web package
└── cli.go          - CLI command test harness
```

**Key functions**:
```go
// database.go
func SetupTestDB(t *testing.T) *gorm.DB
func SetupTestDBWithConfig(t *testing.T, config db.DBConfig) *gorm.DB

// builders/project.go
func NewProject(opts ...ProjectOption) *services.Project
func NewProjectModel(opts ...ProjectModelOption) *models.ProjectModel

// cli.go
func ExecuteCommand(cmd *cobra.Command, args []string) (stdout, stderr string, err error)
func SetupTestApp(t *testing.T) (*app.App, string) // returns app and temp dir

// http.go
func CreateFormRequest(method, url string, data map[string]string) *http.Request
func SetupTestRouter(t *testing.T) *chi.Mux
```

#### 1.2 Standardize Mock Implementations
**Location**: `internal/mocks/` with auto-generation using mockery

**Approach**:
```go
//go:generate mockery --name=GitExecutor --outpkg=mocks --output=internal/mocks --with-expecter
//go:generate mockery --name=ProjectManager --outpkg=mocks --output=internal/mocks --with-expecter
//go:generate mockery --name=ComposeProjectInterface --outpkg=mocks --output=internal/mocks --with-expecter
```

**Benefits**: Eliminates hand-rolled mock duplication, ensures interface compliance

#### 1.3 CLI Command Test Harness
**Problem**: Each CLI test file duplicates environment setup, temp directories, and output capture

**Solution**: Centralized harness in `internal/testutil/cli.go`
```go
type CommandTestHarness struct {
    app     *app.App
    tempDir string
    stdout  *bytes.Buffer
    stderr  *bytes.Buffer
}

func NewCommandHarness(t *testing.T) *CommandTestHarness
func (h *CommandTestHarness) Execute(cmd string, args ...string) error
func (h *CommandTestHarness) AssertOutput(t *testing.T, expectedStdout, expectedStderr string)
```

### Phase 2: Test Separation and Organization (High Priority)

#### 2.1 Integration Test Isolation
- Add `//go:build integration` tags to all Docker/git dependent tests
- Rename integration test files with `_integration_test.go` suffix
- Use local git repositories (file:// URLs) instead of network calls where possible

**Files to update**:
- `services/git_integration_test.go`
- `services/compose_integration_test.go`
- `services/project_manager_integration_test.go`

#### 2.2 Makefile Test Targets
```makefile
test:
	go test ./... -race -short

test-integration:
	go test ./... -race -tags=integration

test-all:
	make test && make test-integration

test-coverage:
	go test ./... -race -short -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
```

#### 2.3 Fix Time-Based Test Flakiness
**Problem**: Watcher tests use real `time.Sleep` making them flaky

**Solution**: Inject clock interface using `benbjohnson/clock` or similar
```go
type TickerFactory func(d time.Duration) Ticker
type Ticker interface {
    C() <-chan time.Time
    Stop()
}

// In tests, use fake clock instead of real time
```

### Phase 3: Database and SQLite Improvements (Medium Priority)

#### 3.1 Fix SQLite Connection Issues
**Current**: `:memory:` DSN causes issues with GORM pooling
**Solution**: Use `file::memory:?cache=shared&_fk=1` or temp file DBs for better stability

#### 3.2 Transaction Test Patterns
- Add tests for rollback scenarios
- Test constraint violations and error handling
- Verify migration edge cases and schema changes

### Phase 4: Enhanced Test Coverage (Medium Priority)

#### 4.1 Web Package Integration Tests
**Current**: Only unit tests for individual handler functions
**Add**: Router-level tests that exercise middleware, routing, and template rendering
```go
func TestProjectHandlerIntegration(t *testing.T) {
    router := setupTestRouter(t)
    // Test actual HTTP requests through chi router
    // Verify status codes, headers, response bodies
    // Test middleware execution and template rendering
}
```

#### 4.2 CLI Behavior and Golden File Tests
**Current**: Basic structure validation
**Add**:
- Comprehensive flag parsing and validation tests
- Golden file comparisons for command output
- Standardized `-update` flag for regenerating golden files
- Edge case testing for invalid inputs and error scenarios

#### 4.3 Error Path and Edge Case Testing
- Standardize error testing patterns across all packages
- Add concurrent access scenario testing
- Test resource cleanup and error recovery paths
- Add property-based testing for input validation

### Phase 5: Code Quality and Performance (Low Priority)

#### 5.1 Test Hygiene Improvements
- Add `t.Helper()` to all helper functions for better error reporting
- Use `t.Parallel()` for safe unit tests to speed up execution
- Standardize `require` vs `assert` usage patterns
- Convert repetitive tests to table-driven format where appropriate

#### 5.2 Concurrency and Race Condition Testing
- Add race condition tests for concurrent service operations
- Test watcher/service interactions under load
- Ensure `-race` flag is used consistently in CI

#### 5.3 Performance and Benchmark Testing
- Add benchmark tests for critical code paths
- Test memory usage patterns and resource leaks
- Add stress tests for high-load scenarios

## Implementation Strategy

### Phase 1 Actions (Week 1-2)
1. Create `internal/testutil/` package with shared utilities
2. Consolidate `setupTestDB` and move to testutil
3. Create shared crypto/helpers functions
4. Generate standard mocks using mockery
5. Build CLI command test harness

### Phase 2 Actions (Week 3-4)
1. Tag integration tests with build constraints
2. Update Makefile with separate test targets
3. Implement clock interface injection for watcher tests
4. Split oversized test files (`services/project_test.go` -> focused files)

### Phase 3 Actions (Week 5-6)
1. Fix SQLite connection stability issues
2. Add router-level web integration tests
3. Implement golden file testing for CLI commands
4. Add comprehensive error path coverage

### Phase 4 Actions (Week 7-8)
1. Add concurrency and race condition tests
2. Implement benchmark tests for performance monitoring
3. Add property-based testing for validation functions
4. Continuous test quality improvements

## Success Metrics

### Before Implementation
- 43 test files with significant code duplication
- Flaky time-based tests causing CI failures
- Mixed unit/integration tests causing 30s+ test runs
- 4+ different mock implementations for same interfaces
- Missing coverage for web routing and CLI edge cases

### After Implementation
- Zero code duplication in test utilities
- Deterministic tests with no time.Sleep dependencies
- Fast unit tests (<5s) separated from integration tests (tagged)
- Consistent mocking pattern using generated mocks across all packages
- Comprehensive test coverage including web integration and CLI golden files
- Race condition testing enabled with `-race` flag
- >90% test coverage for critical paths
- Maintainable test suite requiring minimal setup code for new tests

## Files That Will Be Modified/Created

### New Files
- `internal/testutil/{database,crypto,helpers,http,cli}.go`
- `internal/testutil/builders/{project,deployment}.go`
- `internal/mocks/*.go` (generated by mockery)
- `testdata/` directories with golden files

### Major Refactors
- `services/testutils_test.go` → Move utilities to shared package
- `models/testutils_test.go` → Move utilities to shared package
- `services/mocks_test.go` → Replace with generated mocks
- `watcher/watcher_test.go` → Use shared mocks + fake clock
- All `cmd/` test files → Use centralized CLI harness
- All integration tests → Add build tags
- `services/project_test.go` → Split into focused files

### Infrastructure Updates
- `Makefile` → Add separate test targets and coverage reporting
- `go.mod` → Add testing dependencies (mockery, clock library)
- CI configuration → Separate unit and integration test pipelines

This master plan provides a comprehensive roadmap for transforming the Oar project's test suite into a maintainable, fast, and robust testing foundation that will scale as the project grows.

## Progress

This master plan consolidates insights from all four coding agents and provides a prioritized, actionable roadmap for test improvement. The focus is on eliminating duplication, improving reliability, standardizing patterns, and establishing comprehensive coverage that will benefit long-term maintainability.

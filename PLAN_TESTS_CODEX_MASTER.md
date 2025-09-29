# Master Test Improvement Plan for Oar Project

## Harmonize Test Infrastructure
- Create a shared `internal/testsupport` (or similarly scoped) package that exposes reusable helpers for database setup, encryption key/fernet generation, git repo scaffolding, HTTP/form utilities, and pointer builders; migrate duplicated helpers from `services/testutils_test.go`, `models/testutils_test.go`, and `cmd/test` into it.
- Update the centralized DB helper to use a safe SQLite DSN (`file::memory:?cache=shared&_fk=1` or temp files) and ensure every helper is marked with `t.Helper()` and optional cleanup hooks.
- Provide builders for common fixtures (projects, deployments, config objects) under `internal/testsupport/builders` so services, web, and watcher tests share consistent data creation.

## Standardize Mocks and Harnesses
- Expand `testing/mocks` (or a new `internal/mocks`) to host canonical, generated mocks for interfaces such as `GitExecutor`, `ProjectManager`, `ComposeProjectInterface`, and `EnvProvider`, replacing hand-written mocks across packages.
- Add `//go:generate mockery --with-expecter` (or equivalent) directives so interface changes regenerate mocks in one place.
- Grow the `cmd/test` utilities into a full harness that initializes the app with default config/mocks, captures stdio, and supports golden comparisons; refactor all `cmd` suites to rely on it.

## Organize Suites and Execution Tiers
- Split monolithic suites like `services/project_test.go` into focused files grouped by behavior (list, create, deploy, etc.) and apply the same approach to other oversized tests.
- Enforce consistent naming (`*_test.go`, `*_integration_test.go`) and table-driven structure where multiple scenarios share logic.
- Isolate slow or environment-dependent suites with `//go:build integration` tags, guard them with capability checks (Docker socket, git binary), and mirror the separation in the Makefile (`make test`, `make test_integration`, `make test_all`).
- Document package-level responsibilities so unit, integration, and end-to-end tests live in predictable locations (`cmd/`, `services/`, `web/`, `watcher/`).

## Improve Determinism and Reliability
- Replace `time.Sleep` patterns (especially in watcher tests) with injected clock/ticker interfaces or a fake clock implementation to remove flakiness.
- Where feasible, run unit tests with `t.Parallel()` and ensure helpers call `t.Helper()` to give clear failure locations.
- Use `go-git` in-memory repositories or local temp repos for git-related tests to avoid network dependency.
- Adopt consistent error assertion helpers (e.g., `AssertErrorContains`) to keep tests resilient to minor message wording changes.

## Strengthen Coverage and Behavior Checks
- Add router-level HTTP tests that mount the real chi router to exercise middleware, routing, and template rendering end-to-end; expand web handler/component tests to cover templ rendering.
- Increase CLI coverage for flag parsing, output formatting, help text, and error flows using the shared harness and golden files.
- Deepen service coverage: extend `services/git_test.go` using in-memory repos (including auth paths), add integration tests that span project creation through compose deployment, and cover watcher error paths (project listing failures, git pull errors, context cancellation).
- Bolster database and config testing with transaction/rollback scenarios, constraint violations, config parsing edge cases, and concurrency/race-condition checks (including CI runs with `-race`).
- Introduce targeted property-based tests for validation-heavy logic and lightweight benchmarks for critical hot paths where regressions have performance impact.

## Manage Test Data and Goldens
- Centralize static assets (compose files, golden outputs, fixtures) under `testdata/` directories near their packages and ensure tests use those locations exclusively.
- Provide a shared `-update` flag (via the CLI harness or a helper) to regenerate golden files intentionally, and standardize assertions against goldens vs. ad-hoc `assert.Contains` checks.

## Documentation and Automation
- Update the testing guide (referenced from `AGENTS.md`) to explain new utilities, mock generation, test tiers, golden update workflow, and expectations for new tests.
- Align Makefile and CI targets with the new structure (fast unit tests by default, optional integration/benchmark/property suites) and ensure capability checks fail fast with clear guidance.
- Encourage incremental adoption by noting in contributor docs that new or refactored tests must use the shared infrastructure and adhere to the updated patterns.

# Test Suite Improvement Plan

## Current Observations
- Tests remain colocated with their packages, but heavy Docker/go-git integration suites (e.g. `services/compose_integration_test.go`, `services/git_integration_test.go`) sit beside fast unit tests. Relying only on `testing.Short()` means `go test ./...` still trips on missing Docker or git.
- Command tests under `cmd/project` duplicate environment setup (temp dirs, `app.InitializeWithConfig`, encryption key env vars) and roll their own stdout/stderr capture; the `cmd/test` helpers are underused and inconsistent.
- Test helpers are scattered: `setupTestDB`, Fernet key generators, and `stringPtr` appear in both `services/testutils_test.go` and `models/testutils_test.go`; multiple packages define their own mocks for project managers or git executors despite the `testing/mocks` module.
- Web tests like `web/routes/routes_test.go` and `web/actions/actions_test.go` stub handlers directly instead of exercising the real chi router, leaving middleware and wiring uncovered.
- `services/project_test.go` spans 600+ lines covering unrelated behaviors (list/get/create/deploy) while each scenario wires bespoke mocks, obscuring coverage gaps and encouraging more duplication.
- Golden output assertions vary—`cmd/project/add_test.go` renders templates on the fly, others rely on `assert.Contains`, and no shared golden update flag exists.

## Improvement Plan
1. **Shared Test Support Library**
   - Factor common helpers into an `internal/testsupport` (or similar) package that provides temp database setup, encryption keys, git repo scaffolding, and reusable fixtures. Migrate duplicated code from `services/testutils_test.go`, `models/testutils_test.go`, and `cmd/test/utils.go`.
   - Add HTTP/form helpers so web packages can import a single `testsupport` function instead of redefining `createFormRequest` or manual chi context wiring.
2. **Unify Mock Implementations**
   - Expand `testing/mocks` to host canonical `MockProjectManager`, `MockGitExecutor`, `MockDockerComposeExecutor`, etc., replacing ad-hoc mocks living in `services/mocks_test.go`, `watcher/watcher_test.go`, and other suites.
   - Ensure interface changes propagate by adjusting shared mocks in one place and updating dependent tests accordingly.
3. **CLI Command Test Harness**
   - Grow the `cmd/test` utilities into a harness that initializes the app with default config/mocks, captures command IO, and compares against golden outputs.
   - Refactor command suites (`add`, `list`, `status`, `deploy`, etc.) to rely on the harness rather than repeating env setup, buffer management, and custom assertions.
4. **Stratify Test Tiers & Execution Controls**
   - Introduce build tags (e.g. `//go:build integration`) for Docker/git dependent suites, pairing with new make targets (`make test_integration`) so the default test run stays fast and deterministic.
   - Add environment capability checks (Docker socket, git binary) that call `t.Skipf` when prerequisites are missing, complementing `testing.Short()` guards.
5. **Split Overgrown Suites**
   - Break `services/project_test.go` into focused files (`project_service_list_test.go`, `project_service_create_test.go`, `project_service_deploy_test.go`) that each import shared fixtures to reduce cognitive load and state leakage.
   - Apply the same pattern to other large suites such as `services/project_manager_integration_test.go` once shared support is available.
6. **Targeted Coverage Additions**
   - Web layer: add router-level tests that mount the chi router and exercise health/version/git-auth endpoints end-to-end, verifying middleware, template rendering, and redirects.
   - Watcher: using unified mocks, cover error paths (project listing failures, git pull errors, context cancellations) that are currently untested.
   - Logging/CLI flags: convert `assert.Contains` usage text checks into golden or structured assertions to detect regressions in help/usage output.
7. **Golden File & Fixture Hygiene**
   - Standardize on package-local `testdata/` directories with `*.golden` files for CLI and web output. Provide a shared `-update` flag in the harness to regenerate goldens intentionally.
   - Consolidate fixtures (compose files, git repos) under consistent directories so tests reference a single source of truth.
8. **Documentation & Automation**
   - Document new test tiers, harness usage, and golden update workflow in a testing guide linked from `AGENTS.md`.
   - Update CI pipelines to run unit tests by default and schedule integration suites separately, ensuring make targets align with automation.

## Recommended Next Steps
- Prioritize building the shared test support package and mock consolidation—these unlock the CLI harness refactor and suite splits.
- Track incremental progress in the appropriate plan/progress files per `CLAUDE.md`, refactoring one package at a time to keep diffs reviewable.
- After utilities land, migrate command suites first (highest duplication), then web/watcher, followed by integration-test tagging.


# Plan for Improving Tests in the Oar Project

This document outlines a plan to improve the testing strategy in the Oar project. The goal is to increase test coverage, reduce code duplication in tests, and make the test suite more maintainable and robust.

## 1. General Recommendations

*   **Test Data Management**: Centralize test data in `testdata` directories within each package. This includes golden files, compose files, and any other static data used in tests.
*   **Consistent Naming**: Ensure consistent naming for test files and functions. Test files should be named `[filename]_test.go`. Test functions should be named `Test[Module]_[Scenario]`.
*   **Table-Driven Tests**: Continue using table-driven tests for testing multiple scenarios with the same test logic. This is already well-utilized in many parts of the codebase.

## 2. `cmd` Package Improvements

The `cmd` package tests have a lot of duplicated code for setting up the test environment.

*   **Create a Test Helper**: Create a `cmd/test/helper.go` file to encapsulate the common test setup logic. This helper should provide functions for:
    *   Initializing a test application with a temporary data directory.
    *   Creating a temporary git repository with specified files.
    *   Executing a cobra command and capturing its output.
*   **Refactor `cmd` Tests**: Refactor the tests in the `cmd` package to use the new test helper. This will significantly reduce the amount of boilerplate code in each test file.

## 3. `services` Package Improvements

The `services` package has a good foundation of tests, but some areas can be improved.

*   **Enhance `git_test.go`**: The tests in `services/git_test.go` are currently quite basic. They should be expanded to:
    *   Use a temporary, in-memory git repository to test clone, pull, and commit operations without relying on the filesystem or network. The `go-git` library's in-memory storage is perfect for this.
    *   Add tests for authentication (both SSH and HTTP).
*   **Integration Tests**: Add more integration tests to verify the interaction between different services. For example, an integration test could cover the entire project creation and deployment flow, from the `ProjectService` to the `GitService` and `ComposeService`.

## 4. `web` Package Improvements

The `web` package tests currently focus on individual handlers and actions.

*   **HTTP-Level Tests**: Introduce tests that operate at the HTTP level to test the web application's routes and handlers in a more integrated way. The `net/http/httptest` package is well-suited for this.
*   **Component Tests**: For the `templ` components, consider adding tests that render the components and verify the generated HTML.

## 5. Weak Spots and Areas to Address

*   **Database Interactions**: The database tests are minimal. It would be beneficial to add more tests for the `db` package that cover different query scenarios and edge cases.
*   **Concurrency**: Some parts of the application, like the `watcher` package, deal with concurrency. Add tests that specifically target potential race conditions and other concurrency-related issues.
*   **Error Handling**: While error handling is generally good, there are places where errors are simply logged without being asserted in tests. Improve tests to ensure that the correct errors are returned in all failure scenarios.

## 6. Proposed Plan of Action

1.  **Implement `cmd/test/helper.go`**: Create the test helper for the `cmd` package as described above.
2.  **Refactor `cmd/project/add_test.go`**: Refactor this test to use the new helper as a proof-of-concept.
3.  **Refactor Remaining `cmd` Tests**: Apply the same refactoring to the other tests in the `cmd` package.
4.  **Enhance `services/git_test.go`**: Improve the git service tests to use an in-memory repository.
5.  **Add Integration Tests**: Add at least one new integration test for the `services` package.
6.  **Add HTTP Tests for `web` Package**: Add HTTP-level tests for the main routes in the `web` package.

By following this plan, the Oar project's test suite will be more comprehensive, maintainable, and effective at catching regressions.

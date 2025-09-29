# Master Plan for Test Refactoring

This document outlines a comprehensive plan for refactoring and improving the tests in the Oar project.

## I. Test Organization and Structure

1.  **Create a shared test utility package:** Create a new package, for example, `testing/testutil`, to house common test helper functions and data. This package can be used across all other packages.
2.  **Centralize test setup for `cmd` package:** Move the repetitive test setup code from `cmd/project/*_test.go` files into the new `testing/testutil` package. This includes functions for creating temporary directories, initializing mock git repositories, and setting up the app for testing.
3.  **Refactor `services` integration tests:** Extract the duplicated setup code from `services/project_manager_integration_test.go` and `services/compose_integration_test.go` into helper functions within the `services` package or the new `testing/testutil` package.
4.  **Separate unit and integration tests:** Use Go's build tags (e.g., `//go:build integration`) to separate unit tests from integration tests. This will allow running them independently (e.g., `go test -v ./...` for unit tests and `go test -v --tags=integration ./...` for integration tests).

## II. Code Duplication and Reusability

1.  **Create test data builders:** Implement the builder pattern for creating test data, such as `Project` and `Deployment` objects. This will make tests more readable and easier to maintain. For example, a `ProjectBuilder` could have methods like `WithName("my-project")`, `WithGitURL("...")`, etc.
2.  **Consolidate mock objects:** The `testing/mocks` directory is good. Ensure all mock objects are stored there and are up-to-date.
3.  **Create a test helper for running cobra commands:** Create a helper function that takes a `cobra.Command` and arguments, and returns the stdout, stderr, and error. This will reduce boilerplate in the `cmd` tests.

## III. Test Coverage and Quality

1.  **Increase test coverage for `cmd` package:**
    *   Add tests for all command-line flags, especially for the `project add` and `project update` commands (e.g., authentication flags).
    *   Add tests for different output formats (e.g., JSON output if available).
2.  **Improve `web` package tests:**
    *   Use `httptest` to test the full HTTP request-response cycle for all handlers.
    *   Render the HTML templates and use a library like `goquery` to assert on the HTML output.
    *   Test the SSE (Server-Sent Events) streaming endpoints to ensure they are sending the correct data.
3.  **Improve error handling assertions:**
    *   Instead of `assert.Error(t, err)`, use `assert.EqualError(t, err, "expected error message")` or `assert.True(t, errors.Is(err, expectedError))` to make tests more specific.
4.  **Add more test cases for edge conditions:**
    *   Test with invalid inputs (e.g., invalid UUIDs, empty strings, etc.).
    *   Test for race conditions in concurrent code.

## IV. Testing Best Practices

1.  **Use table-driven tests:** Continue using table-driven tests (`for _, tt := range tests`) to keep tests DRY and easy to extend.
2.  **Use `require` for setup and `assert` for checks:** Use `require` for assertions that should stop the test immediately if they fail (e.g., in test setup). Use `assert` for the actual test assertions.
3.  **Add comments to complex tests:** Add comments to explain the purpose of complex test cases or setup logic.

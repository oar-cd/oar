package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscovery_DiscoverComposeFiles(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "oar-discovery-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint: errcheck

	// Create test files
	testFiles := map[string]string{
		"docker-compose.yml": `services:
  web:
    image: nginx
    ports:
      - "80:80"
`,
		"compose.yml": `services:
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
`,
		"not-compose.yml": `# Just a regular YAML file
name: test
version: 1.0
`,
		"package.json": `{"name": "test"}`, // Non-YAML file
		"README.md":    "# Test project",   // Non-YAML file
	}

	// Create subdirectories to test
	os.MkdirAll(filepath.Join(tempDir, "subdir"), 0o755)       // nolint: errcheck
	os.MkdirAll(filepath.Join(tempDir, ".git"), 0o755)         // nolint: errcheck
	os.MkdirAll(filepath.Join(tempDir, "node_modules"), 0o755) // nolint: errcheck

	// Write test files
	for filename, content := range testFiles {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Write a compose file in subdirectory
	subComposeContent := `services:
  redis:
    image: redis:alpine
`
	err = os.WriteFile(filepath.Join(tempDir, "subdir", "docker-compose.dev.yml"), []byte(subComposeContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create subdirectory compose file: %v", err)
	}

	// Write a file in a directory that should be skipped
	err = os.WriteFile(filepath.Join(tempDir, ".git", "config"), []byte("gitconfig"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create git config file: %v", err)
	}

	// Test the discovery
	discovery := NewProjectDiscoveryService(nil, &Config{})
	results, err := discovery.DiscoverComposeFiles(tempDir)
	if err != nil {
		t.Fatalf("DiscoverComposeFiles failed: %v", err)
	}

	// Verify results
	if len(results) != 3 {
		t.Errorf("Expected 3 compose files, got %d", len(results))
		for _, result := range results {
			t.Logf("Found: %s", result.Path)
		}
	}

	// Check that expected files are found
	expectedFiles := []string{"docker-compose.yml", "compose.yml", "subdir/docker-compose.dev.yml"}
	for _, expected := range expectedFiles {
		found := false
		for _, result := range results {
			if result.Path == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file %s was not found", expected)
		}
	}
}

func TestDiscovery_shouldSkipDir(t *testing.T) {
	discovery := NewProjectDiscoveryService(nil, &Config{})

	testCases := []struct {
		dir      string
		expected bool
	}{
		{".git", true},
		{".github", true},
		{".vscode", true},
		{".hidden", true},
		{"src", false},
		{"lib", false},
		{"node_modules", false}, // Not hidden, so should be processed (if not gitignored)
		{"vendor", false},       // Not hidden, so should be processed (if not gitignored)
	}

	for _, tc := range testCases {
		result := discovery.shouldSkipDir(tc.dir)
		if result != tc.expected {
			t.Errorf("shouldSkipDir(%q) = %v, expected %v", tc.dir, result, tc.expected)
		}
	}
}

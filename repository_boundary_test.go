package surfacecontract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractTestsDoNotNameProviderRepositories(t *testing.T) {
	prefix := strings.Join([]string{"soksak", "sidecar", "terminal"}, "-") + "-"
	entries, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range entries {
		if path == "repository_boundary_test.go" {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(source), prefix) {
			t.Errorf("%s names a provider repository; contract tests use neutral fixtures", path)
		}
	}
}

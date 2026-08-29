package surfacecontract

import (
	"os"
	"strings"
	"testing"
)

func TestMakeOwnsSurfaceContractCommands(t *testing.T) {
	body, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	for _, target := range []string{"preflight:", "prepare:", "build:", "verify:"} {
		if !strings.Contains(source, target) {
			t.Errorf("Makefile omits %s", target)
		}
	}
	if strings.Contains(source, "GO_VERSION :=") {
		t.Error("Makefile duplicates Go metadata")
	}
	if strings.Contains(source, "PATH=") || strings.Contains(source, "$$HOME/.cargo") {
		t.Error("Makefile overrides the login-shell tool selection")
	}
}

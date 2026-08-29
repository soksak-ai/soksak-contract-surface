package surfacecontract

import (
	"os"
	"strings"
	"testing"
)

func TestSelectionContractDeclaresTheWholeGestureState(t *testing.T) {
	spec, err := os.ReadFile("SPEC.md")
	if err != nil {
		t.Fatal(err)
	}
	commands, err := os.ReadFile("commands.go")
	if err != nil {
		t.Fatal(err)
	}
	joined := string(spec) + "\n" + string(commands)
	for _, required := range []string{
		`action: "read"`, `action: "clear"`, `action: "gesture"`,
		`phase`, `kind`, `point`, `side`, `modifiers`, `SelectionSnapshot`,
	} {
		if !strings.Contains(joined, required) {
			t.Errorf("surface.selection omits %s", required)
		}
	}
	if strings.Contains(string(spec), "from{row, col}, to{row, col}") {
		t.Error("surface.selection still declares the replaced from/to request")
	}
}

func TestRustHalfOwnsTheSameStrictSelectionDecoder(t *testing.T) {
	rust, err := os.ReadFile("src/lib.rs")
	if err != nil {
		t.Fatal(err)
	}
	source := string(rust)
	for _, required := range []string{"decode_selection_request", "decode_selection_snapshot", "deny_unknown_fields"} {
		if !strings.Contains(source, required) {
			t.Errorf("Rust selection contract omits %s", required)
		}
	}
}

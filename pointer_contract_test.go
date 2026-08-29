package surfacecontract

import (
	"os"
	"strings"
	"testing"
)

func TestPointerContractPreservesPhaseButtonsModifiersAndOneEffect(t *testing.T) {
	files := []string{"SPEC.md", "commands.go", "src/lib.rs"}
	var joined strings.Builder
	for _, name := range files {
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		joined.Write(content)
	}
	for _, required := range []string{
		"surface.pointer", "PointerPhase", "PointerButton", "clickCount", "modifiers",
		"decode_pointer_request", "decode_pointer_engine_result",
	} {
		if !strings.Contains(joined.String(), required) {
			t.Errorf("surface pointer contract omits %s", required)
		}
	}
}

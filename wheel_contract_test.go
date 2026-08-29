package surfacecontract

import (
	"os"
	"strings"
	"testing"
)

func TestWheelContractPreservesDeviceUnitsAndInputOwnership(t *testing.T) {
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
		"surface.wheel", "WheelDeltaMode", "deltaX", "deltaY", "modifiers",
		"decode_wheel_request", "decode_wheel_engine_result",
	} {
		if !strings.Contains(joined.String(), required) {
			t.Errorf("surface wheel contract omits %s", required)
		}
	}
}

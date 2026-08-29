package surfacecontract

import (
	"encoding/base64"
	"testing"
)

func pointerRequest() map[string]any {
	return map[string]any{
		"window": "win-a", "pane": "tab-a.1",
		"point": map[string]any{"x": float64(12), "y": float64(24)},
		"phase": "down", "button": "left", "clickCount": float64(1),
		"modifiers": map[string]any{"shift": false, "alt": false, "control": false, "meta": false},
	}
}

func TestPointerRequestIsExactAndDistinguishesMotionWithoutAButton(t *testing.T) {
	request, err := ValidatePointer(pointerRequest())
	if err != nil || request.Phase != PointerDown || request.Button != PointerLeft {
		t.Fatalf("pointer request = %#v, %v", request, err)
	}
	move := pointerRequest()
	move["phase"], move["button"], move["clickCount"] = "move", "none", float64(0)
	if _, err := ValidatePointer(move); err != nil {
		t.Fatal(err)
	}
	invalid := pointerRequest()
	invalid["button"] = "none"
	if _, err := ValidatePointer(invalid); err == nil {
		t.Fatal("buttonless press was accepted")
	}
	partial := pointerRequest()
	delete(partial["modifiers"].(map[string]any), "meta")
	if _, err := ValidatePointer(partial); err == nil {
		t.Fatal("partial pointer modifiers were accepted")
	}
}

func TestPointerEngineResultHasOneEffect(t *testing.T) {
	input := base64.StdEncoding.EncodeToString([]byte("\x1b[<0;2;3M"))
	if _, err := ValidatePointerEngineResult(map[string]any{
		"route": "mouse-report", "dataB64": input,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidatePointerEngineResult(map[string]any{
		"route": "ignored", "dataB64": input,
	}); err == nil {
		t.Fatal("ignored pointer answer retained PTY input")
	}
}

package surfacecontract

import (
	"encoding/base64"
	"testing"
)

func wheelRequest() map[string]any {
	return map[string]any{
		"window": "win-a", "pane": "tab-a.1",
		"point":  map[string]any{"x": float64(12), "y": float64(24)},
		"deltaX": float64(0), "deltaY": float64(36), "deltaMode": "pixel",
		"modifiers": map[string]any{"shift": false, "alt": false, "control": false, "meta": false},
	}
}

func TestWheelRequestIsExactAndKeepsItsUnit(t *testing.T) {
	request, err := ValidateWheel(wheelRequest())
	if err != nil {
		t.Fatal(err)
	}
	if request.DeltaMode != WheelDeltaPixel || request.DeltaY != 36 || request.Point.X != 12 {
		t.Fatalf("wheel request = %#v", request)
	}
	for name, mutate := range map[string]func(map[string]any){
		"missing modifier": func(value map[string]any) { delete(value["modifiers"].(map[string]any), "meta") },
		"unknown field":    func(value map[string]any) { value["provider"] = "alacritty" },
		"empty delta":      func(value map[string]any) { value["deltaY"] = float64(0) },
		"unknown unit":     func(value map[string]any) { value["deltaMode"] = "device" },
	} {
		t.Run(name, func(t *testing.T) {
			value := wheelRequest()
			mutate(value)
			if _, err := ValidateWheel(value); err == nil {
				t.Fatalf("invalid wheel request was accepted: %#v", value)
			}
		})
	}
}

func TestWheelEngineResultDeclaresOneRoute(t *testing.T) {
	input := base64.StdEncoding.EncodeToString([]byte("\x1b[<64;2;3M"))
	for _, answer := range []map[string]any{
		{"route": "scrollback", "offset": float64(3), "historySize": float64(20), "dataB64": nil},
		{"route": "mouse-report", "offset": nil, "historySize": nil, "dataB64": input},
		{"route": "alternate-scroll", "offset": nil, "historySize": nil, "dataB64": input},
		{"route": "ignored", "offset": nil, "historySize": nil, "dataB64": nil},
	} {
		if _, err := ValidateWheelEngineResult(answer); err != nil {
			t.Fatalf("valid wheel answer %#v: %v", answer, err)
		}
	}
	invalid := map[string]any{
		"route": "scrollback", "offset": float64(3), "historySize": float64(20), "dataB64": input,
	}
	if _, err := ValidateWheelEngineResult(invalid); err == nil {
		t.Fatal("scrollback answer retained PTY input")
	}
}

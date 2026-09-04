package surfacecontract

import (
	"slices"
	"testing"
)

// A dim is what the surface draws, not how much of the document shows through it.
//
// Measured 2026-09-04: a dimmed surface was declared at alpha 1-dim, so the document behind it was
// visible through it. The picture that stands in for a parked surface is drawn in that document, so
// staging it under the surface brightened the pane by 0.5×(191−127) for two frames, and taking the
// surface off before the picture was drawn darkened it for two more. An opaque surface has neither
// frame: what is behind it is never on screen, so the picture waits there unseen.
func TestDimIsANamedCommand(t *testing.T) {
	if !slices.Contains(CommandNames(), "surface.dim") {
		t.Fatal("surface.dim is not a named command")
	}
}

func TestValidateDim(t *testing.T) {
	request, err := ValidateDim(map[string]any{
		"window": "win-a", "pane": "tab-a.1", "dim": 0.5,
	})
	if err != nil {
		t.Fatalf("a valid request is refused: %v", err)
	}
	if request.Window != "win-a" || request.Pane != "tab-a.1" || request.Dim != 0.5 {
		t.Fatalf("the request read back as %+v", request)
	}

	for name, bad := range map[string]map[string]any{
		"an amount above one":  {"window": "win-a", "pane": "tab-a.1", "dim": 1.5},
		"an amount below zero": {"window": "win-a", "pane": "tab-a.1", "dim": -0.1},
		"a missing amount":     {"window": "win-a", "pane": "tab-a.1"},
		"an extra key":         {"window": "win-a", "pane": "tab-a.1", "dim": 0.5, "alpha": 1},
	} {
		if _, err := ValidateDim(bad); err == nil {
			t.Fatalf("%s is accepted", name)
		}
	}
}

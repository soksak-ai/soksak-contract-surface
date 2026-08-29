package surfacecontract

import "testing"

func TestFocusTransactionPreservesTheOwnerAndPresentationPolicy(t *testing.T) {
	request, err := ValidateFocus(map[string]any{
		"window": "win-a", "pane": "tab-a.1", "focused": false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.Window != "win-a" || request.Pane != "tab-a.1" || request.Focused {
		t.Fatalf("request=%+v", request)
	}
	result, err := ValidateFocusEngineResult(map[string]any{
		"focused": false, "cursorPresentation": "hollow-block",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Focused || result.CursorPresentation != CursorHollowBlock {
		t.Fatalf("result=%+v", result)
	}
}

func TestFocusTransactionRejectsUnknownOrContradictoryFields(t *testing.T) {
	if _, err := ValidateFocus(map[string]any{
		"window": "win-a", "pane": "tab-a.1", "focused": true, "retry": true,
	}); err == nil {
		t.Fatal("surface.focus accepted an unknown field")
	}
	if _, err := ValidateFocusEngineResult(map[string]any{
		"focused": true, "cursorPresentation": "hollow-block",
	}); err == nil {
		t.Fatal("focused engine answer selected the inactive cursor presentation")
	}
}

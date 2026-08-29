package surfacecontract

import "testing"

func gestureRequest() map[string]any {
	return map[string]any{
		"window": "win-a", "pane": "tab-a.1", "action": "gesture", "gestureId": "sel-1",
		"phase": "begin", "kind": "simple",
		"point": map[string]any{"row": float64(2), "col": float64(3), "side": "right"},
		"modifiers": map[string]any{
			"shift": false, "alt": false, "control": false, "meta": false,
		},
	}
}

func TestSelectionRequestUnionIsExact(t *testing.T) {
	read, err := ValidateSelection(map[string]any{"window": "win-a", "pane": "tab-a.1", "action": "read"})
	if err != nil || read.Action != SelectionRead {
		t.Fatalf("read selection = %#v, %v", read, err)
	}
	gesture, err := ValidateSelection(gestureRequest())
	if err != nil {
		t.Fatal(err)
	}
	if gesture.GestureID != "sel-1" || gesture.Phase != SelectionBegin || gesture.Kind != SelectionSimple ||
		gesture.Point == nil || gesture.Point.Row != 2 || gesture.Point.Col != 3 || gesture.Point.Side != CellRight {
		t.Fatalf("gesture selection = %#v", gesture)
	}
	missingOwner := gestureRequest()
	delete(missingOwner, "window")
	if _, err := ValidateSelection(missingOwner); err == nil {
		t.Fatal("selection without its window owner was accepted")
	}

	old := map[string]any{
		"pane": "tab-a.1", "action": "gesture",
		"from": map[string]any{"row": float64(0), "col": float64(0)},
		"to":   map[string]any{"row": float64(0), "col": float64(1)},
	}
	if _, err := ValidateSelection(old); err == nil {
		t.Fatal("replaced from/to selection request was accepted")
	}
	missing := gestureRequest()
	delete(missing["modifiers"].(map[string]any), "meta")
	if _, err := ValidateSelection(missing); err == nil {
		t.Fatal("partial modifier state was accepted")
	}
}

func TestSelectionSnapshotIsCompleteAndVersioned(t *testing.T) {
	answer := map[string]any{
		"active": true, "text": "selected", "kind": "simple", "gestureId": "sel-1",
		"anchor":   map[string]any{"row": float64(1), "col": float64(2), "side": "left"},
		"focus":    map[string]any{"row": float64(3), "col": float64(4), "side": "right"},
		"sequence": float64(7),
	}
	snapshot, err := ValidateSelectionSnapshot(answer)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Active || snapshot.Sequence != 7 || snapshot.Text != "selected" ||
		snapshot.GestureID == nil || *snapshot.GestureID != "sel-1" {
		t.Fatalf("selection snapshot = %#v", snapshot)
	}

	inactive := map[string]any{
		"active": false, "text": "", "kind": nil, "anchor": nil, "focus": nil,
		"gestureId": nil, "sequence": float64(8),
	}
	if _, err := ValidateSelectionSnapshot(inactive); err != nil {
		t.Fatal(err)
	}
	inactive["gestureId"] = "stale"
	if _, err := ValidateSelectionSnapshot(inactive); err == nil {
		t.Fatal("inactive selection retained a gesture owner")
	}
	inactive["gestureId"] = nil
	inactive["sequence"] = float64(1 << 53)
	if _, err := ValidateSelectionSnapshot(inactive); err == nil {
		t.Fatal("selection accepted a sequence outside JavaScript's safe integer range")
	}
}

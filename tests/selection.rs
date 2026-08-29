use serde_json::json;
use soksak_contract_surface::{
    decode_selection_request, decode_selection_snapshot, encode_selection_snapshot, CellSide,
    SelectionKind, SelectionPhase, SelectionRequest,
};

#[test]
fn request_union_rejects_partial_and_replaced_shapes() {
    let read = decode_selection_request(&json!({"pane":"tab-a.1","action":"read"})).unwrap();
    assert!(matches!(read, SelectionRequest::Read { pane } if pane == "tab-a.1"));

    let gesture = decode_selection_request(&json!({
        "pane":"tab-a.1", "action":"gesture", "gestureId":"sel-1",
        "phase":"begin", "kind":"simple",
        "point":{"row":2,"col":3,"side":"right"},
        "modifiers":{"shift":false,"alt":false,"control":false,"meta":false}
    }))
    .unwrap();
    assert!(matches!(
        gesture,
        SelectionRequest::Gesture {
            phase: SelectionPhase::Begin,
            kind: SelectionKind::Simple,
            point: soksak_contract_surface::SelectionPoint {
                row: 2,
                col: 3,
                side: CellSide::Right
            },
            ..
        }
    ));

    assert!(decode_selection_request(&json!({
        "pane":"tab-a.1", "action":"gesture",
        "from":{"row":0,"col":0}, "to":{"row":0,"col":1}
    }))
    .is_err());
    assert!(decode_selection_request(&json!({
        "pane":"tab-a.1", "action":"gesture", "gestureId":"sel-1",
        "phase":"begin", "kind":"simple",
        "point":{"row":2,"col":3,"side":"right"},
        "modifiers":{"shift":false,"alt":false,"control":false}
    }))
    .is_err());
}

#[test]
fn snapshot_round_trip_enforces_active_and_inactive_shapes() {
    let value = json!({
        "active":true, "text":"selected", "kind":"simple", "gestureId":"sel-1",
        "anchor":{"row":1,"col":2,"side":"left"},
        "focus":{"row":3,"col":4,"side":"right"}, "sequence":7
    });
    let snapshot = decode_selection_snapshot(&value).unwrap();
    assert_eq!(encode_selection_snapshot(&snapshot).unwrap(), value);

    let inactive = json!({
        "active":false, "text":"", "kind":null, "anchor":null, "focus":null,
        "gestureId":null, "sequence":8
    });
    assert_eq!(
        encode_selection_snapshot(&decode_selection_snapshot(&inactive).unwrap()).unwrap(),
        inactive
    );
    assert!(decode_selection_snapshot(&json!({
        "active":false, "text":"stale", "kind":null, "anchor":null, "focus":null,
        "gestureId":null, "sequence":8
    }))
    .is_err());
    assert!(decode_selection_snapshot(&json!({
        "active":false, "text":"", "kind":null, "anchor":null, "focus":null,
        "gestureId":null, "sequence":8, "extra":true
    }))
    .is_err());
}

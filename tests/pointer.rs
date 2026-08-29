use base64::Engine as _;
use soksak_contract_surface::{
    decode_pointer_engine_result, decode_pointer_request, PointerButton, PointerPhase, PointerRoute,
};
use serde_json::json;

#[test]
fn pointer_request_keeps_phase_button_and_modifiers() {
    let request = decode_pointer_request(&json!({
        "window": "win-a", "pane": "tab-a.1",
        "point": {"x": 12.0, "y": 24.0},
        "phase": "down", "button": "left", "clickCount": 1,
        "modifiers": {"shift": false, "alt": false, "control": false, "meta": false}
    })).unwrap();
    assert_eq!(request.phase, PointerPhase::Down);
    assert_eq!(request.button, PointerButton::Left);
    assert!(decode_pointer_request(&json!({
        "window": "win-a", "pane": "tab-a.1",
        "point": {"x": 12.0, "y": 24.0},
        "phase": "up", "button": "none", "clickCount": 1,
        "modifiers": {"shift": false, "alt": false, "control": false, "meta": false}
    })).is_err());
}

#[test]
fn pointer_engine_result_has_one_effect() {
    let data = base64::engine::general_purpose::STANDARD.encode(b"\x1b[<0;2;3M");
    let result = decode_pointer_engine_result(&json!({
        "route": "mouse-report", "dataB64": data
    })).unwrap();
    assert_eq!(result.route, PointerRoute::MouseReport);
    assert!(decode_pointer_engine_result(&json!({
        "route": "ignored", "dataB64": data
    })).is_err());
}

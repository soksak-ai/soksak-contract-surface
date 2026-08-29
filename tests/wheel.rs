use base64::Engine as _;
use soksak_contract_surface::{
    decode_wheel_engine_result, decode_wheel_request, WheelDeltaMode, WheelRoute,
};
use serde_json::json;

#[test]
fn wheel_request_preserves_units_and_modifiers() {
    let request = decode_wheel_request(&json!({
        "window": "win-a", "pane": "tab-a.1",
        "point": {"x": 12.0, "y": 24.0},
        "deltaX": 0.0, "deltaY": 36.0, "deltaMode": "pixel",
        "modifiers": {"shift": false, "alt": false, "control": false, "meta": false}
    })).unwrap();
    assert_eq!(request.delta_mode, WheelDeltaMode::Pixel);
    assert_eq!(request.delta_y, 36.0);
}

#[test]
fn wheel_engine_result_has_one_effect_route() {
    let data = base64::engine::general_purpose::STANDARD.encode(b"\x1b[<64;2;3M");
    let result = decode_wheel_engine_result(&json!({
        "route": "mouse-report", "offset": null, "historySize": null, "dataB64": data
    })).unwrap();
    assert_eq!(result.route, WheelRoute::MouseReport);
    assert!(decode_wheel_engine_result(&json!({
        "route": "scrollback", "offset": 2, "historySize": 20, "dataB64": data
    })).is_err());
}

use soksak_contract_surface::{
    decode_focus_engine_result, decode_focus_request, CursorPresentation,
};
use serde_json::json;

#[test]
fn focus_transaction_is_closed_and_owner_bound() {
    let request = decode_focus_request(&json!({
        "window": "win-a", "pane": "tab-a.1", "focused": false
    })).expect("focus request");
    assert_eq!(request.window, "win-a");
    assert_eq!(request.pane, "tab-a.1");
    assert!(!request.focused);

    let result = decode_focus_engine_result(&json!({
        "focused": false, "cursorPresentation": "hollow-block"
    })).expect("focus result");
    assert_eq!(result.cursor_presentation, CursorPresentation::HollowBlock);

    assert!(decode_focus_request(&json!({
        "window": "win-a", "pane": "tab-a.1", "focused": true, "retry": true
    })).is_err());
    assert!(decode_focus_engine_result(&json!({
        "focused": true, "cursorPresentation": "hollow-block"
    })).is_err());
}

use serde_json::json;
use soksak_contract_surface::{decode_measure_request, decode_measure_result};

#[test]
fn measure_defines_the_grid_before_a_process_starts() {
    let request = decode_measure_request(&json!({
        "pixelW": 989.0, "pixelH": 468.0, "scale": 2.0,
        "font": { "family": "Menlo", "pt": 13.0 }
    })).expect("measure request");
    assert_eq!(request.pixel_w, 989.0);
    assert_eq!(request.font.family, "Menlo");

    let result = decode_measure_result(&json!({
        "cols": 126, "rows": 30, "cellW": 15.6875, "cellH": 31.2
    })).expect("measure result");
    assert_eq!((result.cols, result.rows), (126, 30));

    assert!(decode_measure_request(&json!({
        "pixelW": 989.0, "pixelH": 468.0, "scale": 2.0,
        "font": { "family": "Menlo", "pt": 13.0 }, "pane": "tab-a.1"
    })).is_err());
    assert!(decode_measure_result(&json!({
        "cols": 0, "rows": 30, "cellW": 15.6875, "cellH": 31.2
    })).is_err());
}

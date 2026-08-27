// Both halves speak the committed bytes: every fixture decodes, re-encodes to the identical
// bytes, and a foreign magic or version is refused by name.
use soksak_contract_surface::{decode, encode, Message};

fn fixture(name: &str) -> Vec<u8> {
    std::fs::read(format!("{}/testdata/messages/{name}.bin", env!("CARGO_MANIFEST_DIR"))).unwrap()
}

#[test]
fn every_fixture_round_trips_to_identical_bytes() {
    for name in ["hello", "ring", "frameReady", "released", "gap", "ended"] {
        let wire = fixture(name);
        let message = decode(&wire).unwrap_or_else(|error| panic!("{name}: {error}"));
        let again = encode(&message).unwrap_or_else(|error| panic!("{name}: {error}"));
        assert_eq!(again, wire, "{name}: re-encoded bytes differ");
    }
}

#[test]
fn the_fixture_fields_are_the_canonical_set() {
    match decode(&fixture("frameReady")).unwrap() {
        Message::FrameReady { pane, ring_index, seq, cursor_row, cursor_col, cursor_visible, damage } => {
            assert_eq!(pane, "tab-abc123.1");
            assert_eq!(ring_index, 2);
            assert_eq!(seq, 42);
            assert_eq!((cursor_row, cursor_col, cursor_visible), (3, 7, true));
            assert_eq!(damage, vec![(0, 3, 80, 1), (4, 9, 2, 2)]);
        }
        other => panic!("frameReady decoded as {other:?}"),
    }
    match decode(&fixture("ring")).unwrap() {
        Message::Ring { pane, pixel_w, pixel_h, scale, cell_w, cell_h } => {
            assert_eq!(pane, "tab-abc123.1");
            assert_eq!((pixel_w, pixel_h), (1280, 720));
            assert_eq!((scale, cell_w, cell_h), (2.0, 14.5, 29.0));
        }
        other => panic!("ring decoded as {other:?}"),
    }
}

#[test]
fn foreign_magic_and_version_are_refused_by_name() {
    let mut bad = fixture("gap");
    bad[0] ^= 0xff;
    let error = decode(&bad).unwrap_err();
    assert!(error.contains("magic"), "{error}");
    let mut bad = fixture("gap");
    bad[4] = 99;
    let error = decode(&bad).unwrap_err();
    assert!(error.contains("version"), "{error}");
}

#[test]
fn port_counts_are_declared_per_kind() {
    assert_eq!(decode(&fixture("hello")).unwrap().port_count(), 1);
    assert_eq!(decode(&fixture("ring")).unwrap().port_count(), 3);
    for name in ["frameReady", "released", "gap", "ended"] {
        assert_eq!(decode(&fixture(name)).unwrap().port_count(), 0, "{name}");
    }
}

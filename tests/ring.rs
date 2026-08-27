// The ring rule of SPEC.md section 4: render only into a free surface, never release the
// displayed one.
use soksak_contract_surface::RingState;

#[test]
fn never_renders_into_an_unreleased_surface() {
    let mut ring = RingState::new();
    let first = ring.acquire_for_render().unwrap();
    ring.signal(first);
    ring.display(first).unwrap();
    let second = ring.acquire_for_render().unwrap();
    ring.signal(second);
    let third = ring.acquire_for_render().unwrap();
    ring.signal(third);
    assert!(ring.acquire_for_render().is_err(), "a fourth target while nothing was released");
    ring.display(second).unwrap();
    ring.release(first).unwrap();
    assert!(ring.acquire_for_render().is_ok(), "a released surface is reusable");
}

#[test]
fn refuses_releasing_the_displayed_surface() {
    let mut ring = RingState::new();
    let index = ring.acquire_for_render().unwrap();
    ring.signal(index);
    ring.display(index).unwrap();
    assert!(ring.release(index).is_err(), "the displayed surface was released");
}

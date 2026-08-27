//! The Rust half of soksak-contract-surface. SPEC.md is normative; this crate holds the channel
//! name derivation, the message codec and the ring state machine a render sidecar implements
//! against. The Go half beside it serves the application, and both decode the same committed
//! wire fixtures.

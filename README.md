# soksak-contract-surface

Public contract for sidecar-rendered terminal surfaces.

Contract id: **`soksak-spec-sidecar-surface`**.

A render sidecar owns a terminal grid and paints it into an IOSurface ring. The application
composites the surface into its window and forwards input. This contract defines the surface
channel (a bootstrap-derived mach service), the ring hand-off, the frame-ready and release
messages, and the `surface.*` command payloads that ride the control envelope. It contains no
transport or renderer implementation. See [SPEC.md](SPEC.md) for the normative boundary.

The channel carries mach ports, so it exists only on darwin. Every other platform fails by name;
Windows (DXGI shared handles) and Linux (dmabuf) arrive as their own channel sections when a
backend exists for them.

## Two halves, one wire

The Go package serves the application side; the Rust crate serves the render sidecar side. Both
encode and decode the committed bytes under `testdata/messages/`, so a layout change that forgets
one side fails a test here rather than on a live channel. The ring state machine exists in both
languages and holds the same rule.

## Verification

```sh
make verify
```

Regenerating the wire fixtures after a deliberate layout change:

```sh
go test -run TestWireFixtures -update ./...
```

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

## Verification

```sh
make verify
```

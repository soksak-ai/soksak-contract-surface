# soksak-contract-surface — sidecar-rendered terminal surfaces

Contract id: **`soksak-spec-sidecar-surface`**, version **0.0.2**.

A render sidecar owns a terminal grid mirror and paints it. The application owns the window and
the input devices. This contract is the seam between them: an IOSurface ring the sidecar fills,
a mach channel that carries the ring and its frame signals, and `surface.*` commands that ride
the versioned control envelope. This repository implements nothing and ships nothing.

## 1. Purpose and boundary

- **The render sidecar (owns pixels):** the grid mirror, glyph rasterization, the Metal pipeline,
  the IOSurface ring, damage tracking, preedit/selection/hover overlays, cell metrics. It computes
  `cols`/`rows` from the pixel box the application hands it.
- **The application (owns the window):** composites one surface of the ring into its layer tree,
  captures keyboard/IME/mouse/scroll, forwards input bytes to the PTY, reports geometry, reads
  surface pixels for parking. It holds no cell, no glyph and no atlas.

A frame never crosses this seam as cells. What crosses is a surface identity (a mach port) once
per ring, and a fixed-size signal per painted frame.

## 2. The channel

- The application derives one service name per installation: `<identifier>.surface`, where
  `<identifier>` is the installation identifier that already names the control socket. The
  sidecar process holds no identifier of its own — `surface.open` carries it, and both halves
  derive the same name from it. The name
  must stay within the 128-byte bootstrap limit; a longer identifier is refused by name.
- The application `bootstrap_check_in`s the name (receive right). A sidecar `bootstrap_look_up`s
  it (send right). Children inherit the bootstrap namespace from the process that spawned them,
  which is the application's sidecar host.
- The first message a sidecar sends is `hello`, and `hello` carries a reply port. Every
  application-to-sidecar channel message travels on that reply port, so the channel is
  bidirectional after `hello` and nothing polls.
- The channel exists on darwin. Every other platform fails by name; a Windows (DXGI shared
  handle) or Linux (dmabuf) channel arrives as its own section when a backend exists.

## 3. Channel messages

### The mach packet

- A packet's payload is exactly one wire message — no extra prefix. A message
  with no rights carries no body word: its bytes start right after the mach
  header. A message with rights is complex: body, then the descriptors, then
  the bytes.
- Mach sizes are 4-byte aligned, so a receiver trims the padding with the
  wire's own length (`WireLength`/`wire_length`) before decoding.


Every message starts with a fixed header: `magic u32 'sksf'`, `version u8 = 1`, `kind u8`,
`payloadLen u16`, all big-endian. A message whose magic or version differs is refused by name.
Pane keys travel as `len u8` + UTF-8 bytes and follow the pane-key grammar of the terminal
plugin contract.

| kind | direction | payload |
| --- | --- | --- |
| 1 hello | sidecar → app | `sidecarIdLen u8, sidecarId`, one mach send right (the reply port) |
| 2 ring | sidecar → app | pane, `pixelW u32, pixelH u32, scale f64, cellW f64, cellH f64`, three IOSurface send rights, ring order 0..2 |
| 3 frameReady | sidecar → app | pane, `ringIndex u8, seq u64, cursorRow u16, cursorCol u16, cursorVisible u8, damageCount u8`, damage rects `x,y,w,h u16` each |
| 4 released | app → sidecar | pane, `ringIndex u8` |
| 5 gap | sidecar → app | pane — the mirror lost source continuity; recovery follows the terminal contract |
| 6 ended | sidecar → app | pane, `reasonLen u8, reason` |

## 4. The ring rule

Three surfaces per pane. Each is in exactly one state: `free`, `rendering`, `signaled`,
`displayed`.

- The sidecar renders only into a `free` surface, moves it to `signaled` with `frameReady`, and
  never touches it again until the application releases it.
- The application displays the most recent `signaled` surface, releases the surface it displayed
  before (`released`), and never releases the one currently displayed.
- A `resize` or a reconnect invalidates the ring: the sidecar sends a new `ring` message and the
  application releases every old index. Old surfaces die with the ring; the sidecar owns their
  lifetime.

The state machine is part of this contract: an implementation that renders into a surface the
application has not released fails conformance.

## 5. Commands on the control envelope (application → sidecar)

Payloads are JSON in `args.request`, answers in `result.data`, like every sidecar command. A
missing or malformed field is refused with its name.

| command | request | answer |
| --- | --- | --- |
| `surface.open` | `identifier, window, pane, pixelW, pixelH, scale, font{family, pt}, theme{fg, bg, cursor, cursorAccent, selectionBg, selectionFg, ansi[256]}, cwd?` | `cols, rows, cellW, cellH` — the ring follows on the channel |
| `surface.resize` | `pane, pixelW, pixelH, scale` | `cols, rows` |
| `surface.setPaused` | `pane, paused` | `{}` — paused produces no frame |
| `surface.preedit` | `pane, text, caret` | `{}` — drawn as overlay, never written to the PTY |
| `surface.selection` | `pane, from{row, col}, to{row, col}` or `pane, clear: true` | `text` of the selection |
| `surface.hover` | `pane, row, col` or `pane, clear: true` | `{}` — link underline |
| `surface.scroll` | `pane, offset` \| `lines` \| `edge: "top"\|"bottom"` | `offset, historySize` |
| `surface.read` | `pane, lines?` | `text` — the viewport at the current offset |
| `surface.theme` | `pane, theme{…}` | `{}` — no ring rebuild |
| `surface.close` | `pane` | `{}` — ends the ring with `ended` |

`cols`/`rows` are the sidecar's answer, never the application's guess: the sidecar measured the
font. The application drives `pty.resize` with the answered numbers.

## 6. Presence

The process that renders is the process that displays. The render sidecar therefore declares
`displays: true` when it registers its PTY observer, and the PTY daemon counts a displaying
observer as renderer presence for abandonment. The field itself lives in the PTY contract; this
section states who sets it and why.

## 7. Judged by

- Conformance (this repository): channel-name derivation and its length refusal; message
  round-trips for every kind; the ring state machine refusing a render into an unreleased
  surface; command payload validation naming the first missing field.
- Numbers (application side): `surface.composition` worst 0, `frameReady`-to-commit under one
  frame, zero channel messages while paused.

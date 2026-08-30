# soksak-contract-surface — sidecar-rendered terminal surfaces

Contract id: **`soksak-spec-sidecar-surface`**, version **0.0.7**.

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
| `surface.measure` | `pixelW, pixelH, scale, font{family, pt}` | `cols, rows, cellW, cellH` — no pane, ring or process is created |
| `surface.open` | `identifier, window, pane, pixelW, pixelH, scale, font{family, pt}, theme{fg, bg, cursor, cursorAccent, selectionBg, selectionFg, ansi[256]}, cwd?` | `cols, rows, cellW, cellH` — the ring follows on the channel |
| `surface.resize` | `pane, pixelW, pixelH, scale` | `cols, rows` |
| `surface.setPaused` | `pane, paused` | `{}` — paused produces no frame |
| `surface.preedit` | `pane, text, caret` | `{}` — drawn as overlay, never written to the PTY |
| `surface.selection` | `window, pane, action: "read"` \| `action: "clear"` \| `action: "gesture", gestureId, phase, kind, point{row,col,side}, modifiers{shift,alt,control,meta}` | complete `SelectionSnapshot` |
| `surface.hover` | `pane, row, col` or `pane, clear: true` | `{}` — link underline |
| `surface.pointer` | `window, pane, point{x,y}, phase: "down"\|"move"\|"up", button: "none"\|"left"\|"middle"\|"right", clickCount, modifiers{shift,alt,control,meta}` | engine result: `route: "mouse-report"\|"ignored", dataB64` |
| `surface.wheel` | `window, pane, point{x,y}, deltaX, deltaY, deltaMode: "pixel"\|"line"\|"page", modifiers{shift,alt,control,meta}` | engine result: `route: "scrollback"\|"mouse-report"\|"alternate-scroll"\|"ignored", offset, historySize, dataB64` |
| `surface.focus` | `window, pane, focused` | `focused, cursorPresentation: "engine"\|"hollow-block"` |
| `surface.scroll` | `pane, offset` \| `lines` \| `edge: "top"\|"bottom"`; positive `lines` moves into history and negative `lines` moves toward the bottom | `offset, historySize` |
| `surface.read` | `pane, lines?` | `text` — the viewport at the current offset |
| `surface.theme` | `pane, theme{…}` | `{}` — no ring rebuild |
| `surface.close` | `pane` | `{}` — ends the ring with `ended` |

`cols`/`rows` are the sidecar's answer, never the application's guess: the sidecar measured the
font. `surface.measure` is side-effect free and is valid before a mirror, pane, ring or PTY exists.
For a fresh pane the application must measure first and pass those exact numbers to observer
preparation, `pty.open`, and engine subscription. `surface.open` then creates the ring with the same
pixel/font facts. A resize after process start is only a later geometry change; it is not the way a
fresh shell receives its initial size.

`surface.selection` is a strict discriminated union. `window` and `pane` are the selection owner
address and are required on every action. `phase` is `begin|update|end`; `kind` is
`simple|block|semantic|line|extend`; `side` is `left|right`. A point is in the currently presented
viewport and the render owner translates its row through the current scroll offset. Every gesture
request carries all four modifiers and one non-empty opaque `gestureId`. A begin claims that owner;
an update or end for another owner is refused as `STALE_GESTURE`. A later begin supersedes the old
owner. Clear is unconditional.

Every action returns `SelectionSnapshot`:

```
active, text, kind, anchor{row,col,side}, focus{row,col,side}, gestureId, sequence
```

`sequence` is monotonic per pane, including clear. A caller adopts only a snapshot whose sequence is
not older than its current observation. When inactive, text is empty and kind, anchor, focus and
gestureId are null. The engine owns gesture expansion, selected text and row ranges used by the
painter. Core, the application service and the Plugin do not reconstruct selection from cells.

`surface.wheel` preserves the device facts rather than converting them at the DOM boundary.
`deltaMode` states whether each delta is in pixels, lines or pages; `point` is in CSS pixels relative
to the surface and all four modifiers are required. A zero delta, a non-finite coordinate or an
unknown field is refused. The render owner chooses exactly one route from its current engine modes.
`scrollback` returns non-null `offset` and `historySize` and null `dataB64`. `mouse-report` and
`alternate-scroll` return non-empty base64 PTY input and null scroll state. `ignored` returns all
three effect fields null. The application validates that answer and is the only process that writes
decoded input to the PTY. Core and the Plugin do not encode mouse escape sequences.

`surface.focus` is presentation state, not terminal parser state. Focused restores the engine-owned
cursor shape and blink policy. Unfocused stops the renderer animation clock and paints one steady
hollow block while preserving the engine shape/blink value that will return on focus. The request
is owner-bound by `window` and `pane`; a missing boolean or a contradictory engine answer is
invalid. No adapter parses DECSCUSR or private mode 12 to implement this policy.

`surface.pointer` also preserves surface-relative CSS coordinates and all modifiers. A down or up
names one physical button and has a positive click count. Move may name the held button, or `none`
for no-button motion. The render owner compares the phase against its current 1000/1002/1003 modes
and returns exactly one route. `mouse-report` has non-empty base64 input; `ignored` has null input.
Shift bypasses terminal mouse capture so the Plugin can keep local selection. The application
validates and writes returned bytes through the same single PTY writer used by wheel input.

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

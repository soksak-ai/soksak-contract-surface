//! The Rust half of soksak-contract-surface. SPEC.md is normative; this crate holds the channel
//! name derivation, the message codec and the ring state machine a render sidecar implements
//! against. The Go half beside it serves the application, and both decode the same committed
//! wire fixtures.

use base64::Engine as _;

/// Appended to the installation identifier to derive the bootstrap service name.
pub const CHANNEL_SUFFIX: &str = ".surface";

/// The bootstrap server's name capacity. A longer name is refused, never truncated: a truncated
/// name is a second address.
pub const BOOTSTRAP_NAME_LIMIT: usize = 128;

/// Derives the one service name of an installation's surface channel.
pub fn channel_name(identifier: &str) -> Result<String, String> {
    if identifier.is_empty() {
        return Err("surface channel identifier is empty".into());
    }
    let name = format!("{identifier}{CHANNEL_SUFFIX}");
    if name.len() > BOOTSTRAP_NAME_LIMIT {
        return Err(format!(
            "surface channel name {name:?} exceeds the bootstrap limit of {BOOTSTRAP_NAME_LIMIT} bytes"
        ));
    }
    Ok(name)
}

const WIRE_MAGIC: u32 = 0x736b_7366; // "sksf"
const WIRE_VERSION: u8 = 1;
const HEADER_BYTES: usize = 8;

/// One damage region in cells: (x, y, w, h).
pub type DamageRect = (u16, u16, u16, u16);

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SelectionPhase { Begin, Update, End }

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SelectionKind { Simple, Block, Semantic, Line, Extend }

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum CellSide { Left, Right }

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SelectionPoint {
    pub row: u16,
    pub col: u16,
    pub side: CellSide,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SelectionModifiers {
    pub shift: bool,
    pub alt: bool,
    pub control: bool,
    pub meta: bool,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SelectionRequest {
    Read { window: String, pane: String },
    Clear { window: String, pane: String },
    Gesture {
        window: String,
        pane: String,
        gesture_id: String,
        phase: SelectionPhase,
        kind: SelectionKind,
        point: SelectionPoint,
        modifiers: SelectionModifiers,
    },
}

#[derive(Debug, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct SelectionSnapshot {
    pub active: bool,
    pub text: String,
    pub kind: Option<SelectionKind>,
    pub anchor: Option<SelectionPoint>,
    pub focus: Option<SelectionPoint>,
    pub gesture_id: Option<String>,
    pub sequence: u64,
}

#[derive(serde::Deserialize)]
#[serde(tag = "action", rename_all = "lowercase", rename_all_fields = "camelCase", deny_unknown_fields)]
enum SelectionRequestWire {
    Read { window: String, pane: String },
    Clear { window: String, pane: String },
    Gesture {
        window: String,
        pane: String,
        gesture_id: String,
        phase: SelectionPhase,
        kind: SelectionKind,
        point: SelectionPoint,
        modifiers: SelectionModifiers,
    },
}

pub fn decode_selection_request(value: &serde_json::Value) -> Result<SelectionRequest, String> {
    let wire: SelectionRequestWire = serde_json::from_value(value.clone())
        .map_err(|error| format!("surface.selection request is invalid: {error}"))?;
    let request = match wire {
        SelectionRequestWire::Read { window, pane } => SelectionRequest::Read { window, pane },
        SelectionRequestWire::Clear { window, pane } => SelectionRequest::Clear { window, pane },
        SelectionRequestWire::Gesture { window, pane, gesture_id, phase, kind, point, modifiers } => {
            if gesture_id.is_empty() {
                return Err("surface.selection gestureId is empty".into());
            }
            SelectionRequest::Gesture { window, pane, gesture_id, phase, kind, point, modifiers }
        }
    };
    let (window, pane) = match &request {
        SelectionRequest::Read { window, pane } | SelectionRequest::Clear { window, pane }
        | SelectionRequest::Gesture { window, pane, .. } => (window, pane),
    };
    if window.is_empty() {
        return Err("surface.selection window is empty".into());
    }
    if pane.is_empty() {
        return Err("surface.selection pane is empty".into());
    }
    Ok(request)
}

pub fn decode_selection_snapshot(value: &serde_json::Value) -> Result<SelectionSnapshot, String> {
    let snapshot: SelectionSnapshot = serde_json::from_value(value.clone())
        .map_err(|error| format!("surface.selection snapshot is invalid: {error}"))?;
    if snapshot.active {
        if snapshot.kind.is_none() || snapshot.anchor.is_none() || snapshot.focus.is_none()
            || snapshot.gesture_id.as_deref().unwrap_or_default().is_empty()
        {
            return Err("surface.selection active snapshot is incomplete".into());
        }
    } else if !snapshot.text.is_empty() || snapshot.kind.is_some() || snapshot.anchor.is_some()
        || snapshot.focus.is_some() || snapshot.gesture_id.is_some()
    {
        return Err("surface.selection inactive snapshot retains selection state".into());
    }
    Ok(snapshot)
}

pub fn encode_selection_snapshot(snapshot: &SelectionSnapshot) -> Result<serde_json::Value, String> {
    decode_selection_snapshot(&serde_json::to_value(snapshot)
        .map_err(|error| format!("surface.selection snapshot could not encode: {error}"))?)
        .and_then(|valid| serde_json::to_value(valid)
            .map_err(|error| format!("surface.selection snapshot could not encode: {error}")))
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum WheelDeltaMode { Pixel, Line, Page }

#[derive(Debug, Clone, Copy, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SurfacePoint {
    pub x: f64,
    pub y: f64,
}

#[derive(Debug, Clone, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct WheelRequest {
    pub window: String,
    pub pane: String,
    pub point: SurfacePoint,
    pub delta_x: f64,
    pub delta_y: f64,
    pub delta_mode: WheelDeltaMode,
    pub modifiers: SelectionModifiers,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum WheelRoute { Scrollback, MouseReport, AlternateScroll, Ignored }

#[derive(Debug, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct WheelEngineResult {
    pub route: WheelRoute,
    pub offset: Option<u64>,
    pub history_size: Option<u64>,
    pub data_b64: Option<String>,
}

pub fn decode_wheel_request(value: &serde_json::Value) -> Result<WheelRequest, String> {
    let request: WheelRequest = serde_json::from_value(value.clone())
        .map_err(|error| format!("surface.wheel request is invalid: {error}"))?;
    if request.window.is_empty() {
        return Err("surface.wheel window is empty".into());
    }
    if request.pane.is_empty() {
        return Err("surface.wheel pane is empty".into());
    }
    if !request.point.x.is_finite() || request.point.x < 0.0
        || !request.point.y.is_finite() || request.point.y < 0.0
    {
        return Err("surface.wheel point is not non-negative and finite".into());
    }
    if !request.delta_x.is_finite() || !request.delta_y.is_finite() {
        return Err("surface.wheel delta is not finite".into());
    }
    if request.delta_x == 0.0 && request.delta_y == 0.0 {
        return Err("surface.wheel delta is empty".into());
    }
    Ok(request)
}

pub fn decode_wheel_engine_result(value: &serde_json::Value) -> Result<WheelEngineResult, String> {
    let result: WheelEngineResult = serde_json::from_value(value.clone())
        .map_err(|error| format!("surface.wheel engine result is invalid: {error}"))?;
    let input = match result.data_b64.as_deref() {
        None => None,
        Some("") => return Err("surface.wheel engine result dataB64 is empty".into()),
        Some(value) => {
            let decoded = base64::engine::general_purpose::STANDARD.decode(value)
                .map_err(|error| format!("surface.wheel engine result dataB64 is invalid: {error}"))?;
            if decoded.is_empty() {
                return Err("surface.wheel engine result dataB64 decodes empty".into());
            }
            Some(decoded)
        }
    };
    match result.route {
        WheelRoute::Scrollback
            if result.offset.is_some() && result.history_size.is_some() && input.is_none() => {}
        WheelRoute::MouseReport | WheelRoute::AlternateScroll
            if result.offset.is_none() && result.history_size.is_none() && input.is_some() => {}
        WheelRoute::Ignored
            if result.offset.is_none() && result.history_size.is_none() && input.is_none() => {}
        WheelRoute::Scrollback => return Err("surface.wheel scrollback result is incomplete".into()),
        WheelRoute::MouseReport | WheelRoute::AlternateScroll => {
            return Err("surface.wheel input result is incomplete".into())
        }
        WheelRoute::Ignored => return Err("surface.wheel ignored result retains an effect".into()),
    }
    Ok(result)
}

pub fn encode_wheel_engine_result(result: &WheelEngineResult) -> Result<serde_json::Value, String> {
    decode_wheel_engine_result(&serde_json::to_value(result)
        .map_err(|error| format!("surface.wheel engine result could not encode: {error}"))?)
        .and_then(|valid| serde_json::to_value(valid)
            .map_err(|error| format!("surface.wheel engine result could not encode: {error}")))
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum PointerPhase { Down, Move, Up }

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum PointerButton { None, Left, Middle, Right }

#[derive(Debug, Clone, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct PointerRequest {
    pub window: String,
    pub pane: String,
    pub point: SurfacePoint,
    pub phase: PointerPhase,
    pub button: PointerButton,
    pub click_count: u8,
    pub modifiers: SelectionModifiers,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum PointerRoute { MouseReport, Ignored }

#[derive(Debug, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct PointerEngineResult {
    pub route: PointerRoute,
    pub data_b64: Option<String>,
}

pub fn decode_pointer_request(value: &serde_json::Value) -> Result<PointerRequest, String> {
    let request: PointerRequest = serde_json::from_value(value.clone())
        .map_err(|error| format!("surface.pointer request is invalid: {error}"))?;
    if request.window.is_empty() {
        return Err("surface.pointer window is empty".into());
    }
    if request.pane.is_empty() {
        return Err("surface.pointer pane is empty".into());
    }
    if !request.point.x.is_finite() || request.point.x < 0.0
        || !request.point.y.is_finite() || request.point.y < 0.0
    {
        return Err("surface.pointer point is not non-negative and finite".into());
    }
    if !matches!(request.phase, PointerPhase::Move) && matches!(request.button, PointerButton::None) {
        return Err("surface.pointer down/up button is none".into());
    }
    if request.click_count > 3
        || (!matches!(request.phase, PointerPhase::Move) && request.click_count == 0)
    {
        return Err("surface.pointer clickCount is invalid".into());
    }
    Ok(request)
}

pub fn decode_pointer_engine_result(value: &serde_json::Value) -> Result<PointerEngineResult, String> {
    let result: PointerEngineResult = serde_json::from_value(value.clone())
        .map_err(|error| format!("surface.pointer engine result is invalid: {error}"))?;
    let input = match result.data_b64.as_deref() {
        None => None,
        Some("") => return Err("surface.pointer engine result dataB64 is empty".into()),
        Some(value) => {
            let decoded = base64::engine::general_purpose::STANDARD.decode(value)
                .map_err(|error| format!("surface.pointer engine result dataB64 is invalid: {error}"))?;
            if decoded.is_empty() {
                return Err("surface.pointer engine result dataB64 decodes empty".into());
            }
            Some(decoded)
        }
    };
    match result.route {
        PointerRoute::MouseReport if input.is_some() => {}
        PointerRoute::Ignored if input.is_none() => {}
        PointerRoute::MouseReport => return Err("surface.pointer mouse-report result has no input".into()),
        PointerRoute::Ignored => return Err("surface.pointer ignored result retains input".into()),
    }
    Ok(result)
}

pub fn encode_pointer_engine_result(result: &PointerEngineResult) -> Result<serde_json::Value, String> {
    decode_pointer_engine_result(&serde_json::to_value(result)
        .map_err(|error| format!("surface.pointer engine result could not encode: {error}"))?)
        .and_then(|valid| serde_json::to_value(valid)
            .map_err(|error| format!("surface.pointer engine result could not encode: {error}")))
}

/// One channel message. Ports travel out of band as mach descriptors; `port_count` states how
/// many rights ride beside the bytes of each kind.
#[derive(Debug, Clone, PartialEq)]
pub enum Message {
    Hello { sidecar_id: String },
    Ring { pane: String, pixel_w: u32, pixel_h: u32, scale: f64, cell_w: f64, cell_h: f64 },
    FrameReady {
        pane: String,
        ring_index: u8,
        seq: u64,
        cursor_row: u16,
        cursor_col: u16,
        cursor_visible: bool,
        damage: Vec<DamageRect>,
    },
    Released { pane: String, ring_index: u8 },
    Gap { pane: String },
    Ended { pane: String, reason: String },
}

impl Message {
    pub fn kind(&self) -> u8 {
        match self {
            Message::Hello { .. } => 1,
            Message::Ring { .. } => 2,
            Message::FrameReady { .. } => 3,
            Message::Released { .. } => 4,
            Message::Gap { .. } => 5,
            Message::Ended { .. } => 6,
        }
    }

    pub fn port_count(&self) -> usize {
        match self {
            Message::Hello { .. } => 1,
            Message::Ring { .. } => 3,
            _ => 0,
        }
    }
}

struct Writer {
    out: Vec<u8>,
}

impl Writer {
    fn short(&mut self, text: &str, field: &str) -> Result<(), String> {
        if text.len() > u8::MAX as usize {
            return Err(format!("surface message {field} of {} bytes exceeds {}", text.len(), u8::MAX));
        }
        self.out.push(text.len() as u8);
        self.out.extend_from_slice(text.as_bytes());
        Ok(())
    }
    fn u8(&mut self, value: u8) {
        self.out.push(value);
    }
    fn u16(&mut self, value: u16) {
        self.out.extend_from_slice(&value.to_be_bytes());
    }
    fn u32(&mut self, value: u32) {
        self.out.extend_from_slice(&value.to_be_bytes());
    }
    fn u64(&mut self, value: u64) {
        self.out.extend_from_slice(&value.to_be_bytes());
    }
    fn f64(&mut self, value: f64) {
        self.out.extend_from_slice(&value.to_bits().to_be_bytes());
    }
    fn boolean(&mut self, value: bool) {
        self.out.push(if value { 1 } else { 0 });
    }
}

struct Reader<'wire> {
    body: &'wire [u8],
    at: usize,
}

impl<'wire> Reader<'wire> {
    fn take(&mut self, n: usize, field: &str) -> Result<&'wire [u8], String> {
        if self.at + n > self.body.len() {
            return Err(format!("surface message payload ends inside {field}"));
        }
        let out = &self.body[self.at..self.at + n];
        self.at += n;
        Ok(out)
    }
    fn short(&mut self, field: &str) -> Result<String, String> {
        let len = self.take(1, field)?[0] as usize;
        let raw = self.take(len, field)?;
        String::from_utf8(raw.to_vec()).map_err(|_| format!("surface message {field} is not UTF-8"))
    }
    fn u8(&mut self, field: &str) -> Result<u8, String> {
        Ok(self.take(1, field)?[0])
    }
    fn u16(&mut self, field: &str) -> Result<u16, String> {
        Ok(u16::from_be_bytes(self.take(2, field)?.try_into().unwrap()))
    }
    fn u32(&mut self, field: &str) -> Result<u32, String> {
        Ok(u32::from_be_bytes(self.take(4, field)?.try_into().unwrap()))
    }
    fn u64(&mut self, field: &str) -> Result<u64, String> {
        Ok(u64::from_be_bytes(self.take(8, field)?.try_into().unwrap()))
    }
    fn f64(&mut self, field: &str) -> Result<f64, String> {
        Ok(f64::from_bits(self.u64(field)?))
    }
    fn boolean(&mut self, field: &str) -> Result<bool, String> {
        Ok(self.u8(field)? != 0)
    }
    fn done(self) -> Result<(), String> {
        if self.at != self.body.len() {
            return Err(format!("surface message payload holds {} trailing bytes", self.body.len() - self.at));
        }
        Ok(())
    }
}

/// Frames one message: magic, version, kind, payload length, payload.
pub fn encode(message: &Message) -> Result<Vec<u8>, String> {
    let mut writer = Writer { out: Vec::new() };
    match message {
        Message::Hello { sidecar_id } => writer.short(sidecar_id, "sidecar id")?,
        Message::Ring { pane, pixel_w, pixel_h, scale, cell_w, cell_h } => {
            writer.short(pane, "pane")?;
            writer.u32(*pixel_w);
            writer.u32(*pixel_h);
            writer.f64(*scale);
            writer.f64(*cell_w);
            writer.f64(*cell_h);
        }
        Message::FrameReady { pane, ring_index, seq, cursor_row, cursor_col, cursor_visible, damage } => {
            if damage.len() > u8::MAX as usize {
                return Err(format!("frameReady damage of {} rects exceeds {}", damage.len(), u8::MAX));
            }
            writer.short(pane, "pane")?;
            writer.u8(*ring_index);
            writer.u64(*seq);
            writer.u16(*cursor_row);
            writer.u16(*cursor_col);
            writer.boolean(*cursor_visible);
            writer.u8(damage.len() as u8);
            for (x, y, w, h) in damage {
                writer.u16(*x);
                writer.u16(*y);
                writer.u16(*w);
                writer.u16(*h);
            }
        }
        Message::Released { pane, ring_index } => {
            writer.short(pane, "pane")?;
            writer.u8(*ring_index);
        }
        Message::Gap { pane } => writer.short(pane, "pane")?,
        Message::Ended { pane, reason } => {
            writer.short(pane, "pane")?;
            writer.short(reason, "reason")?;
        }
    }
    let body = writer.out;
    if body.len() > u16::MAX as usize {
        return Err(format!("surface message payload of {} bytes exceeds the frame limit", body.len()));
    }
    let mut wire = Vec::with_capacity(HEADER_BYTES + body.len());
    wire.extend_from_slice(&WIRE_MAGIC.to_be_bytes());
    wire.push(WIRE_VERSION);
    wire.push(message.kind());
    wire.extend_from_slice(&(body.len() as u16).to_be_bytes());
    wire.extend_from_slice(&body);
    Ok(wire)
}

/// Reads one framed message and refuses a foreign magic, version, kind or length by name.
/// How many bytes one wire message occupies: a receiver trims mach padding
/// with this instead of restating the header grammar.
pub fn wire_length(wire: &[u8]) -> Result<usize, String> {
    if wire.len() < HEADER_BYTES {
        return Err(format!("wire of {} bytes cannot hold a header", wire.len()));
    }
    if wire[0..4] != WIRE_MAGIC.to_be_bytes() {
        return Err("wire magic mismatch".to_string());
    }
    let payload = u16::from_be_bytes([wire[6], wire[7]]) as usize;
    Ok(HEADER_BYTES + payload)
}

pub fn decode(wire: &[u8]) -> Result<Message, String> {
    if wire.len() < HEADER_BYTES {
        return Err(format!("surface message of {} bytes is shorter than the header", wire.len()));
    }
    let magic = u32::from_be_bytes(wire[0..4].try_into().unwrap());
    if magic != WIRE_MAGIC {
        return Err(format!("surface message magic {magic:08x} is not this contract's"));
    }
    if wire[4] != WIRE_VERSION {
        return Err(format!("surface message version {} is not {WIRE_VERSION}", wire[4]));
    }
    let declared = u16::from_be_bytes(wire[6..8].try_into().unwrap()) as usize;
    let body = &wire[HEADER_BYTES..];
    if declared != body.len() {
        return Err(format!("surface message declares {declared} payload bytes and carries {}", body.len()));
    }
    let mut reader = Reader { body, at: 0 };
    let message = match wire[5] {
        1 => Message::Hello { sidecar_id: reader.short("sidecar id")? },
        2 => Message::Ring {
            pane: reader.short("pane")?,
            pixel_w: reader.u32("pixelW")?,
            pixel_h: reader.u32("pixelH")?,
            scale: reader.f64("scale")?,
            cell_w: reader.f64("cellW")?,
            cell_h: reader.f64("cellH")?,
        },
        3 => {
            let pane = reader.short("pane")?;
            let ring_index = reader.u8("ringIndex")?;
            let seq = reader.u64("seq")?;
            let cursor_row = reader.u16("cursorRow")?;
            let cursor_col = reader.u16("cursorCol")?;
            let cursor_visible = reader.boolean("cursorVisible")?;
            let count = reader.u8("damageCount")? as usize;
            let mut damage = Vec::with_capacity(count);
            for _ in 0..count {
                damage.push((
                    reader.u16("damage.x")?,
                    reader.u16("damage.y")?,
                    reader.u16("damage.w")?,
                    reader.u16("damage.h")?,
                ));
            }
            Message::FrameReady { pane, ring_index, seq, cursor_row, cursor_col, cursor_visible, damage }
        }
        4 => Message::Released { pane: reader.short("pane")?, ring_index: reader.u8("ringIndex")? },
        5 => Message::Gap { pane: reader.short("pane")? },
        6 => Message::Ended { pane: reader.short("pane")?, reason: reader.short("reason")? },
        kind => return Err(format!("surface message kind {kind} is not this contract's")),
    };
    reader.done()?;
    Ok(message)
}

/// The number of surfaces per pane.
pub const RING_SIZE: usize = 3;

#[derive(Clone, Copy, PartialEq)]
enum RingPhase {
    Free,
    Rendering,
    Signaled,
    Displayed,
}

/// The conformance state machine of SPEC.md section 4. The sidecar side acquires, signals and
/// receives releases; the application side displays and releases. An implementation that asks for
/// a render target while nothing is free is refused rather than handed a displayed surface.
pub struct RingState {
    phase: [RingPhase; RING_SIZE],
    displayed: Option<usize>,
}

impl Default for RingState {
    fn default() -> Self {
        Self::new()
    }
}

impl RingState {
    pub fn new() -> Self {
        Self { phase: [RingPhase::Free; RING_SIZE], displayed: None }
    }

    /// Hands out one free surface for painting.
    pub fn acquire_for_render(&mut self) -> Result<usize, String> {
        for (index, phase) in self.phase.iter_mut().enumerate() {
            if *phase == RingPhase::Free {
                *phase = RingPhase::Rendering;
                return Ok(index);
            }
        }
        Err("surface ring holds no free surface: nothing was released".into())
    }

    /// Marks one painted surface ready for display.
    pub fn signal(&mut self, index: usize) {
        if index < RING_SIZE && self.phase[index] == RingPhase::Rendering {
            self.phase[index] = RingPhase::Signaled;
        }
    }

    /// Moves the application to one signaled surface.
    pub fn display(&mut self, index: usize) -> Result<(), String> {
        if index >= RING_SIZE || self.phase[index] != RingPhase::Signaled {
            return Err(format!("surface {index} is not signaled"));
        }
        self.phase[index] = RingPhase::Displayed;
        self.displayed = Some(index);
        Ok(())
    }

    /// Returns one surface to the sidecar. The displayed surface is refused: releasing what the
    /// compositor is reading is the tear this state machine exists to prevent.
    pub fn release(&mut self, index: usize) -> Result<(), String> {
        if index >= RING_SIZE {
            return Err(format!("surface {index} is outside the ring"));
        }
        if self.displayed == Some(index) {
            return Err(format!("surface {index} is displayed and cannot be released"));
        }
        self.phase[index] = RingPhase::Free;
        Ok(())
    }
}

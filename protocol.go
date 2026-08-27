// Package surfacecontract is the seam between a render sidecar that owns a terminal grid and the
// application that composites its surfaces. SPEC.md is normative; this package holds the derived
// channel name, the channel message wire, the ring state machine and the command payloads.
package surfacecontract

import (
	"encoding/binary"
	"fmt"
	"math"
)

// ChannelSuffix is appended to the installation identifier to derive the bootstrap service name.
const ChannelSuffix = ".surface"

// BootstrapNameLimit is the bootstrap server's name capacity. A longer name is refused, never
// truncated: a truncated name is a second address.
const BootstrapNameLimit = 128

// ChannelName derives the one service name of an installation's surface channel.
func ChannelName(identifier string) (string, error) {
	if identifier == "" {
		return "", fmt.Errorf("surface channel identifier is empty")
	}
	name := identifier + ChannelSuffix
	if len(name) > BootstrapNameLimit {
		return "", fmt.Errorf("surface channel name %q exceeds the bootstrap limit of %d bytes", name, BootstrapNameLimit)
	}
	return name, nil
}

// Wire framing. Ports travel out of band as mach descriptors; PortCount states how many rights
// ride beside the bytes of each kind.
const (
	wireMagic   uint32 = 0x736b7366 // "sksf"
	wireVersion byte   = 1
	headerBytes        = 8
)

const (
	KindHello      byte = 1
	KindRing       byte = 2
	KindFrameReady byte = 3
	KindReleased   byte = 4
	KindGap        byte = 5
	KindEnded      byte = 6
)

// Message is one channel message. Encode writes the inline payload; the mach rights a kind
// carries are declared by PortCount and attached by the transport.
type Message interface {
	Kind() byte
	PortCount() int
	payload() ([]byte, error)
	readPayload(body []byte) error
}

// Hello is the sidecar's first message. Its one right is the reply port that makes the channel
// bidirectional.
type Hello struct{ SidecarID string }

// Ring hands the application one pane's surfaces. Its three rights are the IOSurfaces in ring
// order 0..2.
type Ring struct {
	Pane           string
	PixelW, PixelH uint32
	Scale          float64
	CellW, CellH   float64
}

// DamageRect is one changed region in cells.
type DamageRect struct{ X, Y, W, H uint16 }

// FrameReady signals one painted surface.
type FrameReady struct {
	Pane                 string
	RingIndex            byte
	Seq                  uint64
	CursorRow, CursorCol uint16
	CursorVisible        bool
	Damage               []DamageRect
}

// Released returns one surface to the sidecar.
type Released struct {
	Pane      string
	RingIndex byte
}

// Gap reports lost source continuity; recovery follows the terminal contract.
type Gap struct{ Pane string }

// Ended closes one pane's ring with its reason.
type Ended struct {
	Pane   string
	Reason string
}

func (m *Hello) Kind() byte      { return KindHello }
func (m *Ring) Kind() byte       { return KindRing }
func (m *FrameReady) Kind() byte { return KindFrameReady }
func (m *Released) Kind() byte   { return KindReleased }
func (m *Gap) Kind() byte        { return KindGap }
func (m *Ended) Kind() byte      { return KindEnded }

func (m *Hello) PortCount() int      { return 1 }
func (m *Ring) PortCount() int       { return 3 }
func (m *FrameReady) PortCount() int { return 0 }
func (m *Released) PortCount() int   { return 0 }
func (m *Gap) PortCount() int        { return 0 }
func (m *Ended) PortCount() int      { return 0 }

// Encode frames one message: magic, version, kind, payload length, payload.
func Encode(message Message) ([]byte, error) {
	body, err := message.payload()
	if err != nil {
		return nil, err
	}
	if len(body) > math.MaxUint16 {
		return nil, fmt.Errorf("surface message payload of %d bytes exceeds the frame limit", len(body))
	}
	wire := make([]byte, headerBytes+len(body))
	binary.BigEndian.PutUint32(wire[0:4], wireMagic)
	wire[4] = wireVersion
	wire[5] = message.Kind()
	binary.BigEndian.PutUint16(wire[6:8], uint16(len(body)))
	copy(wire[headerBytes:], body)
	return wire, nil
}

// Decode reads one framed message and refuses a foreign magic, version, kind or length by name.
func Decode(wire []byte) (Message, error) {
	if len(wire) < headerBytes {
		return nil, fmt.Errorf("surface message of %d bytes is shorter than the header", len(wire))
	}
	if magic := binary.BigEndian.Uint32(wire[0:4]); magic != wireMagic {
		return nil, fmt.Errorf("surface message magic %08x is not this contract's", magic)
	}
	if wire[4] != wireVersion {
		return nil, fmt.Errorf("surface message version %d is not %d", wire[4], wireVersion)
	}
	body := wire[headerBytes:]
	if declared := int(binary.BigEndian.Uint16(wire[6:8])); declared != len(body) {
		return nil, fmt.Errorf("surface message declares %d payload bytes and carries %d", declared, len(body))
	}
	var message Message
	switch wire[5] {
	case KindHello:
		message = &Hello{}
	case KindRing:
		message = &Ring{}
	case KindFrameReady:
		message = &FrameReady{}
	case KindReleased:
		message = &Released{}
	case KindGap:
		message = &Gap{}
	case KindEnded:
		message = &Ended{}
	default:
		return nil, fmt.Errorf("surface message kind %d is not this contract's", wire[5])
	}
	if err := message.readPayload(body); err != nil {
		return nil, err
	}
	return message, nil
}

// Format renders one message for records and test comparisons.
func Format(message Message) string {
	switch m := message.(type) {
	case *Hello:
		return fmt.Sprintf("hello{%s}", m.SidecarID)
	case *Ring:
		return fmt.Sprintf("ring{%s %dx%d @%g cell %gx%g}", m.Pane, m.PixelW, m.PixelH, m.Scale, m.CellW, m.CellH)
	case *FrameReady:
		return fmt.Sprintf("frameReady{%s ring %d seq %d cursor %d,%d visible %t damage %v}",
			m.Pane, m.RingIndex, m.Seq, m.CursorRow, m.CursorCol, m.CursorVisible, m.Damage)
	case *Released:
		return fmt.Sprintf("released{%s ring %d}", m.Pane, m.RingIndex)
	case *Gap:
		return fmt.Sprintf("gap{%s}", m.Pane)
	case *Ended:
		return fmt.Sprintf("ended{%s %s}", m.Pane, m.Reason)
	}
	return fmt.Sprintf("unknown{%T}", message)
}

type wireWriter struct{ out []byte }

func (w *wireWriter) short(text, field string) error {
	if len(text) > math.MaxUint8 {
		return fmt.Errorf("surface message %s of %d bytes exceeds %d", field, len(text), math.MaxUint8)
	}
	w.out = append(w.out, byte(len(text)))
	w.out = append(w.out, text...)
	return nil
}
func (w *wireWriter) u8(v byte)      { w.out = append(w.out, v) }
func (w *wireWriter) u16(v uint16)   { w.out = binary.BigEndian.AppendUint16(w.out, v) }
func (w *wireWriter) u32(v uint32)   { w.out = binary.BigEndian.AppendUint32(w.out, v) }
func (w *wireWriter) u64(v uint64)   { w.out = binary.BigEndian.AppendUint64(w.out, v) }
func (w *wireWriter) f64(v float64)  { w.out = binary.BigEndian.AppendUint64(w.out, math.Float64bits(v)) }
func (w *wireWriter) boolean(v bool) { w.u8(map[bool]byte{true: 1}[v]) }

type wireReader struct {
	body []byte
	at   int
	err  error
}

func (r *wireReader) take(n int, field string) []byte {
	if r.err != nil {
		return nil
	}
	if r.at+n > len(r.body) {
		r.err = fmt.Errorf("surface message payload ends inside %s", field)
		return nil
	}
	out := r.body[r.at : r.at+n]
	r.at += n
	return out
}
func (r *wireReader) short(field string) string {
	lead := r.take(1, field)
	if r.err != nil {
		return ""
	}
	return string(r.take(int(lead[0]), field))
}
func (r *wireReader) u8(field string) byte {
	out := r.take(1, field)
	if r.err != nil {
		return 0
	}
	return out[0]
}
func (r *wireReader) u16(field string) uint16 {
	out := r.take(2, field)
	if r.err != nil {
		return 0
	}
	return binary.BigEndian.Uint16(out)
}
func (r *wireReader) u32(field string) uint32 {
	out := r.take(4, field)
	if r.err != nil {
		return 0
	}
	return binary.BigEndian.Uint32(out)
}
func (r *wireReader) u64(field string) uint64 {
	out := r.take(8, field)
	if r.err != nil {
		return 0
	}
	return binary.BigEndian.Uint64(out)
}
func (r *wireReader) f64(field string) float64 { return math.Float64frombits(r.u64(field)) }
func (r *wireReader) boolean(field string) bool { return r.u8(field) != 0 }
func (r *wireReader) done() error {
	if r.err != nil {
		return r.err
	}
	if r.at != len(r.body) {
		return fmt.Errorf("surface message payload holds %d trailing bytes", len(r.body)-r.at)
	}
	return nil
}

func (m *Hello) payload() ([]byte, error) {
	w := &wireWriter{}
	if err := w.short(m.SidecarID, "sidecar id"); err != nil {
		return nil, err
	}
	return w.out, nil
}
func (m *Hello) readPayload(body []byte) error {
	r := &wireReader{body: body}
	m.SidecarID = r.short("sidecar id")
	return r.done()
}

func (m *Ring) payload() ([]byte, error) {
	w := &wireWriter{}
	if err := w.short(m.Pane, "pane"); err != nil {
		return nil, err
	}
	w.u32(m.PixelW)
	w.u32(m.PixelH)
	w.f64(m.Scale)
	w.f64(m.CellW)
	w.f64(m.CellH)
	return w.out, nil
}
func (m *Ring) readPayload(body []byte) error {
	r := &wireReader{body: body}
	m.Pane = r.short("pane")
	m.PixelW = r.u32("pixelW")
	m.PixelH = r.u32("pixelH")
	m.Scale = r.f64("scale")
	m.CellW = r.f64("cellW")
	m.CellH = r.f64("cellH")
	return r.done()
}

func (m *FrameReady) payload() ([]byte, error) {
	if len(m.Damage) > math.MaxUint8 {
		return nil, fmt.Errorf("frameReady damage of %d rects exceeds %d", len(m.Damage), math.MaxUint8)
	}
	w := &wireWriter{}
	if err := w.short(m.Pane, "pane"); err != nil {
		return nil, err
	}
	w.u8(m.RingIndex)
	w.u64(m.Seq)
	w.u16(m.CursorRow)
	w.u16(m.CursorCol)
	w.boolean(m.CursorVisible)
	w.u8(byte(len(m.Damage)))
	for _, rect := range m.Damage {
		w.u16(rect.X)
		w.u16(rect.Y)
		w.u16(rect.W)
		w.u16(rect.H)
	}
	return w.out, nil
}
func (m *FrameReady) readPayload(body []byte) error {
	r := &wireReader{body: body}
	m.Pane = r.short("pane")
	m.RingIndex = r.u8("ringIndex")
	m.Seq = r.u64("seq")
	m.CursorRow = r.u16("cursorRow")
	m.CursorCol = r.u16("cursorCol")
	m.CursorVisible = r.boolean("cursorVisible")
	count := int(r.u8("damageCount"))
	m.Damage = nil
	for index := 0; index < count; index++ {
		m.Damage = append(m.Damage, DamageRect{
			X: r.u16("damage.x"), Y: r.u16("damage.y"), W: r.u16("damage.w"), H: r.u16("damage.h"),
		})
	}
	return r.done()
}

func (m *Released) payload() ([]byte, error) {
	w := &wireWriter{}
	if err := w.short(m.Pane, "pane"); err != nil {
		return nil, err
	}
	w.u8(m.RingIndex)
	return w.out, nil
}
func (m *Released) readPayload(body []byte) error {
	r := &wireReader{body: body}
	m.Pane = r.short("pane")
	m.RingIndex = r.u8("ringIndex")
	return r.done()
}

func (m *Gap) payload() ([]byte, error) {
	w := &wireWriter{}
	if err := w.short(m.Pane, "pane"); err != nil {
		return nil, err
	}
	return w.out, nil
}
func (m *Gap) readPayload(body []byte) error {
	r := &wireReader{body: body}
	m.Pane = r.short("pane")
	return r.done()
}

func (m *Ended) payload() ([]byte, error) {
	w := &wireWriter{}
	if err := w.short(m.Pane, "pane"); err != nil {
		return nil, err
	}
	if err := w.short(m.Reason, "reason"); err != nil {
		return nil, err
	}
	return w.out, nil
}
func (m *Ended) readPayload(body []byte) error {
	r := &wireReader{body: body}
	m.Pane = r.short("pane")
	m.Reason = r.short("reason")
	return r.done()
}

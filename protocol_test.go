package surfacecontract

import (
	"strings"
	"testing"
)

// The channel name is derived from the installation identifier, once. A second spelling is a
// second address, and the 128-byte bootstrap limit is refused by name rather than truncated.
func TestChannelNameDerivesFromIdentifier(t *testing.T) {
	name, err := ChannelName("com.soksak.wails.perfanalysis")
	if err != nil {
		t.Fatal(err)
	}
	if name != "com.soksak.wails.perfanalysis.surface" {
		t.Fatalf("unexpected channel name %q", name)
	}
}

func TestChannelNameRefusesOverlongIdentifier(t *testing.T) {
	_, err := ChannelName(strings.Repeat("a", 128))
	if err == nil || !strings.Contains(err.Error(), "128") {
		t.Fatalf("overlong identifier was not refused with the limit named: %v", err)
	}
}

func TestChannelNameRefusesEmptyIdentifier(t *testing.T) {
	if _, err := ChannelName(""); err == nil {
		t.Fatal("empty identifier was accepted")
	}
}

// Every message round-trips: what the sidecar encodes is what the application decodes.
// Ports travel out of band; PortCount states how many rights ride beside the bytes.
func TestMessagesRoundTrip(t *testing.T) {
	for _, message := range []Message{
		&Hello{SidecarID: "soksak-sidecar-terminal-alacritty"},
		&Ring{Pane: "tab-abc123.1", PixelW: 1280, PixelH: 720, Scale: 2, CellW: 14, CellH: 28},
		&FrameReady{Pane: "tab-abc123.1", RingIndex: 2, Seq: 42, CursorRow: 3, CursorCol: 7,
			CursorVisible: true, Damage: []DamageRect{{X: 0, Y: 3, W: 80, H: 1}}},
		&Released{Pane: "tab-abc123.1", RingIndex: 2},
		&Gap{Pane: "tab-abc123.1"},
		&Ended{Pane: "tab-abc123.1", Reason: "closed"},
	} {
		wire, err := Encode(message)
		if err != nil {
			t.Fatalf("%T: %v", message, err)
		}
		decoded, err := Decode(wire)
		if err != nil {
			t.Fatalf("%T: %v", message, err)
		}
		if got, want := stringOf(decoded), stringOf(message); got != want {
			t.Fatalf("round trip changed the message: got %s want %s", got, want)
		}
	}
}

func TestDecodeRefusesForeignMagicAndVersion(t *testing.T) {
	wire, err := Encode(&Gap{Pane: "tab-abc123.1"})
	if err != nil {
		t.Fatal(err)
	}
	bad := append([]byte(nil), wire...)
	bad[0] ^= 0xff
	if _, err := Decode(bad); err == nil || !strings.Contains(err.Error(), "magic") {
		t.Fatalf("foreign magic was not refused by name: %v", err)
	}
	bad = append([]byte(nil), wire...)
	bad[4] = 99
	if _, err := Decode(bad); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("foreign version was not refused by name: %v", err)
	}
}

func TestPortCountsAreDeclaredPerKind(t *testing.T) {
	if got := (&Hello{}).PortCount(); got != 1 {
		t.Fatalf("hello carries the reply port: got %d", got)
	}
	if got := (&Ring{}).PortCount(); got != 3 {
		t.Fatalf("ring carries three surfaces: got %d", got)
	}
	for _, message := range []Message{&FrameReady{}, &Released{}, &Gap{}, &Ended{}} {
		if got := message.PortCount(); got != 0 {
			t.Fatalf("%T carries no right: got %d", message, got)
		}
	}
}

// The ring state machine is the contract: rendering into a surface the application has not
// released is a conformance failure, not a glitch.
func TestRingNeverRendersIntoAnUnreleasedSurface(t *testing.T) {
	ring := NewRingState()
	first, err := ring.AcquireForRender()
	if err != nil {
		t.Fatal(err)
	}
	ring.Signal(first)
	if err := ring.Display(first); err != nil {
		t.Fatal(err)
	}
	second, err := ring.AcquireForRender()
	if err != nil {
		t.Fatal(err)
	}
	ring.Signal(second)
	third, err := ring.AcquireForRender()
	if err != nil {
		t.Fatal(err)
	}
	ring.Signal(third)
	if _, err := ring.AcquireForRender(); err == nil {
		t.Fatal("a fourth render target was handed out while nothing was released")
	}
	if err := ring.Display(second); err != nil {
		t.Fatal(err)
	}
	if err := ring.Release(first); err != nil {
		t.Fatal(err)
	}
	if _, err := ring.AcquireForRender(); err != nil {
		t.Fatalf("released surface was not reusable: %v", err)
	}
}

func TestRingRefusesReleasingTheDisplayedSurface(t *testing.T) {
	ring := NewRingState()
	index, err := ring.AcquireForRender()
	if err != nil {
		t.Fatal(err)
	}
	ring.Signal(index)
	if err := ring.Display(index); err != nil {
		t.Fatal(err)
	}
	if err := ring.Release(index); err == nil {
		t.Fatal("the displayed surface was released")
	}
}

// Command payloads are validated once, here, and a refusal names the first missing field.
func TestValidateOpenNamesTheFirstMissingField(t *testing.T) {
	_, err := ValidateOpen(map[string]any{"window": "win-abc123"})
	if err == nil || !strings.Contains(err.Error(), "pane") {
		t.Fatalf("missing pane was not named: %v", err)
	}
	open, err := ValidateOpen(map[string]any{
		"window": "win-abc123", "pane": "tab-abc123.1", "pixelW": 1280.0, "pixelH": 720.0,
		"scale": 2.0, "font": map[string]any{"family": "Menlo", "pt": 13.0},
		"theme": map[string]any{"fg": "#ffffff", "bg": "#000000", "cursor": "#ffffff",
			"cursorAccent": "#000000", "selectionBg": "#334455", "selectionFg": "#ffffff",
			"ansi": ansiFixture()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if open.Pane != "tab-abc123.1" || open.PixelW != 1280 || open.Font.Pt != 13 {
		t.Fatalf("validated open lost fields: %+v", open)
	}
}

func TestCommandNamesAreTheContract(t *testing.T) {
	want := []string{
		"surface.open", "surface.resize", "surface.setPaused", "surface.preedit",
		"surface.selection", "surface.hover", "surface.scroll", "surface.read",
		"surface.theme", "surface.close",
	}
	if got := strings.Join(CommandNames(), ","); got != strings.Join(want, ",") {
		t.Fatalf("command table drifted: %s", got)
	}
}

func ansiFixture() []any {
	out := make([]any, 256)
	for index := range out {
		out[index] = "#101010"
	}
	return out
}

func stringOf(message Message) string { return Format(message) }

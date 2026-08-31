package surfacecontract

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateFixtures = flag.Bool("update", false, "rewrite testdata/messages from the canonical set")

// fixtureSet is the canonical message of every kind. The committed bytes are what both halves —
// this package and the Rust crate — must encode and decode; a layout change that forgets one side
// fails here or in tests/wire.rs rather than on a live channel.
func fixtureSet() map[string]Message {
	return map[string]Message{
		"hello":      &Hello{SidecarID: "fixture-surface-provider"},
		"ring":       &Ring{Pane: "tab-abc123.1", PixelW: 1280, PixelH: 720, Scale: 2, CellW: 14.5, CellH: 29},
		"frameReady": &FrameReady{Pane: "tab-abc123.1", RingIndex: 2, Seq: 42, CursorRow: 3, CursorCol: 7, CursorVisible: true, Damage: []DamageRect{{X: 0, Y: 3, W: 80, H: 1}, {X: 4, Y: 9, W: 2, H: 2}}},
		"released":   &Released{Pane: "tab-abc123.1", RingIndex: 2},
		"gap":        &Gap{Pane: "tab-abc123.1"},
		"ended":      &Ended{Pane: "tab-abc123.1", Reason: "closed"},
	}
}

func TestWireFixturesMatchTheCommittedBytes(t *testing.T) {
	for name, message := range fixtureSet() {
		path := filepath.Join("testdata", "messages", name+".bin")
		wire, err := Encode(message)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if *updateFixtures {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, wire, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		committed, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v (run: go test -run TestWireFixtures -update)", name, err)
		}
		if !bytes.Equal(committed, wire) {
			t.Fatalf("%s: committed %d bytes differ from the encoder's %d", name, len(committed), len(wire))
		}
		decoded, err := Decode(committed)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if Format(decoded) != Format(message) {
			t.Fatalf("%s: decoded %s, canonical %s", name, Format(decoded), Format(message))
		}
	}
}

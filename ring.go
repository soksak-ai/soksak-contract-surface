package surfacecontract

import "fmt"

// RingSize is the number of surfaces per pane.
const RingSize = 3

type ringPhase byte

const (
	ringFree ringPhase = iota
	ringRendering
	ringSignaled
	ringDisplayed
)

// RingState is the conformance state machine of SPEC.md §4. The sidecar side acquires, signals
// and receives releases; the application side displays and releases. An implementation that asks
// for a render target while nothing is free is refused rather than handed a displayed surface.
type RingState struct {
	phase     [RingSize]ringPhase
	displayed int
}

// NewRingState answers a ring with every surface free and nothing displayed.
func NewRingState() *RingState { return &RingState{displayed: -1} }

// AcquireForRender hands out one free surface for painting.
func (ring *RingState) AcquireForRender() (int, error) {
	for index := range ring.phase {
		if ring.phase[index] == ringFree {
			ring.phase[index] = ringRendering
			return index, nil
		}
	}
	return -1, fmt.Errorf("surface ring holds no free surface: nothing was released")
}

// Signal marks one painted surface ready for display.
func (ring *RingState) Signal(index int) {
	if index >= 0 && index < RingSize && ring.phase[index] == ringRendering {
		ring.phase[index] = ringSignaled
	}
}

// Display moves the application to one signaled surface.
func (ring *RingState) Display(index int) error {
	if index < 0 || index >= RingSize || ring.phase[index] != ringSignaled {
		return fmt.Errorf("surface %d is not signaled", index)
	}
	ring.phase[index] = ringDisplayed
	ring.displayed = index
	return nil
}

// Release returns one surface to the sidecar. The displayed surface is refused: releasing what
// the compositor is reading is the tear this state machine exists to prevent.
func (ring *RingState) Release(index int) error {
	if index < 0 || index >= RingSize {
		return fmt.Errorf("surface %d is outside the ring", index)
	}
	if index == ring.displayed {
		return fmt.Errorf("surface %d is displayed and cannot be released", index)
	}
	ring.phase[index] = ringFree
	return nil
}

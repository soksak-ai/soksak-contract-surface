package surfacecontract

import (
	"strings"
	"testing"
)

// The sidecar process holds no installation identifier of its own; the open
// request is the only place the channel name's material can arrive.
func TestOpenNamesTheIdentifierFirst(t *testing.T) {
	_, err := ValidateOpen(map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "identifier") {
		t.Fatalf("an empty open must miss the identifier first, got: %v", err)
	}
}

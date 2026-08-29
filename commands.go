package surfacecontract

import (
	"fmt"
	"math"
	"sort"
)

// CommandNames is the closed surface.* table of SPEC.md §5, in its order.
func CommandNames() []string {
	return []string{
		"surface.open", "surface.resize", "surface.setPaused", "surface.preedit",
		"surface.selection", "surface.hover", "surface.scroll", "surface.read",
		"surface.theme", "surface.close",
	}
}

// FontSpec is the face the sidecar measures cells from.
type FontSpec struct {
	Family string
	Pt     float64
}

// ThemeSpec carries every color a frame resolves against.
type ThemeSpec struct {
	Fg, Bg, Cursor, CursorAccent, SelectionBg, SelectionFg string
	Ansi                                                   [256]string
}

// Open is a validated surface.open request.
type Open struct {
	// Identifier is the installation identifier the channel name derives
	// from; the sidecar process holds none of its own.
	Identifier     string
	Window, Pane   string
	PixelW, PixelH uint32
	Scale          float64
	Font           FontSpec
	Theme          ThemeSpec
	Cwd            string
}

type SelectionAction string

const (
	SelectionRead    SelectionAction = "read"
	SelectionClear   SelectionAction = "clear"
	SelectionGesture SelectionAction = "gesture"
)

type SelectionPhase string

const (
	SelectionBegin  SelectionPhase = "begin"
	SelectionUpdate SelectionPhase = "update"
	SelectionEnd    SelectionPhase = "end"
)

type SelectionKind string

const (
	SelectionSimple   SelectionKind = "simple"
	SelectionBlock    SelectionKind = "block"
	SelectionSemantic SelectionKind = "semantic"
	SelectionLine     SelectionKind = "line"
	SelectionExtend   SelectionKind = "extend"
)

type CellSide string

const (
	CellLeft  CellSide = "left"
	CellRight CellSide = "right"
)

// SelectionPoint is a cell in the currently presented viewport. The side keeps a half-cell drag
// endpoint exact; the render owner translates viewport rows through its current scroll offset.
type SelectionPoint struct {
	Row, Col uint16
	Side     CellSide
}

type SelectionModifiers struct {
	Shift, Alt, Control, Meta bool
}

// SelectionRequest is the closed surface.selection request union. GestureID is an opaque owner
// identity: begin claims it, and update/end for any other identity are refused as STALE_GESTURE.
type SelectionRequest struct {
	Pane      string
	Action    SelectionAction
	GestureID string
	Phase     SelectionPhase
	Kind      SelectionKind
	Point     *SelectionPoint
	Modifiers SelectionModifiers
}

// SelectionSnapshot is returned by every selection action. Sequence is monotonic per pane; a
// caller never replaces a newer observed snapshot with an older async reply.
type SelectionSnapshot struct {
	Active    bool
	Text      string
	Kind      *SelectionKind
	Anchor    *SelectionPoint
	Focus     *SelectionPoint
	GestureID *string
	Sequence  uint64
}

// ValidateSelection checks the exact discriminated request shape.
func ValidateSelection(request map[string]any) (SelectionRequest, error) {
	pane, err := text(request, "pane")
	if err != nil {
		return SelectionRequest{}, err
	}
	actionText, err := text(request, "action")
	if err != nil {
		return SelectionRequest{}, err
	}
	action := SelectionAction(actionText)
	selection := SelectionRequest{Pane: pane, Action: action}
	switch action {
	case SelectionRead, SelectionClear:
		if err := exactKeys("surface.selection", request, "pane", "action"); err != nil {
			return SelectionRequest{}, err
		}
		return selection, nil
	case SelectionGesture:
		if err := exactKeys("surface.selection", request,
			"pane", "action", "gestureId", "phase", "kind", "point", "modifiers"); err != nil {
			return SelectionRequest{}, err
		}
		if selection.GestureID, err = text(request, "gestureId"); err != nil {
			return SelectionRequest{}, err
		}
		phaseText, phaseErr := text(request, "phase")
		if phaseErr != nil {
			return SelectionRequest{}, phaseErr
		}
		selection.Phase = SelectionPhase(phaseText)
		if selection.Phase != SelectionBegin && selection.Phase != SelectionUpdate && selection.Phase != SelectionEnd {
			return SelectionRequest{}, fmt.Errorf("surface.selection phase is not begin, update or end")
		}
		kindText, kindErr := text(request, "kind")
		if kindErr != nil {
			return SelectionRequest{}, kindErr
		}
		selection.Kind = SelectionKind(kindText)
		if selection.Kind != SelectionSimple && selection.Kind != SelectionBlock &&
			selection.Kind != SelectionSemantic && selection.Kind != SelectionLine && selection.Kind != SelectionExtend {
			return SelectionRequest{}, fmt.Errorf("surface.selection kind is not simple, block, semantic, line or extend")
		}
		point, pointErr := selectionPoint(request, "point")
		if pointErr != nil {
			return SelectionRequest{}, pointErr
		}
		selection.Point = &point
		modifiers, modifiersErr := object(request, "modifiers")
		if modifiersErr != nil {
			return SelectionRequest{}, modifiersErr
		}
		if err := exactKeys("surface.selection modifiers", modifiers, "shift", "alt", "control", "meta"); err != nil {
			return SelectionRequest{}, err
		}
		if selection.Modifiers.Shift, err = boolean(modifiers, "shift"); err != nil {
			return SelectionRequest{}, err
		}
		if selection.Modifiers.Alt, err = boolean(modifiers, "alt"); err != nil {
			return SelectionRequest{}, err
		}
		if selection.Modifiers.Control, err = boolean(modifiers, "control"); err != nil {
			return SelectionRequest{}, err
		}
		if selection.Modifiers.Meta, err = boolean(modifiers, "meta"); err != nil {
			return SelectionRequest{}, err
		}
		return selection, nil
	default:
		return SelectionRequest{}, fmt.Errorf("surface.selection action is not read, clear or gesture")
	}
}

func selectionPoint(request map[string]any, field string) (SelectionPoint, error) {
	point, err := object(request, field)
	if err != nil {
		return SelectionPoint{}, err
	}
	if err := exactKeys("surface.selection "+field, point, "row", "col", "side"); err != nil {
		return SelectionPoint{}, err
	}
	row, err := nonNegativeInteger(point, "row")
	if err != nil || row > math.MaxUint16 {
		return SelectionPoint{}, fmt.Errorf("surface.selection %s.row is not a uint16", field)
	}
	col, err := nonNegativeInteger(point, "col")
	if err != nil || col > math.MaxUint16 {
		return SelectionPoint{}, fmt.Errorf("surface.selection %s.col is not a uint16", field)
	}
	sideText, err := text(point, "side")
	if err != nil {
		return SelectionPoint{}, err
	}
	side := CellSide(sideText)
	if side != CellLeft && side != CellRight {
		return SelectionPoint{}, fmt.Errorf("surface.selection %s.side is not left or right", field)
	}
	return SelectionPoint{Row: uint16(row), Col: uint16(col), Side: side}, nil
}

// ValidateSelectionSnapshot checks the full answer and its inactive-state null rule.
func ValidateSelectionSnapshot(answer map[string]any) (SelectionSnapshot, error) {
	if err := exactKeys("surface.selection answer", answer,
		"active", "text", "kind", "anchor", "focus", "gestureId", "sequence"); err != nil {
		return SelectionSnapshot{}, err
	}
	active, err := boolean(answer, "active")
	if err != nil {
		return SelectionSnapshot{}, err
	}
	textValue, textual := answer["text"].(string)
	if !textual {
		return SelectionSnapshot{}, fmt.Errorf("surface.selection answer text is not a string")
	}
	sequence, err := nonNegativeInteger(answer, "sequence")
	if err != nil {
		return SelectionSnapshot{}, err
	}
	snapshot := SelectionSnapshot{Active: active, Text: textValue, Sequence: sequence}
	if !active {
		for _, field := range []string{"kind", "anchor", "focus", "gestureId"} {
			if answer[field] != nil {
				return SelectionSnapshot{}, fmt.Errorf("surface.selection inactive answer %s is not null", field)
			}
		}
		return snapshot, nil
	}
	kindText, ok := answer["kind"].(string)
	if !ok {
		return SelectionSnapshot{}, fmt.Errorf("surface.selection active answer kind is not a string")
	}
	kind := SelectionKind(kindText)
	if kind != SelectionSimple && kind != SelectionBlock && kind != SelectionSemantic && kind != SelectionLine && kind != SelectionExtend {
		return SelectionSnapshot{}, fmt.Errorf("surface.selection active answer kind is invalid")
	}
	anchor, err := selectionPoint(answer, "anchor")
	if err != nil {
		return SelectionSnapshot{}, err
	}
	focus, err := selectionPoint(answer, "focus")
	if err != nil {
		return SelectionSnapshot{}, err
	}
	gestureID, ok := answer["gestureId"].(string)
	if !ok || gestureID == "" {
		return SelectionSnapshot{}, fmt.Errorf("surface.selection active answer gestureId is not a non-empty string")
	}
	snapshot.Kind, snapshot.Anchor, snapshot.Focus, snapshot.GestureID = &kind, &anchor, &focus, &gestureID
	return snapshot, nil
}

// ValidateOpen checks one surface.open payload and names the first missing or malformed field.
func ValidateOpen(request map[string]any) (Open, error) {
	var open Open
	var err error
	if open.Identifier, err = text(request, "identifier"); err != nil {
		return Open{}, err
	}
	if open.Window, err = text(request, "window"); err != nil {
		return Open{}, err
	}
	if open.Pane, err = text(request, "pane"); err != nil {
		return Open{}, err
	}
	pixelW, err := number(request, "pixelW")
	if err != nil {
		return Open{}, err
	}
	pixelH, err := number(request, "pixelH")
	if err != nil {
		return Open{}, err
	}
	if pixelW < 1 || pixelH < 1 {
		return Open{}, fmt.Errorf("surface.open pixel box %gx%g is empty", pixelW, pixelH)
	}
	open.PixelW, open.PixelH = uint32(pixelW), uint32(pixelH)
	if open.Scale, err = number(request, "scale"); err != nil {
		return Open{}, err
	}
	font, err := object(request, "font")
	if err != nil {
		return Open{}, err
	}
	if open.Font.Family, err = labeled(font, "family", "font.family"); err != nil {
		return Open{}, err
	}
	if open.Font.Pt, err = labeledNumber(font, "pt", "font.pt"); err != nil {
		return Open{}, err
	}
	theme, err := object(request, "theme")
	if err != nil {
		return Open{}, err
	}
	for field, into := range map[string]*string{
		"fg": &open.Theme.Fg, "bg": &open.Theme.Bg, "cursor": &open.Theme.Cursor,
		"cursorAccent": &open.Theme.CursorAccent, "selectionBg": &open.Theme.SelectionBg,
		"selectionFg": &open.Theme.SelectionFg,
	} {
		if *into, err = labeled(theme, field, "theme."+field); err != nil {
			return Open{}, err
		}
	}
	ansi, held := theme["ansi"].([]any)
	if !held || len(ansi) != len(open.Theme.Ansi) {
		return Open{}, fmt.Errorf("surface.open theme.ansi must hold %d colors", len(open.Theme.Ansi))
	}
	for index, value := range ansi {
		color, textual := value.(string)
		if !textual {
			return Open{}, fmt.Errorf("surface.open theme.ansi[%d] is not a color", index)
		}
		open.Theme.Ansi[index] = color
	}
	if raw, present := request["cwd"]; present {
		if open.Cwd, err = textOf(raw, "cwd"); err != nil {
			return Open{}, err
		}
	}
	return open, nil
}

func text(request map[string]any, field string) (string, error) {
	return labeled(request, field, field)
}

// labeled reads by key and refuses by label, so a nested field is named by its path.
func labeled(request map[string]any, key, label string) (string, error) {
	raw, present := request[key]
	if !present {
		return "", fmt.Errorf("surface request is missing %s", label)
	}
	return textOf(raw, label)
}

func textOf(raw any, field string) (string, error) {
	value, textual := raw.(string)
	if !textual || value == "" {
		return "", fmt.Errorf("surface request field %s is not a non-empty string", field)
	}
	return value, nil
}

func number(request map[string]any, field string) (float64, error) {
	return labeledNumber(request, field, field)
}

func labeledNumber(request map[string]any, key, label string) (float64, error) {
	raw, present := request[key]
	if !present {
		return 0, fmt.Errorf("surface request is missing %s", label)
	}
	value, numeric := raw.(float64)
	if !numeric {
		return 0, fmt.Errorf("surface request field %s is not a number", label)
	}
	return value, nil
}

func object(request map[string]any, field string) (map[string]any, error) {
	raw, present := request[field]
	if !present {
		return nil, fmt.Errorf("surface request is missing %s", field)
	}
	value, mapped := raw.(map[string]any)
	if !mapped {
		return nil, fmt.Errorf("surface request field %s is not an object", field)
	}
	return value, nil
}

func boolean(request map[string]any, field string) (bool, error) {
	raw, present := request[field]
	if !present {
		return false, fmt.Errorf("surface request is missing %s", field)
	}
	value, valid := raw.(bool)
	if !valid {
		return false, fmt.Errorf("surface request field %s is not a boolean", field)
	}
	return value, nil
}

func nonNegativeInteger(request map[string]any, field string) (uint64, error) {
	raw, present := request[field]
	if !present {
		return 0, fmt.Errorf("surface request is missing %s", field)
	}
	value, valid := raw.(float64)
	if !valid || value < 0 || value >= 1<<53 || math.Trunc(value) != value {
		return 0, fmt.Errorf("surface request field %s is not a non-negative safe integer", field)
	}
	return uint64(value), nil
}

func exactKeys(label string, request map[string]any, fields ...string) error {
	allowed := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		allowed[field] = struct{}{}
	}
	unknown := make([]string, 0)
	for field := range request {
		if _, held := allowed[field]; !held {
			unknown = append(unknown, field)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return fmt.Errorf("%s has unknown field %s", label, unknown[0])
}

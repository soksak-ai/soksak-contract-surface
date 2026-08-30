package surfacecontract

import (
	"encoding/base64"
	"fmt"
	"math"
	"sort"
)

// CommandNames is the closed surface.* table of SPEC.md §5, in its order.
func CommandNames() []string {
	return []string{
		"surface.measure", "surface.open", "surface.resize", "surface.setPaused", "surface.preedit",
		"surface.selection", "surface.hover", "surface.pointer", "surface.wheel", "surface.focus", "surface.scroll", "surface.read",
		"surface.theme", "surface.close",
	}
}

// Measure is the pixel and font input used to determine a terminal grid before a process starts.
type Measure struct {
	PixelW float64
	PixelH float64
	Scale  float64
	Font   FontSpec
}

// MeasureResult is the exact grid and physical cell size selected by the renderer.
type MeasureResult struct {
	Cols  uint64
	Rows  uint64
	CellW float64
	CellH float64
}

// ValidateMeasure validates one surface.measure request.
func ValidateMeasure(request map[string]any) (Measure, error) {
	if err := exactKeys("surface.measure", request, "pixelW", "pixelH", "scale", "font"); err != nil {
		return Measure{}, err
	}
	var measured Measure
	var err error
	if measured.PixelW, err = finiteNumber(request, "pixelW"); err != nil || measured.PixelW <= 0 {
		if err != nil {
			return Measure{}, err
		}
		return Measure{}, fmt.Errorf("surface.measure pixelW is not positive")
	}
	if measured.PixelH, err = finiteNumber(request, "pixelH"); err != nil || measured.PixelH <= 0 {
		if err != nil {
			return Measure{}, err
		}
		return Measure{}, fmt.Errorf("surface.measure pixelH is not positive")
	}
	if measured.Scale, err = finiteNumber(request, "scale"); err != nil || measured.Scale <= 0 {
		if err != nil {
			return Measure{}, err
		}
		return Measure{}, fmt.Errorf("surface.measure scale is not positive")
	}
	font, err := object(request, "font")
	if err != nil {
		return Measure{}, err
	}
	if err := exactKeys("surface.measure font", font, "family", "pt"); err != nil {
		return Measure{}, err
	}
	if measured.Font.Family, err = labeled(font, "family", "font.family"); err != nil {
		return Measure{}, err
	}
	if measured.Font.Pt, err = labeledNumber(font, "pt", "font.pt"); err != nil ||
		!isPositiveFinite(measured.Font.Pt) {
		if err != nil {
			return Measure{}, err
		}
		return Measure{}, fmt.Errorf("surface.measure font.pt is not positive and finite")
	}
	return measured, nil
}

// ValidateMeasureResult validates one surface.measure result.
func ValidateMeasureResult(answer map[string]any) (MeasureResult, error) {
	if err := exactKeys("surface.measure engine answer", answer, "cols", "rows", "cellW", "cellH"); err != nil {
		return MeasureResult{}, err
	}
	cols, err := positiveCount(answer, "cols")
	if err != nil {
		return MeasureResult{}, err
	}
	rows, err := positiveCount(answer, "rows")
	if err != nil {
		return MeasureResult{}, err
	}
	cellW, err := finiteNumber(answer, "cellW")
	if err != nil || cellW <= 0 {
		if err != nil {
			return MeasureResult{}, err
		}
		return MeasureResult{}, fmt.Errorf("surface.measure engine answer cellW is not positive")
	}
	cellH, err := finiteNumber(answer, "cellH")
	if err != nil || cellH <= 0 {
		if err != nil {
			return MeasureResult{}, err
		}
		return MeasureResult{}, fmt.Errorf("surface.measure engine answer cellH is not positive")
	}
	return MeasureResult{Cols: cols, Rows: rows, CellW: cellW, CellH: cellH}, nil
}

func isPositiveFinite(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func positiveCount(request map[string]any, field string) (uint64, error) {
	raw, present := request[field]
	if !present {
		return 0, fmt.Errorf("surface request is missing %s", field)
	}
	var value uint64
	switch number := raw.(type) {
	case uint64:
		value = number
	case float64:
		if number <= 0 || number >= 1<<53 || math.Trunc(number) != number {
			return 0, fmt.Errorf("surface request field %s is not a positive integer", field)
		}
		value = uint64(number)
	default:
		return 0, fmt.Errorf("surface request field %s is not a positive integer", field)
	}
	if value == 0 {
		return 0, fmt.Errorf("surface request field %s is not a positive integer", field)
	}
	return value, nil
}

type CursorPresentation string

const (
	CursorEngine      CursorPresentation = "engine"
	CursorHollowBlock CursorPresentation = "hollow-block"
)

type FocusRequest struct {
	Window, Pane string
	Focused      bool
}

type FocusEngineResult struct {
	Focused            bool
	CursorPresentation CursorPresentation
}

func ValidateFocus(request map[string]any) (FocusRequest, error) {
	if err := exactKeys("surface.focus", request, "window", "pane", "focused"); err != nil {
		return FocusRequest{}, err
	}
	window, err := text(request, "window")
	if err != nil {
		return FocusRequest{}, err
	}
	pane, err := text(request, "pane")
	if err != nil {
		return FocusRequest{}, err
	}
	focused, err := boolean(request, "focused")
	if err != nil {
		return FocusRequest{}, err
	}
	return FocusRequest{Window: window, Pane: pane, Focused: focused}, nil
}

func ValidateFocusEngineResult(answer map[string]any) (FocusEngineResult, error) {
	if err := exactKeys("surface.focus engine answer", answer, "focused", "cursorPresentation"); err != nil {
		return FocusEngineResult{}, err
	}
	focused, err := boolean(answer, "focused")
	if err != nil {
		return FocusEngineResult{}, err
	}
	presentationText, err := text(answer, "cursorPresentation")
	if err != nil {
		return FocusEngineResult{}, err
	}
	presentation := CursorPresentation(presentationText)
	want := CursorHollowBlock
	if focused {
		want = CursorEngine
	}
	if presentation != want {
		return FocusEngineResult{}, fmt.Errorf("surface.focus cursor presentation contradicts focused state")
	}
	return FocusEngineResult{Focused: focused, CursorPresentation: presentation}, nil
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

type WheelDeltaMode string

const (
	WheelDeltaPixel WheelDeltaMode = "pixel"
	WheelDeltaLine  WheelDeltaMode = "line"
	WheelDeltaPage  WheelDeltaMode = "page"
)

type SurfacePoint struct {
	X, Y float64
}

type PointerPhase string

const (
	PointerDown PointerPhase = "down"
	PointerMove PointerPhase = "move"
	PointerUp   PointerPhase = "up"
)

type PointerButton string

const (
	PointerNone   PointerButton = "none"
	PointerLeft   PointerButton = "left"
	PointerMiddle PointerButton = "middle"
	PointerRight  PointerButton = "right"
)

type PointerRequest struct {
	Window     string
	Pane       string
	Point      SurfacePoint
	Phase      PointerPhase
	Button     PointerButton
	ClickCount uint8
	Modifiers  SelectionModifiers
}

type PointerRoute string

const (
	PointerMouseReport PointerRoute = "mouse-report"
	PointerIgnored     PointerRoute = "ignored"
)

type PointerEngineResult struct {
	Route   PointerRoute
	DataB64 *string
}

func ValidatePointer(request map[string]any) (PointerRequest, error) {
	if err := exactKeys("surface.pointer", request,
		"window", "pane", "point", "phase", "button", "clickCount", "modifiers"); err != nil {
		return PointerRequest{}, err
	}
	window, err := text(request, "window")
	if err != nil {
		return PointerRequest{}, err
	}
	pane, err := text(request, "pane")
	if err != nil {
		return PointerRequest{}, err
	}
	pointValue, err := object(request, "point")
	if err != nil {
		return PointerRequest{}, err
	}
	if err := exactKeys("surface.pointer point", pointValue, "x", "y"); err != nil {
		return PointerRequest{}, err
	}
	x, err := finiteNumber(pointValue, "x")
	if err != nil || x < 0 {
		return PointerRequest{}, fmt.Errorf("surface.pointer point.x is not a non-negative finite number")
	}
	y, err := finiteNumber(pointValue, "y")
	if err != nil || y < 0 {
		return PointerRequest{}, fmt.Errorf("surface.pointer point.y is not a non-negative finite number")
	}
	phaseText, err := text(request, "phase")
	if err != nil {
		return PointerRequest{}, err
	}
	phase := PointerPhase(phaseText)
	if phase != PointerDown && phase != PointerMove && phase != PointerUp {
		return PointerRequest{}, fmt.Errorf("surface.pointer phase is not down, move or up")
	}
	buttonText, err := text(request, "button")
	if err != nil {
		return PointerRequest{}, err
	}
	button := PointerButton(buttonText)
	if button != PointerNone && button != PointerLeft && button != PointerMiddle && button != PointerRight {
		return PointerRequest{}, fmt.Errorf("surface.pointer button is invalid")
	}
	if phase != PointerMove && button == PointerNone {
		return PointerRequest{}, fmt.Errorf("surface.pointer down/up button is none")
	}
	clickCount, err := nonNegativeInteger(request, "clickCount")
	if err != nil || clickCount > 3 || (phase != PointerMove && clickCount == 0) {
		return PointerRequest{}, fmt.Errorf("surface.pointer clickCount is invalid")
	}
	modifierValue, err := object(request, "modifiers")
	if err != nil {
		return PointerRequest{}, err
	}
	if err := exactKeys("surface.pointer modifiers", modifierValue, "shift", "alt", "control", "meta"); err != nil {
		return PointerRequest{}, err
	}
	modifiers := SelectionModifiers{}
	if modifiers.Shift, err = boolean(modifierValue, "shift"); err != nil {
		return PointerRequest{}, err
	}
	if modifiers.Alt, err = boolean(modifierValue, "alt"); err != nil {
		return PointerRequest{}, err
	}
	if modifiers.Control, err = boolean(modifierValue, "control"); err != nil {
		return PointerRequest{}, err
	}
	if modifiers.Meta, err = boolean(modifierValue, "meta"); err != nil {
		return PointerRequest{}, err
	}
	return PointerRequest{
		Window: window, Pane: pane, Point: SurfacePoint{X: x, Y: y}, Phase: phase,
		Button: button, ClickCount: uint8(clickCount), Modifiers: modifiers,
	}, nil
}

func ValidatePointerEngineResult(answer map[string]any) (PointerEngineResult, error) {
	if err := exactKeys("surface.pointer engine answer", answer, "route", "dataB64"); err != nil {
		return PointerEngineResult{}, err
	}
	routeText, err := text(answer, "route")
	if err != nil {
		return PointerEngineResult{}, err
	}
	result := PointerEngineResult{Route: PointerRoute(routeText)}
	if answer["dataB64"] != nil {
		value, valid := answer["dataB64"].(string)
		if !valid || value == "" {
			return PointerEngineResult{}, fmt.Errorf("surface.pointer engine answer dataB64 is not a non-empty string")
		}
		decoded, decodeErr := base64.StdEncoding.DecodeString(value)
		if decodeErr != nil || len(decoded) == 0 {
			return PointerEngineResult{}, fmt.Errorf("surface.pointer engine answer dataB64 is not non-empty base64")
		}
		result.DataB64 = &value
	}
	switch result.Route {
	case PointerMouseReport:
		if result.DataB64 == nil {
			return PointerEngineResult{}, fmt.Errorf("surface.pointer mouse-report answer has no input")
		}
	case PointerIgnored:
		if result.DataB64 != nil {
			return PointerEngineResult{}, fmt.Errorf("surface.pointer ignored answer retains input")
		}
	default:
		return PointerEngineResult{}, fmt.Errorf("surface.pointer engine answer route is invalid")
	}
	return result, nil
}

// WheelRequest preserves the input device's coordinate, unit and modifier facts. The render owner
// decides whether those facts scroll history, become a mouse report, or become alternate-screen
// cursor input; callers do not inspect terminal modes and choose a route themselves.
type WheelRequest struct {
	Window         string
	Pane           string
	Point          SurfacePoint
	DeltaX, DeltaY float64
	DeltaMode      WheelDeltaMode
	Modifiers      SelectionModifiers
}

type WheelRoute string

const (
	WheelScrollback      WheelRoute = "scrollback"
	WheelMouseReport     WheelRoute = "mouse-report"
	WheelAlternateScroll WheelRoute = "alternate-scroll"
	WheelIgnored         WheelRoute = "ignored"
)

// WheelEngineResult is the render owner's answer before the application writes any returned input
// through the one PTY writer. All four fields are present on the wire; inactive values are null.
type WheelEngineResult struct {
	Route       WheelRoute
	Offset      *uint64
	HistorySize *uint64
	DataB64     *string
}

func ValidateWheel(request map[string]any) (WheelRequest, error) {
	if err := exactKeys("surface.wheel", request,
		"window", "pane", "point", "deltaX", "deltaY", "deltaMode", "modifiers"); err != nil {
		return WheelRequest{}, err
	}
	window, err := text(request, "window")
	if err != nil {
		return WheelRequest{}, err
	}
	pane, err := text(request, "pane")
	if err != nil {
		return WheelRequest{}, err
	}
	pointValue, err := object(request, "point")
	if err != nil {
		return WheelRequest{}, err
	}
	if err := exactKeys("surface.wheel point", pointValue, "x", "y"); err != nil {
		return WheelRequest{}, err
	}
	x, err := finiteNumber(pointValue, "x")
	if err != nil || x < 0 {
		return WheelRequest{}, fmt.Errorf("surface.wheel point.x is not a non-negative finite number")
	}
	y, err := finiteNumber(pointValue, "y")
	if err != nil || y < 0 {
		return WheelRequest{}, fmt.Errorf("surface.wheel point.y is not a non-negative finite number")
	}
	deltaX, err := finiteNumber(request, "deltaX")
	if err != nil {
		return WheelRequest{}, err
	}
	deltaY, err := finiteNumber(request, "deltaY")
	if err != nil {
		return WheelRequest{}, err
	}
	if deltaX == 0 && deltaY == 0 {
		return WheelRequest{}, fmt.Errorf("surface.wheel delta is empty")
	}
	deltaModeText, err := text(request, "deltaMode")
	if err != nil {
		return WheelRequest{}, err
	}
	deltaMode := WheelDeltaMode(deltaModeText)
	if deltaMode != WheelDeltaPixel && deltaMode != WheelDeltaLine && deltaMode != WheelDeltaPage {
		return WheelRequest{}, fmt.Errorf("surface.wheel deltaMode is not pixel, line or page")
	}
	modifierValue, err := object(request, "modifiers")
	if err != nil {
		return WheelRequest{}, err
	}
	if err := exactKeys("surface.wheel modifiers", modifierValue, "shift", "alt", "control", "meta"); err != nil {
		return WheelRequest{}, err
	}
	modifiers := SelectionModifiers{}
	if modifiers.Shift, err = boolean(modifierValue, "shift"); err != nil {
		return WheelRequest{}, err
	}
	if modifiers.Alt, err = boolean(modifierValue, "alt"); err != nil {
		return WheelRequest{}, err
	}
	if modifiers.Control, err = boolean(modifierValue, "control"); err != nil {
		return WheelRequest{}, err
	}
	if modifiers.Meta, err = boolean(modifierValue, "meta"); err != nil {
		return WheelRequest{}, err
	}
	return WheelRequest{
		Window: window, Pane: pane, Point: SurfacePoint{X: x, Y: y}, DeltaX: deltaX,
		DeltaY: deltaY, DeltaMode: deltaMode, Modifiers: modifiers,
	}, nil
}

func ValidateWheelEngineResult(answer map[string]any) (WheelEngineResult, error) {
	if err := exactKeys("surface.wheel engine answer", answer,
		"route", "offset", "historySize", "dataB64"); err != nil {
		return WheelEngineResult{}, err
	}
	routeText, err := text(answer, "route")
	if err != nil {
		return WheelEngineResult{}, err
	}
	result := WheelEngineResult{Route: WheelRoute(routeText)}
	nullInteger := func(field string) (*uint64, error) {
		if answer[field] == nil {
			return nil, nil
		}
		value, valueErr := nonNegativeInteger(answer, field)
		return &value, valueErr
	}
	if result.Offset, err = nullInteger("offset"); err != nil {
		return WheelEngineResult{}, err
	}
	if result.HistorySize, err = nullInteger("historySize"); err != nil {
		return WheelEngineResult{}, err
	}
	if answer["dataB64"] != nil {
		value, valid := answer["dataB64"].(string)
		if !valid || value == "" {
			return WheelEngineResult{}, fmt.Errorf("surface.wheel engine answer dataB64 is not a non-empty string")
		}
		decoded, decodeErr := base64.StdEncoding.DecodeString(value)
		if decodeErr != nil || len(decoded) == 0 {
			return WheelEngineResult{}, fmt.Errorf("surface.wheel engine answer dataB64 is not non-empty base64")
		}
		result.DataB64 = &value
	}
	switch result.Route {
	case WheelScrollback:
		if result.Offset == nil || result.HistorySize == nil || result.DataB64 != nil {
			return WheelEngineResult{}, fmt.Errorf("surface.wheel scrollback answer is incomplete")
		}
	case WheelMouseReport, WheelAlternateScroll:
		if result.Offset != nil || result.HistorySize != nil || result.DataB64 == nil {
			return WheelEngineResult{}, fmt.Errorf("surface.wheel input answer is incomplete")
		}
	case WheelIgnored:
		if result.Offset != nil || result.HistorySize != nil || result.DataB64 != nil {
			return WheelEngineResult{}, fmt.Errorf("surface.wheel ignored answer retains an effect")
		}
	default:
		return WheelEngineResult{}, fmt.Errorf("surface.wheel engine answer route is invalid")
	}
	return result, nil
}

// SelectionRequest is the closed surface.selection request union. GestureID is an opaque owner
// identity: begin claims it, and update/end for any other identity are refused as STALE_GESTURE.
type SelectionRequest struct {
	Window    string
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
	window, err := text(request, "window")
	if err != nil {
		return SelectionRequest{}, err
	}
	pane, err := text(request, "pane")
	if err != nil {
		return SelectionRequest{}, err
	}
	actionText, err := text(request, "action")
	if err != nil {
		return SelectionRequest{}, err
	}
	action := SelectionAction(actionText)
	selection := SelectionRequest{Window: window, Pane: pane, Action: action}
	switch action {
	case SelectionRead, SelectionClear:
		if err := exactKeys("surface.selection", request, "window", "pane", "action"); err != nil {
			return SelectionRequest{}, err
		}
		return selection, nil
	case SelectionGesture:
		if err := exactKeys("surface.selection", request,
			"window", "pane", "action", "gestureId", "phase", "kind", "point", "modifiers"); err != nil {
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

func finiteNumber(request map[string]any, field string) (float64, error) {
	value, err := number(request, field)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("surface request field %s is not finite", field)
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

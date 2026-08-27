package surfacecontract

import "fmt"

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
	Window, Pane   string
	PixelW, PixelH uint32
	Scale          float64
	Font           FontSpec
	Theme          ThemeSpec
	Cwd            string
}

// ValidateOpen checks one surface.open payload and names the first missing or malformed field.
func ValidateOpen(request map[string]any) (Open, error) {
	var open Open
	var err error
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

package surfacecontract

import "testing"

func TestMeasureDefinesTheGridBeforeAProcessStarts(t *testing.T) {
	request := map[string]any{
		"pixelW": 989.0, "pixelH": 468.0, "scale": 2.0,
		"font": map[string]any{"family": "Menlo", "pt": 13.0},
	}
	measured, err := ValidateMeasure(request)
	if err != nil {
		t.Fatalf("measure request: %v", err)
	}
	if measured.PixelW != 989 || measured.PixelH != 468 || measured.Font.Family != "Menlo" {
		t.Fatalf("measure request changed facts: %+v", measured)
	}
	answer, err := ValidateMeasureResult(map[string]any{
		"cols": uint64(126), "rows": uint64(30), "cellW": 15.6875, "cellH": 31.2,
	})
	if err != nil {
		t.Fatalf("measure result: %v", err)
	}
	if answer.Cols != 126 || answer.Rows != 30 {
		t.Fatalf("measure result changed grid: %+v", answer)
	}
}

func TestMeasureRefusesAnIncompleteOrExpandedTransaction(t *testing.T) {
	valid := map[string]any{
		"pixelW": 989.0, "pixelH": 468.0, "scale": 2.0,
		"font": map[string]any{"family": "Menlo", "pt": 13.0},
	}
	for name, mutate := range map[string]func(map[string]any){
		"missing pixel width": func(value map[string]any) { delete(value, "pixelW") },
		"unknown field":       func(value map[string]any) { value["pane"] = "tab-a.1" },
		"empty font": func(value map[string]any) {
			value["font"] = map[string]any{"family": "", "pt": 13.0}
		},
	} {
		t.Run(name, func(t *testing.T) {
			request := map[string]any{}
			for key, value := range valid {
				request[key] = value
			}
			mutate(request)
			if _, err := ValidateMeasure(request); err == nil {
				t.Fatal("invalid measure request was accepted")
			}
		})
	}
	if _, err := ValidateMeasureResult(map[string]any{
		"cols": uint64(0), "rows": uint64(30), "cellW": 15.0, "cellH": 31.0,
	}); err == nil {
		t.Fatal("zero-column measure result was accepted")
	}
}

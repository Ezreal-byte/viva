package components

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

// targets returns the hits that are interaction targets, leaving out opaque
// blocks.
func targets(canvas *ui.Canvas) []ui.Hit {
	var result []ui.Hit
	for _, hit := range canvas.Hits() {
		if !hit.Block {
			result = append(result, hit)
		}
	}
	return result
}

// blocks returns the opaque regions.
func blocks(canvas *ui.Canvas) []ui.Hit {
	var result []ui.Hit
	for _, hit := range canvas.Hits() {
		if hit.Block {
			result = append(result, hit)
		}
	}
	return result
}

// A modal's panel must shield the dismiss target behind it. The block sits
// after the backdrop and before the panel's own controls, so those still win.
func TestModalPanelsShieldTheirDismissTargetButNotTheirOwnControls(t *testing.T) {
	viewport := ui.Rect{Width: 800, Height: 600}
	for name, paint := range map[string]func(*ui.Canvas){
		"dialog": func(canvas *ui.Canvas) {
			BeginDialog(canvas, ui.LightTheme(), viewport, DialogOptions{Layer: 1, Scope: ui.Item(7), DismissID: ui.Item(1), CloseID: ui.Item(2)}).Paint("T", "D")
		},
		"sheet": func(canvas *ui.Canvas) {
			BeginSheet(canvas, ui.LightTheme(), viewport, SheetOptions{Layer: 1, Scope: ui.Item(7), DismissID: ui.Item(1), CloseID: ui.Item(2)}).Paint("T", "D")
		},
		"drawer": func(canvas *ui.Canvas) {
			BeginDrawer(canvas, ui.LightTheme(), viewport, DrawerOptions{Layer: 1, Scope: ui.Item(7), DismissID: ui.Item(1), CloseID: ui.Item(2)}).Paint("T", "D")
		},
	} {
		t.Run(name, func(t *testing.T) {
			var canvas ui.Canvas
			paint(&canvas)
			hits := canvas.Hits()
			order := map[string]int{}
			for index, hit := range hits {
				switch {
				case hit.Block:
					order["block"] = index
				case hit.ID == ui.Item(1):
					order["backdrop"] = index
				case hit.ID == ui.Item(2):
					order["close"] = index
				}
			}
			if len(order) != 3 || !(order["backdrop"] < order["block"] && order["block"] < order["close"]) {
				t.Fatalf("declaration order %v; want backdrop, then block, then the close control", order)
			}
			panel := hits[order["block"]].Bounds
			if panel.Width <= 0 || panel.Height <= 0 || panel.Width >= viewport.Width && panel.Height >= viewport.Height {
				t.Fatalf("block %+v must cover the panel, not the whole viewport", panel)
			}
		})
	}
}

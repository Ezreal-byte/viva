package components

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

// A sheet, drawer, and confirmation can name the control that takes focus when
// they open, as a dialog already can.
func TestSheetDrawerAndAlertDialogSetInitialFocus(t *testing.T) {
	var sheetCanvas ui.Canvas
	BeginSheet(&sheetCanvas, ui.LightTheme(), ui.Rect{Width: 800, Height: 600}, SheetOptions{
		Scope: ui.Item(1), DismissID: ui.Item(2), InitialFocusID: ui.Item(3),
	}).Paint("Title", "Description")
	if sheetCanvas.InitialFocus() != ui.Item(3) {
		t.Fatalf("sheet initial focus = %v", sheetCanvas.InitialFocus())
	}

	var drawerCanvas ui.Canvas
	BeginDrawer(&drawerCanvas, ui.LightTheme(), ui.Rect{Width: 800, Height: 600}, DrawerOptions{
		Scope: ui.Item(4), DismissID: ui.Item(5), InitialFocusID: ui.Item(6),
	}).Paint("Title", "Description")
	if drawerCanvas.InitialFocus() != ui.Item(6) {
		t.Fatalf("drawer initial focus = %v", drawerCanvas.InitialFocus())
	}

	var alertCanvas ui.Canvas
	BeginAlertDialog(&alertCanvas, ui.LightTheme(), ui.Rect{Width: 800, Height: 600}, AlertDialogOptions{
		Scope: ui.Item(7), ActionID: ui.Item(8), CancelID: ui.Item(9), InitialFocusID: ui.Item(8),
	}).Paint("Title", "Description", "Delete", "Cancel")
	if alertCanvas.InitialFocus() != ui.Item(8) {
		t.Fatalf("alert initial focus = %v", alertCanvas.InitialFocus())
	}
}

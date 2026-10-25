package components

import (
	"slices"
	"testing"
	"time"

	"github.com/vivaui/viva/ui"
)

func TestNavigationMenuCapturesDisclosureAndContentSemantics(t *testing.T) {
	var canvas ui.Canvas
	menu := BeginNavigationMenu(&canvas, ui.LightTheme(), ui.Rect{X: 10, Y: 12, Width: 500, Height: 36}, NavigationMenuOptions{
		Open: true, Content: ui.Rect{X: 10, Y: 54, Width: 360, Height: 150}, Layer: 6, Scope: ui.Item(31), DismissID: ui.Item(88),
	})
	trigger := menu.AddTrigger(NavigationMenuTrigger{ID: ui.Item(1), Label: "Products", Open: true})
	menu.AddLink(NavigationMenuLink{ID: ui.Item(2), Label: "Docs", Value: "/docs", Active: true})
	content := menu.BeginContent()
	link := content.AddLink(NavigationMenuLink{ID: ui.Item(3), Label: "Components", Description: "Reusable native UI building blocks", Value: "/components", Active: true})
	content.AddLink(NavigationMenuLink{ID: ui.Item(4), Label: "Disabled", Disabled: true})
	if trigger.Height != 36 || link != (ui.Rect{X: 18, Y: 62, Width: 344, Height: 56}) {
		t.Fatalf("geometry trigger=%+v link=%+v", trigger, link)
	}
	if canvas.EscapeTarget() != ui.Item(88) || !canvas.TrappedFocusScope().IsZero() {
		t.Fatalf("navigation overlay escape=%d trap=%d", canvas.EscapeTarget(), canvas.TrappedFocusScope())
	}
	if len(canvas.Hits()) != 4 {
		t.Fatalf("hits = %+v", canvas.Hits())
	}
	if hit := canvas.Hits()[0]; hit.Role != ui.RoleButton || !hit.Expanded || !hit.Scope.IsZero() {
		t.Fatalf("trigger hit = %+v", hit)
	}
	if hit := canvas.Hits()[2]; hit.Role != ui.RoleLink || hit.Scope != ui.Item(31) || !hit.Selected {
		t.Fatalf("content link = %+v", hit)
	}
	if hit := canvas.Hits()[3]; !hit.Disabled || hit.Focusable {
		t.Fatalf("disabled link = %+v", hit)
	}
	for _, rectangle := range canvas.Rectangles() {
		if rectangle.Bounds.Y >= 50 && rectangle.Layer != 6 {
			t.Fatalf("content rectangle layer = %+v", rectangle)
		}
	}
}

func TestNavigationMenuPaintIsDeterministicAndAllocationFreeAfterWarmup(t *testing.T) {
	var canvas ui.Canvas
	paint := func() {
		canvas.Reset()
		menu := BeginNavigationMenu(&canvas, ui.DarkTheme(), ui.Rect{X: 2, Y: 3, Width: 420, Height: 36}, NavigationMenuOptions{Open: true, Content: ui.Rect{X: 2, Y: 45, Width: 300, Height: 110}, Layer: 5, Scope: ui.Item(8), DismissID: ui.Item(9)})
		menu.AddTrigger(NavigationMenuTrigger{ID: ui.Item(1), Label: "Learn", Open: true})
		menu.AddLink(NavigationMenuLink{ID: ui.Item(2), Label: "GitHub", Value: "https://github.com"})
		content := menu.BeginContent()
		content.AddLink(NavigationMenuLink{ID: ui.Item(3), Label: "Introduction", Description: "Start building with Viva"})
		content.AddSeparator()
	}
	paint()
	rectangles := slices.Clone(canvas.Rectangles())
	texts := slices.Clone(canvas.Texts())
	hits := slices.Clone(canvas.Hits())
	paint()
	if !slices.Equal(rectangles, canvas.Rectangles()) || !slices.Equal(texts, canvas.Texts()) || !slices.Equal(hits, canvas.Hits()) {
		t.Fatal("navigation stream changed")
	}
	if allocations := testing.AllocsPerRun(100, paint); allocations != 0 {
		t.Fatalf("navigation paint allocations = %v", allocations)
	}
}

// The navigation menu's triggers take the same arrow/Home/End movement, and
// only move the open panel while one is open.
func TestNavigationMenuKeyDownMovesAndOpens(t *testing.T) {
	triggers := []NavigationMenuTrigger{{ID: ui.Item(1), Label: "Products"}, {ID: ui.Item(2), Label: "Docs"}, {ID: ui.Item(3), Label: "Pricing"}}
	focus, open, ok := NavigationMenuKeyDown(ui.Item(1), triggers, ui.ID{}, LTR, ui.KeyRight)
	if !ok || focus != ui.Item(2) || !open.IsZero() {
		t.Fatalf("Right = (%v, %v, %v)", focus, open, ok)
	}
	focus, open, ok = NavigationMenuKeyDown(ui.Item(1), triggers, ui.Item(1), LTR, ui.KeyRight)
	if !ok || focus != ui.Item(2) || open != ui.Item(2) {
		t.Fatalf("Right with open = (%v, %v, %v)", focus, open, ok)
	}
	if focus, open, ok = NavigationMenuKeyDown(ui.Item(2), triggers, open, LTR, ui.KeyDown); !ok || focus != ui.Item(2) || open != ui.Item(2) {
		t.Fatalf("Down = (%v, %v, %v)", focus, open, ok)
	}
	if focus, open, ok = NavigationMenuKeyDown(ui.Item(2), triggers, open, LTR, ui.KeyEscape); !ok || focus != ui.Item(2) || !open.IsZero() {
		t.Fatalf("Escape = (%v, %v, %v)", focus, open, ok)
	}
}

// The navigation menu switches its open panel on hover after the rest delay.
func TestNavigationMenuHoverSwitchesAfterRest(t *testing.T) {
	now := time.Unix(0, 0)
	var intent HoverIntent
	if next, wait := NavigationMenuHover(ui.Item(1), ui.Item(2), now, &intent, 50*time.Millisecond); next != ui.Item(1) || wait != 50*time.Millisecond {
		t.Fatalf("first hover = (%v, %v)", next, wait)
	}
	if next, wait := NavigationMenuHover(ui.Item(1), ui.Item(2), now.Add(60*time.Millisecond), &intent, 50*time.Millisecond); next != ui.Item(2) || wait != 0 {
		t.Fatalf("rested hover = (%v, %v)", next, wait)
	}
}

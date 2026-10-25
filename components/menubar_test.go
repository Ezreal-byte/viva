package components

import (
	"slices"
	"testing"
	"time"

	"github.com/vivaui/viva/ui"
)

func TestMenubarSeparatesBaseTriggersFromOverlayContent(t *testing.T) {
	theme := ui.LightTheme()
	var canvas ui.Canvas
	bar := BeginMenubar(&canvas, theme, ui.Rect{X: 8, Y: 10, Width: 300})
	fileBounds := bar.AddTrigger(MenubarTrigger{ID: ui.Item(1), Label: "File", Open: true})
	bar.AddTrigger(MenubarTrigger{ID: ui.Item(2), Label: "Edit", Disabled: true})
	content := BeginMenubarContent(&canvas, theme, ui.Rect{X: 8, Y: 54, Width: 160, Height: 150}, MenubarOverlayOptions{Layer: 15, Scope: ui.Item(73), DismissID: ui.Item(90)})
	content.AddItem(MenubarItem{ID: ui.Item(11), Label: "Auto Save", Kind: MenubarCheckbox, Checked: true})
	content.AddItem(MenubarItem{ID: ui.Item(12), Label: "UTF-8", Value: "utf8", Kind: MenubarRadio, Selected: true})
	content.AddItem(MenubarItem{ID: ui.Item(13), Label: "Share", Kind: MenubarSubmenu, Expanded: true})
	content.AddItem(MenubarItem{ID: ui.Item(14), Label: "Exit", Disabled: true})

	if bar.Bounds().Height != 36 || fileBounds.Y != 14 || fileBounds.Height != 28 {
		t.Fatalf("bar geometry = %+v file=%+v", bar.Bounds(), fileBounds)
	}
	if content.Bounds().Width != 160 || canvas.TrappedFocusScope() != ui.Item(73) || canvas.EscapeTarget() != ui.Item(90) {
		t.Fatalf("content contract = %+v scope=%d escape=%d", content.Bounds(), canvas.TrappedFocusScope(), canvas.EscapeTarget())
	}
	hits := nonBlockHits(canvas.Hits())
	if len(hits) != 6 {
		t.Fatalf("menubar hits = %+v", hits)
	}
	trigger := hits[0]
	if !trigger.Scope.IsZero() || trigger.Role != ui.RoleMenuItem || !trigger.Selected || !trigger.Expanded {
		t.Fatalf("open trigger = %+v", trigger)
	}
	if !hits[1].Disabled || hits[1].Focusable {
		t.Fatalf("disabled trigger = %+v", hits[1])
	}
	if !hits[2].Checked || !hits[3].Selected || !hits[4].Expanded || !hits[5].Disabled {
		t.Fatalf("content hit states = %+v", hits[2:])
	}
	for _, hit := range hits[2:] {
		if hit.Scope != ui.Item(73) || hit.Role != ui.RoleMenuItem {
			t.Fatalf("content hit = %+v", hit)
		}
	}
	baseLayers, overlayLayers := 0, 0
	for _, rectangle := range canvas.Rectangles() {
		if rectangle.Layer == 0 {
			baseLayers++
		}
		if rectangle.Layer == 15 {
			overlayLayers++
		}
	}
	if baseLayers == 0 || overlayLayers == 0 {
		t.Fatalf("layer split = base %d overlay %d", baseLayers, overlayLayers)
	}
}

func TestMenubarGeometryVariantsAndAllocationFreeRepaint(t *testing.T) {
	var canvas ui.Canvas
	canvas.SetInteraction(ui.ID{}, ui.ID{}, ui.Item(22), true)
	paint := func() {
		canvas.Reset()
		bar := BeginMenubar(&canvas, ui.DarkTheme(), ui.Rect{Width: 280})
		bar.AddTrigger(MenubarTrigger{ID: ui.Item(1), Label: "File"})
		bar.AddTrigger(MenubarTrigger{ID: ui.Item(2), Label: "View", Open: true})
		content := BeginMenubarContent(&canvas, ui.DarkTheme(), ui.Rect{Y: 44, Width: 200, Height: 84}, MenubarOverlayOptions{Layer: 4, Scope: ui.Item(5), DismissID: ui.Item(6)})
		if next := content.NextItemBounds(); next != (ui.Rect{X: 4, Y: 48, Width: 192, Height: 32}) {
			t.Fatalf("next item = %+v", next)
		}
		content.AddItem(MenubarItem{ID: ui.Item(22), Label: "Delete", Variant: MenubarItemDestructive, Shortcut: "⌫"})
		content.AddItem(MenubarItem{ID: ui.Item(23), Label: "Locked", Disabled: true})
	}
	paint()
	rectangles := slices.Clone(canvas.Rectangles())
	texts := slices.Clone(canvas.Texts())
	hits := slices.Clone(canvas.Hits())
	paint()
	if !slices.Equal(rectangles, canvas.Rectangles()) || !slices.Equal(texts, canvas.Texts()) || !slices.Equal(hits, canvas.Hits()) {
		t.Fatal("identical menubars produced different command streams")
	}
	if allocations := testing.AllocsPerRun(100, paint); allocations != 0 {
		t.Fatalf("menubar paint allocations = %v", allocations)
	}
}

func TestMenubarStateTransitionsAreAllocationFree(t *testing.T) {
	var state MenubarState
	if state.IsOpen() || !state.ActiveID().IsZero() || state.IsOpenFor(ui.Item(1)) {
		t.Fatalf("zero state = %d", state)
	}
	state = state.Open(ui.Item(11))
	if !state.IsOpen() || state.ActiveID() != ui.Item(11) || !state.IsOpenFor(ui.Item(11)) || state.IsOpenFor(ui.Item(12)) {
		t.Fatalf("open state = %d", state)
	}
	if got := state.Open(ui.ID{}); got != state {
		t.Fatalf("zero-ID open changed state to %d", got)
	}
	if got := state.Toggle(ui.Item(12)); got.ActiveID() != ui.Item(12) {
		t.Fatalf("switch toggle = %d", got)
	}
	if state.Toggle(ui.Item(11)).IsOpen() || state.Close().IsOpen() || state.Dismiss().IsOpen() || state.Escape().IsOpen() || state.OutsidePress().IsOpen() {
		t.Fatal("close transition left menubar open")
	}
	transition := func() {
		state = state.Open(ui.Item(11)).Toggle(ui.Item(12)).Toggle(ui.Item(12)).Open(ui.Item(13)).Dismiss().Open(ui.Item(14)).Escape().Open(ui.Item(15)).OutsidePress().Open(ui.Item(16)).Close()
	}
	if allocations := testing.AllocsPerRun(1000, transition); allocations != 0 {
		t.Fatalf("menubar state allocations = %v", allocations)
	}
}

// Left/Right and Home/End move between top-level triggers, switching the open
// menu while one is open; Down opens and Escape closes.
func TestMenubarKeyDownMovesAndSwitches(t *testing.T) {
	triggers := []MenubarTrigger{{ID: ui.Item(1), Label: "File"}, {ID: ui.Item(2), Label: "Edit", Disabled: true}, {ID: ui.Item(3), Label: "View"}}
	state, focus, ok := MenubarKeyDown(ui.Item(1), triggers, MenubarState{}, LTR, ui.KeyRight)
	if !ok || focus != ui.Item(3) || state.IsOpen() {
		t.Fatalf("Right = (%v, %v, %v), want View focused and closed", state, focus, ok)
	}
	state = state.Open(ui.Item(1))
	state, focus, ok = MenubarKeyDown(ui.Item(1), triggers, state, LTR, ui.KeyRight)
	if !ok || focus != ui.Item(3) || !state.IsOpenFor(ui.Item(3)) {
		t.Fatalf("Right with open = (%v, %v, %v), want View open", state, focus, ok)
	}
	if _, focus, _ := MenubarKeyDown(ui.Item(3), triggers, state, LTR, ui.KeyHome); focus != ui.Item(1) {
		t.Fatalf("Home = %v", focus)
	}
	if next, focus, ok := MenubarKeyDown(ui.Item(1), triggers, MenubarState{}, LTR, ui.KeyDown); !ok || focus != ui.Item(1) || !next.IsOpenFor(ui.Item(1)) {
		t.Fatalf("Down = (%v, %v, %v)", next, focus, ok)
	}
	if next, focus, ok := MenubarKeyDown(ui.Item(3), triggers, state, LTR, ui.KeyEscape); !ok || focus != ui.Item(3) || next.IsOpen() {
		t.Fatalf("Escape = (%v, %v, %v)", next, focus, ok)
	}
}

// Hovering a sibling trigger switches an open menubar after the rest delay.
func TestMenubarHoverSwitchesAfterRest(t *testing.T) {
	now := time.Unix(0, 0)
	var intent HoverIntent
	state := MenubarState{}.Open(ui.Item(1))
	if next, wait := MenubarHover(state, ui.Item(2), now, &intent, 100*time.Millisecond); !next.IsOpenFor(ui.Item(1)) || wait != 100*time.Millisecond {
		t.Fatalf("first hover = (%v, %v)", next.ActiveID(), wait)
	}
	if next, wait := MenubarHover(state, ui.Item(2), now.Add(120*time.Millisecond), &intent, 100*time.Millisecond); !next.IsOpenFor(ui.Item(2)) || wait != 0 {
		t.Fatalf("rested hover = (%v, %v)", next.ActiveID(), wait)
	}
	if next, _ := MenubarHover(MenubarState{}, ui.Item(2), now, &intent, 100*time.Millisecond); next.IsOpen() {
		t.Fatal("hover opened a closed menubar")
	}
}

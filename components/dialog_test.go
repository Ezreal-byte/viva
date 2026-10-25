package components

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

func TestDialogPaintsCenteredModalLayerAndTypedActions(t *testing.T) {
	var canvas ui.Canvas
	layout := BeginDialog(&canvas, ui.LightTheme(), ui.Rect{Width: 800, Height: 600}, DialogOptions{
		Layer: 7, Scope: ui.Item(91), DismissID: ui.Item(11), CloseID: ui.Item(12), ContentSize: ui.Size{Width: 400, Height: 220},
	}).Paint("Edit profile", "Make changes and save when finished.")
	if layout.Content != (ui.Rect{X: 200, Y: 190, Width: 400, Height: 220}) {
		t.Fatalf("content = %+v", layout.Content)
	}
	for _, rectangle := range canvas.Rectangles() {
		if rectangle.Layer != 7 {
			t.Fatalf("rectangle layer = %d, want 7", rectangle.Layer)
		}
	}
	for _, text := range canvas.Texts() {
		if text.Layer != 7 {
			t.Fatalf("text layer = %d, want 7", text.Layer)
		}
	}
	hits := targets(&canvas)
	if len(hits) != 2 {
		t.Fatalf("hits = %+v", hits)
	}
	if blocks := blocks(&canvas); len(blocks) != 1 || blocks[0].Bounds != layout.Content || blocks[0].Scope != ui.Item(91) {
		t.Fatalf("blocks = %+v, want one covering the dialog surface %+v", blocks, layout.Content)
	}
	if hits[0].ID != ui.Item(11) || hits[0].Role != ui.RoleButton || hits[0].Focusable || hits[0].Scope != ui.Item(91) {
		t.Fatalf("backdrop hit = %+v", hits[0])
	}
	if hits[1].ID != ui.Item(12) || hits[1].Role != ui.RoleButton || !hits[1].Focusable || hits[1].Scope != ui.Item(91) {
		t.Fatalf("close hit = %+v", hits[1])
	}
	if canvas.TrappedFocusScope() != ui.Item(91) || canvas.EscapeTarget() != ui.Item(11) {
		t.Fatalf("modal routing = scope %d escape %d", canvas.TrappedFocusScope(), canvas.EscapeTarget())
	}
}

func TestDialogBeginEndKeepsCallerContentInModalLayer(t *testing.T) {
	var canvas ui.Canvas
	dialog := BeginDialog(&canvas, ui.DarkTheme(), ui.Rect{Width: 640, Height: 480}, DialogOptions{Layer: 4, Scope: ui.Item(8), CloseID: ui.Item(9)})
	layout := dialog.Begin("Title", "Description")
	canvas.Fill(layout.Body, ui.DarkTheme().Accent)
	canvas.HitWithOptions(ui.Item(44), layout.Body, "Custom action", ui.HitOptions{Role: ui.RoleButton})
	dialog.End()
	canvas.Fill(ui.Rect{Width: 10, Height: 10}, ui.DarkTheme().Primary)
	if got := canvas.Rectangles()[len(canvas.Rectangles())-2].Layer; got != 4 {
		t.Fatalf("caller body layer = %d, want 4", got)
	}
	if got := canvas.Rectangles()[len(canvas.Rectangles())-1].Layer; got != 0 {
		t.Fatalf("post-dialog layer = %d, want 0", got)
	}
	if hit := canvas.Hits()[len(canvas.Hits())-1]; hit.ID != ui.Item(44) || hit.Scope != ui.Item(8) {
		t.Fatalf("custom hit = %+v", hit)
	}
}

func TestDialogDisabledActionsRemainSemanticAndInert(t *testing.T) {
	var canvas ui.Canvas
	BeginDialog(&canvas, ui.LightTheme(), ui.Rect{Width: 500, Height: 400}, DialogOptions{
		Layer: 2, Scope: ui.Item(3), DismissID: ui.Item(4), CloseID: ui.Item(5),
	}).Disabled().Paint("Locked", "This dialog cannot be dismissed yet.")
	if !canvas.EscapeTarget().IsZero() {
		t.Fatalf("escape target = %d, want 0", canvas.EscapeTarget())
	}
	for _, hit := range targets(&canvas) {
		if !hit.Disabled || hit.Role != ui.RoleButton {
			t.Fatalf("disabled hit = %+v", hit)
		}
	}
}

func TestDialogWithoutScopeDoesNotPopOrReplaceOuterFocusScope(t *testing.T) {
	var canvas ui.Canvas
	canvas.PushFocusScope(ui.Item(77))
	canvas.TrapFocus(ui.Item(77))
	BeginDialog(&canvas, ui.LightTheme(), ui.Rect{Width: 500, Height: 400}, DialogOptions{
		Layer: 2, CloseID: ui.Item(5),
	}).Paint("Nested", "No independent focus scope.")
	canvas.HitWithLabel(ui.Item(9), ui.Rect{Width: 10, Height: 10}, "Outer action")
	canvas.PopFocusScope()

	last := canvas.Hits()[len(canvas.Hits())-1]
	if last.Scope != ui.Item(77) || canvas.TrappedFocusScope() != ui.Item(77) {
		t.Fatalf("outer focus state was replaced: hit=%+v trap=%d", last, canvas.TrappedFocusScope())
	}
}

func TestDialogInitialFocusIsDeclaredOnlyForAnInteractiveModal(t *testing.T) {
	cases := []struct {
		name     string
		options  DialogOptions
		disabled bool
		want     int
	}{
		{"declared", DialogOptions{Layer: 2, Scope: ui.Item(3), CloseID: ui.Item(5), InitialFocusID: ui.Item(40)}, false, 40},
		{"default keeps focus where it was", DialogOptions{Layer: 2, Scope: ui.Item(3), CloseID: ui.Item(5)}, false, 0},
		{"inert dialog does not take focus", DialogOptions{Layer: 2, Scope: ui.Item(3), CloseID: ui.Item(5), InitialFocusID: ui.Item(40)}, true, 0},
		{"no trap, no modal focus", DialogOptions{Layer: 2, CloseID: ui.Item(5), InitialFocusID: ui.Item(40)}, false, 0},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var canvas ui.Canvas
			dialog := BeginDialog(&canvas, ui.LightTheme(), ui.Rect{Width: 500, Height: 400}, test.options)
			if test.disabled {
				dialog = dialog.Disabled()
			}
			dialog.Paint("Title", "Description")
			if got := canvas.InitialFocus(); got != ui.Item(int(test.want)) {
				t.Fatalf("initial focus = %d, want %d", got, test.want)
			}
		})
	}
}

func footerActions(t *testing.T, theme ui.Theme, disabled bool, width float32, actions ...ButtonAction) (*ui.Canvas, DialogLayout) {
	t.Helper()
	var canvas ui.Canvas
	dialog := BeginDialog(&canvas, theme, ui.Rect{Width: 1200, Height: 800}, DialogOptions{
		Layer: 3, Scope: ui.Item(20), CloseID: ui.Item(21), ContentSize: ui.Size{Width: width, Height: 240},
	})
	if disabled {
		dialog = dialog.Disabled()
	}
	layout := dialog.Begin("Title", "Description")
	dialog.Actions(layout, actions...)
	dialog.End()
	return &canvas, layout
}

func actionHits(canvas *ui.Canvas, ids ...uint32) []ui.Hit {
	var found []ui.Hit
	for _, id := range ids {
		for _, hit := range canvas.Hits() {
			if hit.ID == ui.Item(int(id)) {
				found = append(found, hit)
			}
		}
	}
	return found
}

func TestDialogActionsFitTheirLabelsAndSitAtTheFooterCorner(t *testing.T) {
	theme := ui.LightTheme()
	canvas, layout := footerActions(t, theme, false, 460,
		ButtonAction{ID: ui.Item(31), Label: "Cancel", Variant: ButtonSecondary},
		ButtonAction{ID: ui.Item(32), Label: "Add Host"},
	)
	hits := actionHits(canvas, 31, 32)
	if len(hits) != 2 {
		t.Fatalf("action hits = %+v", hits)
	}
	cancel, add := hits[0].Bounds, hits[1].Bounds
	if got, want := add.X+add.Width, layout.Footer.X+layout.Footer.Width; got != want {
		t.Errorf("the last action ends at %v, want the footer's right edge %v", got, want)
	}
	if gap := add.X - (cancel.X + cancel.Width); gap != theme.Metrics.GapItem {
		t.Errorf("gap between actions = %v, want the theme's item gap %v", gap, theme.Metrics.GapItem)
	}
	for index, label := range []string{"Cancel", "Add Host"} {
		want := BeginButton(canvas, theme, ui.ID{}, layout.Footer).SizeToFit(label).BoundsWidth()
		if got := hits[index].Bounds.Width; got != want {
			t.Errorf("%q button is %v wide, want %v: just its label and padding", label, got, want)
		}
	}
	if cancel.Width >= add.Width {
		t.Errorf("a short label is not narrower than a long one: %v vs %v", cancel.Width, add.Width)
	}
	for _, bounds := range []ui.Rect{cancel, add} {
		if bounds.Height != layout.Footer.Height || bounds.Y != layout.Footer.Y {
			t.Errorf("button %+v does not fill the footer %+v exactly", bounds, layout.Footer)
		}
	}
	for _, hit := range hits {
		if hit.Role != ui.RoleButton || !hit.Focusable || hit.Scope != ui.Item(20) {
			t.Errorf("action hit %+v is not a focusable button inside the dialog's scope", hit)
		}
	}
}

func TestDialogActionsFollowTheLabelAcrossLanguagesAndDensities(t *testing.T) {
	for _, density := range []ui.Density{ui.DensityCompact, ui.DensityRegular, ui.DensityLarge} {
		theme := ui.DarkTheme()
		theme.Metrics = theme.Metrics.WithDensity(density)
		for _, labels := range [][2]string{{"Cancel", "Add Host"}, {"取消", "添加主机"}, {"拒绝", "信任并保存"}} {
			canvas, layout := footerActions(t, theme, false, 460,
				ButtonAction{ID: ui.Item(31), Label: labels[0], Variant: ButtonOutline}, ButtonAction{ID: ui.Item(32), Label: labels[1]})
			hits := actionHits(canvas, 31, 32)
			if len(hits) != 2 {
				t.Fatalf("%v: hits = %+v", labels, hits)
			}
			for index, hit := range hits {
				want := BeginButton(canvas, theme, ui.ID{}, layout.Footer).SizeToFit(labels[index]).BoundsWidth()
				if hit.Bounds.Width != want {
					t.Errorf("density %v, %q: width %v, want %v", density, labels[index], hit.Bounds.Width, want)
				}
				if hit.Bounds.Height > layout.Footer.Height {
					t.Errorf("density %v: a button %v high overflows the %v footer", density, hit.Bounds.Height, layout.Footer.Height)
				}
			}
		}
	}
}

func TestDialogActionsShrinkTogetherInANarrowFooterAndNeverOverflow(t *testing.T) {
	canvas, layout := footerActions(t, ui.LightTheme(), false, 150,
		ButtonAction{ID: ui.Item(31), Label: "A rather long cancel label", Variant: ButtonOutline},
		ButtonAction{ID: ui.Item(32), Label: "An equally long confirm label"},
	)
	hits := actionHits(canvas, 31, 32)
	if len(hits) != 2 {
		t.Fatalf("hits = %+v", hits)
	}
	first, last := hits[0].Bounds, hits[1].Bounds
	if first.X < layout.Footer.X-0.01 || last.X+last.Width > layout.Footer.X+layout.Footer.Width+0.01 {
		t.Errorf("actions %+v %+v leave the footer %+v", first, last, layout.Footer)
	}
	if first.Width <= 0 || last.Width <= 0 {
		t.Errorf("a shrunk action vanished: %+v %+v", first, last)
	}
}

func TestDialogActionsVariantsDisabledAndDisabledDialogs(t *testing.T) {
	canvas, _ := footerActions(t, ui.DarkTheme(), false, 460,
		ButtonAction{ID: ui.Item(31), Label: "Cancel", Variant: ButtonOutline},
		ButtonAction{ID: ui.Item(32), Label: "Delete", Variant: ButtonDestructive, Disabled: true},
	)
	hits := actionHits(canvas, 31, 32)
	if hits[0].Disabled || !hits[1].Disabled {
		t.Errorf("per-action Disabled not honored: %+v", hits)
	}
	canvas, _ = footerActions(t, ui.DarkTheme(), true, 460, ButtonAction{ID: ui.Item(31), Label: "Cancel"}, ButtonAction{ID: ui.Item(32), Label: "OK"})
	for _, hit := range actionHits(canvas, 31, 32) {
		if !hit.Disabled {
			t.Errorf("a disabled dialog left %+v live", hit)
		}
	}
}

func TestDialogActionsWithNothingToPaintOrNowhereToPaintDoNothing(t *testing.T) {
	canvas, _ := footerActions(t, ui.LightTheme(), false, 460)
	if len(actionHits(canvas, 31)) != 0 {
		t.Error("no actions painted a hit")
	}
	var empty ui.Canvas
	BeginDialog(&empty, ui.LightTheme(), ui.Rect{}, DialogOptions{}).Actions(DialogLayout{}, ButtonAction{ID: ui.Item(1), Label: "x"})
	BeginDialog(nil, ui.LightTheme(), ui.Rect{Width: 100, Height: 100}, DialogOptions{}).Actions(DialogLayout{Footer: ui.Rect{Width: 50, Height: 30}}, ButtonAction{ID: ui.Item(1), Label: "x"})
	if len(empty.Hits()) != 0 {
		t.Error("a dialog with no viewport painted something")
	}
}

func TestDialogActionsAllocateNothingOncePainted(t *testing.T) {
	var canvas ui.Canvas
	theme := ui.LightTheme()
	paint := func() {
		canvas.Reset()
		dialog := BeginDialog(&canvas, theme, ui.Rect{Width: 1200, Height: 800}, DialogOptions{Layer: 3, Scope: ui.Item(20), CloseID: ui.Item(21)})
		layout := dialog.Begin("Title", "Description")
		dialog.Actions(layout,
			ButtonAction{ID: ui.Item(31), Label: "Cancel", Variant: ButtonOutline},
			ButtonAction{ID: ui.Item(32), Label: "Confirm"},
		)
		dialog.End()
	}
	paint()
	if allocs := testing.AllocsPerRun(50, paint); allocs != 0 {
		t.Errorf("painting a dialog with actions allocates %v times", allocs)
	}
}

func TestDialogFitBodyMakesTheBodyExactlyAsTallAsAsked(t *testing.T) {
	compact, large := ui.LightTheme(), ui.DarkTheme()
	compact.Metrics = compact.Metrics.WithDensity(ui.DensityCompact)
	large.Metrics = large.Metrics.WithDensity(ui.DensityLarge)
	for _, theme := range []ui.Theme{ui.LightTheme(), compact, large} {
		for _, description := range []string{"", "Something to say."} {
			var canvas ui.Canvas
			layout := BeginDialog(&canvas, theme, ui.Rect{Width: 1200, Height: 900}, DialogOptions{
				Layer: 3, Scope: ui.Item(5), CloseID: ui.Item(6), ContentSize: ui.Size{Width: 460, Height: 999},
				FitBody: true, BodyHeight: 62,
			}).Paint("Title", description)
			if layout.Body.Height != 62 {
				t.Fatalf("description %q: body is %v tall, want 62", description, layout.Body.Height)
			}
			if layout.Content.Width != 460 || layout.Content.Height >= 999 {
				t.Fatalf("content = %+v: the guessed height was not replaced", layout.Content)
			}
			// Nothing is left between the body and the footer beyond the dialog's own gaps.
			gap := layout.Footer.Y - (layout.Body.Y + layout.Body.Height)
			if want := dialogGapsBelowBody * theme.Metrics.GapItem; gap != want {
				t.Fatalf("description %q: %v between body and footer, want %v", description, gap, want)
			}
		}
	}
}

func TestDialogFitBodyStillFitsTheViewport(t *testing.T) {
	var canvas ui.Canvas
	layout := BeginDialog(&canvas, ui.LightTheme(), ui.Rect{Width: 300, Height: 200}, DialogOptions{
		FitBody: true, BodyHeight: 900,
	}).Paint("Title", "")
	if layout.Content.Height != 168 {
		t.Fatalf("content height = %v, want the viewport less its margins", layout.Content.Height)
	}
}

func TestDialogDescriptionWrapsAndTheHeaderGrowsWithItsLines(t *testing.T) {
	theme := ui.LightTheme()
	paint := func(description string) DialogLayout {
		var canvas ui.Canvas
		return BeginDialog(&canvas, theme, ui.Rect{Width: 1200, Height: 900}, DialogOptions{
			Layer: 3, Scope: ui.Item(5), CloseID: ui.Item(6), ContentSize: ui.Size{Width: 360}, FitBody: true, BodyHeight: 40,
		}).Paint("Delete host?", description)
	}
	one := paint("This cannot be undone.")
	many := paint("Every saved session, port forward and snippet of this host is deleted with it, and this cannot be undone afterwards.")
	if many.Header.Height <= one.Header.Height || many.Content.Height <= one.Content.Height {
		t.Fatalf("header %v / dialog %v for a long description, not taller than %v / %v", many.Header.Height, many.Content.Height, one.Header.Height, one.Content.Height)
	}
	if many.Body.Height != 40 {
		t.Fatalf("body = %v tall, want the 40 asked for however long the description is", many.Body.Height)
	}
	if got := BeginDialog(nil, theme, ui.Rect{Width: 1200, Height: 900}, DialogOptions{ContentSize: ui.Size{Width: 360}, FitBody: true}).Layout("x"); got.Header.Height == 0 {
		t.Fatal("Layout without a canvas lost the header")
	}
}

func TestOverlayButtonSizeSetsTheFooterAndTheButtonsInIt(t *testing.T) {
	theme := ui.LightTheme()
	viewport := ui.Rect{Width: 900, Height: 600}
	small := theme.Metrics.ControlHeight - 4
	def := theme.Metrics.ControlHeight

	fit := func(size ButtonSize) (ui.Rect, ui.Rect) {
		var canvas ui.Canvas
		dialog := BeginDialog(&canvas, theme, viewport, DialogOptions{
			Layer: 3, Scope: ui.Item(1), CloseID: ui.Item(2),
			FitBody: true, BodyHeight: 100, ButtonSize: size,
		})
		layout := dialog.Begin("Edit", "")
		dialog.Actions(layout, ButtonAction{ID: ui.Item(3), Label: "Save"})
		dialog.End()
		for _, hit := range canvas.Hits() {
			if hit.ID == ui.Item(3) {
				return layout.Content, hit.Bounds
			}
		}
		t.Fatal("the action was not declared")
		return ui.Rect{}, ui.Rect{}
	}
	defaultContent, defaultButton := fit(ButtonSizeDefault)
	smallContent, smallButton := fit(ButtonSizeSmall)
	if defaultButton.Height != def || smallButton.Height != small {
		t.Fatalf("buttons are %v and %v high, want %v and %v", defaultButton.Height, smallButton.Height, def, small)
	}
	if got := defaultContent.Height - smallContent.Height; got != def-small {
		t.Errorf("a Small footer makes the fitted dialog %v shorter, want %v", got, def-small)
	}

	var canvas ui.Canvas
	sheet := BeginSheet(&canvas, theme, viewport, SheetOptions{Layer: 3, Scope: ui.Item(1), CloseID: ui.Item(2), ButtonSize: ButtonSizeSmall})
	if footer := sheet.Begin("Edit", "").Footer; footer.Height != small {
		t.Errorf("sheet footer is %v high, want %v", footer.Height, small)
	}
	sheet.End()
	canvas.Reset()
	drawer := BeginDrawer(&canvas, theme, viewport, DrawerOptions{Layer: 3, Scope: ui.Item(1), CloseID: ui.Item(2), ButtonSize: ButtonSizeSmall})
	if footer := drawer.Begin("Edit", "").Footer; footer.Height != small {
		t.Errorf("drawer footer is %v high, want %v", footer.Height, small)
	}
	drawer.End()
	canvas.Reset()
	alert := BeginAlertDialog(&canvas, theme, viewport, AlertDialogOptions{Layer: 3, Scope: ui.Item(1), ActionID: ui.Item(3), CancelID: ui.Item(4), ButtonSize: ButtonSizeSmall})
	alert.Begin("Delete?", "Gone for good.", "Delete", "Cancel")
	alert.End()
	for _, hit := range canvas.Hits() {
		if (hit.ID == ui.Item(3) || hit.ID == ui.Item(4)) && hit.Bounds.Height != small {
			t.Errorf("alert button %v is %v high, want %v", hit.ID, hit.Bounds.Height, small)
		}
	}
}

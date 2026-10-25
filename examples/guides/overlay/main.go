package main

import (
	"log"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/flex"
	"github.com/vivaui/viva/ui"
)

// Identities are names in a tree, not numbers: the application declares the
// tree once and every control it paints is a name inside a feature of it.
var (
	tree   = ui.Tree{}
	page   = tree.Feature("overlays")
	dialog = tree.Sub(page, tree.Name("dialog"))

	trigger     = tree.ID(page, "trigger")
	menuDismiss = tree.ID(page, "menu-dismiss")
	menuNew     = tree.ID(page, "menu-new")
	menuDelete  = tree.ID(page, "menu-delete")

	dialogScope   = tree.ID(dialog, "scope")
	dialogClose   = tree.ID(dialog, "close")
	dialogCancel  = tree.ID(dialog, "cancel")
	dialogConfirm = tree.ID(dialog, "confirm")
)

type model struct {
	menuOpen   bool
	dialogOpen bool
	status     string
	view       flex.View
}

func newConfig(m *model) app.Config {
	theme := ui.LightTheme()
	config := app.Config{Tree: &tree, Title: "Overlays", Width: 640, Height: 360}

	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		bounds := ui.Rect{Width: size.Width, Height: size.Height}
		canvas.Fill(bounds, theme.Background)

		m.view.Draw(canvas, theme, bounds, func(f *flex.Frame) {
			f.Column().Fill().Pad(24).Gap(12).Items(flex.Start).Children(func() {
				actions := f.Button(trigger, "Actions").Outline()
				if actions.Clicked() {
					m.menuOpen = !m.menuOpen
				}
				f.Text(m.status).Muted()

				if m.menuOpen {
					// A popup takes no room and is painted over the page. The first one
					// covers the window so a press outside the menu lands on menuDismiss,
					// and Escape is routed to the same identity.
					canvas.SetEscapeTarget(menuDismiss)
					f.Popup().At(0, 0).Fill().Hit(menuDismiss, "Close menu").Children(func() {})
					if canvas.Clicked(menuDismiss) {
						m.menuOpen = false
					}
					f.Popup().BelowBox(actions, 4, flex.Start).Width(180).Pad(4).Background(theme.Popover).
						Border(1, theme.Border).Radius(theme.Metrics.RadiusControl).Shadow(ui.ShadowMD).Children(func() {
						if f.Button(menuNew, "New file").Ghost().Clicked() {
							m.menuOpen, m.status = false, "Created"
						}
						if f.Button(menuDelete, "Delete…").Ghost().Destructive().Clicked() {
							m.menuOpen, m.dialogOpen = false, true
						}
					})
				}

				if m.dialogOpen {
					d := f.Dialog(flex.DialogSpec{
						Title: "Delete file?", Description: "This cannot be undone.", Width: 400,
						Scope: dialogScope, DismissID: dialogCancel, CloseID: dialogClose,
					})
					d.Footer(func() {
						f.Button(dialogCancel, "Cancel").Outline()
						if f.Button(dialogConfirm, "Delete").Destructive().Clicked() {
							m.dialogOpen, m.status = false, "Deleted"
						}
					})
					if d.Dismissed() {
						m.dialogOpen = false
					}
				}
			})
		})
	}
	return config
}

func main() {
	var canvas ui.Canvas
	if err := app.RunCanvas(newConfig(&model{}), &canvas); err != nil {
		log.Fatal(err)
	}
}

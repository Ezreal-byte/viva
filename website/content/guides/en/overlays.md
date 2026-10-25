---
title: Dialogs, menus and popups
summary: Layers, focus scopes and dismissal, with a dropdown menu that closes on an outside press and a modal dialog.
order: 9
group: guides
---

Overlays are painted last, on a higher layer, and they change where input can go. In `flex` an overlay is declared in the page, in the place where it is open, and takes no room there. Viva gives every overlay three pieces.

- **A layer.** `f.Popup()` and `f.Dialog` are painted over the page, and hits declared on them win over hits beneath.
- **A focus scope.** `DialogSpec.Scope` traps Tab inside a dialog while it is open. Nothing behind it can take keyboard focus.
- **A dismiss identity.** The identity that reports a click when the user presses Escape or presses outside the overlay. You read it where the overlay is declared, like any other click.

As everywhere in Viva, whether an overlay is open is your state. You declare it, or you do not.

<!-- example: overlay/main.go -->
```go
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
```

## Menus and popups

`f.Popup()` takes no room in its container, is as big as its content, is kept inside the window and is painted over everything. Place it with `BelowBox`, `AboveBox` or `BesideBox` for an element of the page, or with `At(x, y)`, `Below(rect, ...)` and `Above(rect, ...)` for a point or rectangle; it uses the side you ask for and flips to the other when there is no room. Keep the trigger's `Box` (`actions := f.Button(...)`) to anchor to it: `Box.Rect()` is zero until the frame ends. The rows of a menu are ordinary buttons or `Hit` boxes inside it, and a menu that may be long needs a `MaxHeight` and a `ScrollArea`.

A context menu is the same popup placed `At` the pointer. Open it from `OnPointer` on a `PointerButtonSecondary` down.

### Closing on an outside press

A popup does not close itself, because whether it is open is your state. The example declares a second, window-sized popup under the menu with a `Hit`: a press anywhere outside the menu, including on the button that opened it, lands on that identity, and `canvas.SetEscapeTarget` sends Escape to the same identity. Read it with `canvas.Clicked` and close the menu. The press does nothing else, so one click on the page behind does not both close the menu and activate whatever was under it.

The example's test checks all of it: the press at the far corner lands on the dismiss identity, Escape closes the menu, and choosing Delete… closes it and opens the dialog.

## Dialogs

`f.Dialog(flex.DialogSpec{...})` dims the window, paints a centered surface with a title, a description and a close button, and traps focus. Declare it only while it is open. `Body` declares its content and `Footer` its buttons, which sit against the right edge in the order given; the surface is as tall as the body needs, so a short body never leaves a gap above the footer. `Dismissed()` is true in the paint that delivers a click on the scrim, the close button or Escape, and the dialog routes Escape itself.

`DismissID` is the identity the scrim and Escape report, and `CloseID` the corner button's. The example gives `DismissID` to Cancel's identity, so the scrim and Escape behave as Cancel. Opening a dialog does not move focus unless you set `InitialFocusID`, the control that should take it. Use it for a dialog with an input in it.

## Selects and the rest

`f.Select` is a drop-down list: a `flex.SelectState` and a scope of its own in your model, and the list opens under the trigger, or above it when there is no room, painted over everything. Inside a dialog it opens above the dialog. Tooltips and other non-modal panels are popups that declare no dismiss identity, so they neither trap focus nor swallow outside presses. Combobox, command palette, menubar and navigation menu follow the same layer, scope and dismiss conventions; the [component catalog](/en/ui/dialog/) has a runnable example of each, and the [API reference](/en/api/components/) has every option.

## Entrance and exit

A dialog, a sheet, a drawer and the menus, popovers, tooltips and toasts of the component recipes fade and move in when you start painting them and out when you stop, as the shadcn classes `animate-in` and `animate-out` do. You keep nothing for it: stop declaring the dialog and the canvas draws its last frame out over the next 100 ms, without hit regions, a focus trap or an Escape target, so it never takes a press or the keyboard from what is under it. Painting it again while it leaves turns it around from where it is. It needs a stable identity (the dialog's `DismissID`, a menu's `DismissID` or `Scope`), a menu, select or combobox needs its `End()` call, and a theme whose `Transition` has no duration, reduced motion, or a runtime without a frame clock shows the final state at once.

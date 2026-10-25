package main

import (
	"log"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/flex"
	"github.com/vivaui/viva/ui"
)

var (
	tree = ui.Tree{}
	page = tree.Feature("layout")

	save   = tree.ID(page, "save")
	cancel = tree.ID(page, "cancel")
)

type model struct {
	saved bool
	notes bool
	view  flex.View
}

func newConfig(m *model) app.Config {
	theme := ui.LightTheme()
	config := app.Config{Tree: &tree, Title: "Layout", Width: 420, Height: 220}

	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		canvas.Fill(ui.Rect{Width: size.Width, Height: size.Height}, theme.Background)
		bounds := ui.Rect{Width: size.Width, Height: size.Height}

		m.view.Draw(canvas, theme, bounds, func(f *flex.Frame) {
			f.Column().Fill().Pad(24).Gap(12).Children(func() {
				f.Text("Deploy settings").FontSize(18).Bold()
				f.Text("Where the build goes when it passes.").Muted()
				f.Divider()
				f.Row().Items(flex.Center).Gap(8).Children(func() {
					f.Text("Notify the team")
					f.Spacer()
					f.Switch(tree.ID(page, "notes"), m.notes).Label("Notify the team")
				})
				f.Spacer()
				f.Row().Gap(8).Justify(flex.End).Children(func() {
					f.Button(cancel, "Cancel").Outline()
					if f.Button(save, "Save").Clicked() {
						m.saved = true
					}
				})
			})
		})
	}
	return config
}

func main() {
	m := &model{}
	var canvas ui.Canvas
	if err := app.RunCanvas(newConfig(m), &canvas); err != nil {
		log.Fatal(err)
	}
}

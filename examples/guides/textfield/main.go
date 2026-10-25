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
	tree = ui.Tree{}
	page = tree.Feature("form")

	nameField = tree.ID(page, "name")
	save      = tree.ID(page, "save")
)

type model struct {
	name  flex.Field
	saved string
	view  flex.View
}

func newConfig(m *model) app.Config {
	theme := ui.LightTheme()
	config := app.Config{Tree: &tree, Title: "Form", Width: 480, Height: 240}

	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		bounds := ui.Rect{Width: size.Width, Height: size.Height}
		canvas.Fill(bounds, theme.Background)

		m.view.Draw(canvas, theme, bounds, func(f *flex.Frame) {
			f.Column().Fill().Pad(24).Gap(12).Items(flex.Start).Children(func() {
				f.Field(&m.name, nameField, "Name", "Your name").Width(320)
				if f.Button(save, "Save").Clicked() {
					m.saved = m.name.Text()
				}
				if m.saved != "" {
					f.Text("Saved " + m.saved).Muted()
				}
			})
		})
	}

	// The platform asks which editor belongs to the focused field; Fields answers
	// for every flex.Field the application declares.
	config.TextInput = flex.Fields(&m.name)
	return config
}

func main() {
	var canvas ui.Canvas
	if err := app.RunCanvas(newConfig(&model{}), &canvas); err != nil {
		log.Fatal(err)
	}
}

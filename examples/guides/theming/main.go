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
	page = tree.Feature("theming")

	toggle = tree.ID(page, "toggle")
)

type model struct {
	dark bool
	view flex.View
}

// theme is the active theme: a value, so changing appearance is changing which
// value Paint reads.
func (m *model) theme() ui.Theme {
	theme := ui.LightTheme()
	if m.dark {
		theme = ui.DarkTheme()
	}
	// Tokens are plain fields. Override one to brand the application.
	theme.Primary = ui.Color{Red: 0.9, Green: 0.45, Blue: 0.1, Alpha: 1}
	theme.PrimaryForeground = ui.Color{Red: 1, Green: 1, Blue: 1, Alpha: 1}
	return theme
}

func newConfig(m *model) app.Config {
	config := app.Config{Tree: &tree, Title: "Theme", Width: 480, Height: 240}
	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		theme := m.theme()
		bounds := ui.Rect{Width: size.Width, Height: size.Height}
		canvas.Fill(bounds, theme.Background)

		m.view.Draw(canvas, theme, bounds, func(f *flex.Frame) {
			f.Column().Fill().Pad(24).Gap(12).Items(flex.Start).Children(func() {
				if f.Button(toggle, "Toggle theme").Clicked() {
					m.dark = !m.dark
				}
				f.Text("Hello")
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

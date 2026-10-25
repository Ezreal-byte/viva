package main

import (
	"fmt"
	"log"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/flex"
	"github.com/vivaui/viva/ui"
)

// Identities are names in a tree, not numbers: the application declares the
// tree once and every control it paints is a name inside a feature of it.
var (
	tree = ui.Tree{}
	page = tree.Feature("hello")

	click = tree.ID(page, "click")
)

type model struct {
	clicks int
	view   flex.View
}

// newConfig describes the window. Keeping it in a function lets a test drive
// the same application without opening anything.
func newConfig(m *model) app.Config {
	theme := ui.LightTheme()
	config := app.Config{Tree: &tree, Title: "Hello", Width: 480, Height: 240}

	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		bounds := ui.Rect{Width: size.Width, Height: size.Height}
		canvas.Fill(bounds, theme.Background)

		m.view.Draw(canvas, theme, bounds, func(f *flex.Frame) {
			f.Column().Fill().Pad(24).Gap(12).Children(func() {
				// A button is handled where it is declared: Clicked is true for the
				// one paint that delivers its click, and only then may state change.
				if f.Button(click, fmt.Sprintf("Clicked %d times", m.clicks)).Clicked() {
					m.clicks++
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

package main

import (
	"fmt"
	"log"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/components"
	"github.com/vivaui/viva/ui"
)

// Identities are names in a tree, not numbers: the application declares the
// tree once and every control it paints is a name inside a feature of it.
var (
	tree = ui.Tree{}
	page = tree.Feature("drawing")

	hitNew = page.ID(tree.Name("new"))
)

func paint(canvas *ui.Canvas, size ui.Size) {
	theme := ui.LightTheme()

	// Lay out by slicing rectangles; none of this allocates.
	page := ui.Rect{Width: size.Width, Height: size.Height}.Inset(ui.All(24))
	column := ui.BeginColumn(page, 16)

	canvas.Label("Dashboard", column.Next(32), ui.TextStyle{Size: 20, Weight: ui.WeightSemibold, Color: theme.Foreground})

	toolbar := ui.BeginRow(column.Next(36), 8)
	components.BeginButton(canvas, theme, hitNew, toolbar.Next(100)).Default("New")

	cards := column.NextFlex() // whatever height is left
	grid := ui.BeginGrid(cards, ui.GridSpec{MinCellWidth: 160, CellHeight: 96, Gap: 12})
	for index := 1; index <= 6; index++ {
		cell := grid.Next()
		canvas.RoundedBorder(cell, 8, 1, theme.Border, theme.Card)
		canvas.Label(fmt.Sprintf("Card %d", index), cell.Inset(ui.All(12)), ui.TextStyle{Size: 14, Color: theme.CardForeground})
	}
}

func main() {
	var canvas ui.Canvas
	config := app.Config{Tree: &tree, Title: "Layout", Width: 720, Height: 480, Paint: paint}
	if err := app.RunCanvas(config, &canvas); err != nil {
		log.Fatal(err)
	}
}

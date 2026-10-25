package gallery

import "github.com/vivaui/viva/ui"

const (
	// exampleHeaderHeight holds an example's title and up to two description
	// lines above its stage.
	exampleHeaderHeight = float32(62)
	// exampleStageHeight is the room an example gets. The demos are written for
	// roughly 320-760 by 340 logical pixels.
	exampleStageHeight = float32(340)
)

// exampleGrid lays a component's examples out: as many columns as fit at the
// minimum stage width, stretched to fill the page.
var exampleGrid = ui.GridSpec{MinCellWidth: 420, CellHeight: exampleHeaderHeight + exampleStageHeight, Gap: 28}

// componentHeight returns the height of a component page with that many
// examples at one page width.
func componentHeight(width float32, examples int) float32 {
	return pageHeaderHeight + exampleGrid.Height(width, examples)
}

// paintComponent shows every example of the open component, each live on its
// own stage.
func (state *State) paintComponent(canvas *ui.Canvas, theme ui.Theme, content ui.Rect, viewport ui.Rect) {
	component := &catalog[state.page-1]
	body := pageHeader(canvas, theme, content, component.Title, componentLines[state.page-1])
	grid := ui.BeginGrid(body, exampleGrid)
	for index := range state.live {
		cell := grid.Next()
		example := &component.Examples[index]
		canvas.Label(example.Title, ui.Rect{X: cell.X, Y: cell.Y, Width: cell.Width, Height: 22}, ui.TextStyle{
			Size: 14, Weight: ui.WeightSemibold, Color: theme.Foreground,
		})
		first, rest := wrapLine(example.Description, 12, cell.Width)
		second, _ := wrapLine(rest, 12, cell.Width)
		canvas.Label(first, ui.Rect{X: cell.X, Y: cell.Y + 24, Width: cell.Width, Height: 18}, ui.TextStyle{
			Size: 12, Color: theme.MutedForeground,
		})
		if second != "" {
			canvas.Label(second, ui.Rect{X: cell.X, Y: cell.Y + 42, Width: cell.Width, Height: 18}, ui.TextStyle{
				Size: 12, Color: theme.MutedForeground,
			})
		}
		state.paintStage(canvas, theme, index, ui.Rect{
			X: cell.X, Y: cell.Y + exampleHeaderHeight, Width: cell.Width, Height: exampleStageHeight,
		}, viewport)
	}
}

// paintStage frames one example and embeds what it paints. The example runs on
// a canvas of its own, as it does in a standalone preview, and its hit IDs are
// moved into the example's window of the page's ID space.
func (state *State) paintStage(canvas *ui.Canvas, theme ui.Theme, slot int, stage ui.Rect, viewport ui.Rect) {
	view := &state.live[slot]
	canvas.Border(stage, 1, theme.Border, theme.Background)
	inner := stage.Inset(ui.All(1))
	view.origin = ui.Point{X: inner.X, Y: inner.Y}
	view.size = ui.Size{Width: inner.Width, Height: inner.Height}
	if inner.Y+inner.Height <= viewport.Y || inner.Y >= viewport.Y+viewport.Height {
		return // scrolled out of sight: nothing to paint or reach
	}

	mount := exampleMount(slot)
	view.canvas.BeginEmbedded(canvas, mount)
	view.demo.SetDark(!state.lightMode)
	view.demo.SetDensity(state.density)
	view.demo.Paint(&view.canvas, view.size)
	view.canvas.EndFrame()

	canvas.Clip(inner)
	canvas.Embed(&view.canvas, view.origin, mount)
	canvas.Unclip()
}

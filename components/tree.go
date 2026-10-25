package components

import "github.com/vivaui/viva/ui"

// Tree paints a compact vertical hierarchy. Applications own the hierarchy,
// expansion, selection, and scrolling; the component owns indentation,
// disclosure geometry, focus/hover feedback, and typed hit semantics.
type Tree struct {
	canvas    *ui.Canvas
	theme     ui.Theme
	bounds    ui.Rect
	cursorY   float32
	rowHeight float32
	indent    float32
}

// TreeItem describes one already-flattened visible tree row. Branches use a
// distinct DisclosureID so expanding a branch never also activates its row.
// Depth is clamped to 31 to keep malformed input from pushing content away.
type TreeItem struct {
	ID           ui.ID
	DisclosureID ui.ID
	Label        string
	Depth        uint8
	Branch       bool
	Expanded     bool
	Selected     bool
	Disabled     bool
}

// TreeRow is the geometry reserved for caller-owned row content.
type TreeRow struct {
	Bounds     ui.Rect
	Content    ui.Rect
	Disclosure ui.Rect
}

// BeginTree starts a tree declaration. Keep item IDs and flattened order stable
// between paints so retained commands, hover, and focus remain stable.
//
// shadcn has no tree; the reference is the file tree of AI Elements, whose rows are
// small ghost buttons (one control tall) and whose children are indented 16px
// (ml-4). RowHeight and Indent override both.
func BeginTree(canvas *ui.Canvas, theme ui.Theme, bounds ui.Rect) Tree {
	return Tree{
		canvas: canvas, theme: theme, bounds: bounds, cursorY: bounds.Y,
		rowHeight: theme.Metrics.ControlHeight,
		indent:    16 * theme.Metrics.ControlHeight / 32,
	}
}

// RowHeight changes the height of subsequent rows.
func (tree Tree) RowHeight(height float32) Tree {
	if height > 0 {
		tree.rowHeight = height
	}
	return tree
}

// Indent changes the offset applied for each hierarchy depth.
func (tree Tree) Indent(width float32) Tree {
	if width >= 0 {
		tree.indent = width
	}
	return tree
}

// Item paints one visible row and returns its caller-owned content area. The
// component never walks children or allocates a backing model.
func (tree *Tree) Item(item TreeItem) TreeRow {
	if tree.canvas == nil || tree.bounds.Width <= 0 || tree.rowHeight <= 0 {
		return TreeRow{}
	}
	depth := min(int(item.Depth), 31)
	row := ui.Rect{X: tree.bounds.X, Y: tree.cursorY, Width: tree.bounds.Width, Height: tree.rowHeight}
	tree.cursorY += tree.rowHeight

	enabled := !item.Disabled && !item.ID.IsZero()
	hovered := enabled && tree.canvas.Hovered() == item.ID
	focused := enabled && tree.canvas.FocusVisible() == item.ID
	radius := tree.theme.Metrics.RadiusControl
	if item.Selected || hovered {
		background := tree.theme.Muted
		if item.Selected {
			background = tree.theme.Accent
		}
		tree.canvas.Rounded(row, radius, background)
	}
	if focused {
		ring := tree.theme.Ring
		ring.Alpha *= 0.50
		tree.canvas.Rounded(treeExpand(row, 2), radius+2, ring)
	}

	leading := row.X + float32(depth)*tree.indent
	disclosure := ui.Rect{X: leading, Y: row.Y + (row.Height-20)/2, Width: 20, Height: 20}
	contentX := leading
	if item.Branch {
		contentX += 24
		color := tree.theme.MutedForeground
		if item.Disabled {
			color.Alpha *= 0.50
		}
		paintTreeChevron(tree.canvas, disclosure, item.Expanded, color)
		if !item.DisclosureID.IsZero() {
			tree.canvas.HitWithOptions(item.DisclosureID, disclosure, item.Label, ui.HitOptions{
				Role: ui.RoleButton, Disabled: item.Disabled, Expanded: item.Expanded, Expandable: true,
			})
		}
	}
	content := ui.Rect{X: contentX, Y: row.Y, Width: max(row.X+row.Width-contentX, float32(0)), Height: row.Height}
	if !item.ID.IsZero() {
		hit := row
		if item.Branch && !item.DisclosureID.IsZero() {
			hit.X = disclosure.X + disclosure.Width
			hit.Width = max(row.X+row.Width-hit.X, float32(0))
		}
		tree.canvas.HitWithOptions(item.ID, hit, item.Label, ui.HitOptions{
			Role: ui.RoleButton, Disabled: item.Disabled, Selected: item.Selected,
		})
	}
	return TreeRow{Bounds: row, Content: content, Disclosure: disclosure}
}

func paintTreeChevron(canvas *ui.Canvas, bounds ui.Rect, expanded bool, color ui.Color) {
	icon := &chevronRightIcon
	if expanded {
		icon = &chevronDownIcon
	}
	paintChevron(canvas, icon, bounds, color)
}

func treeExpand(bounds ui.Rect, amount float32) ui.Rect {
	return ui.Rect{X: bounds.X - amount, Y: bounds.Y - amount, Width: bounds.Width + amount*2, Height: bounds.Height + amount*2}
}

// The disclosure arrows of the tree and the accordion: Lucide's chevrons, drawn
// as stroked paths like every other icon, so they match the icons beside them
// at any size and scale.
var (
	chevronDownIcon  = ui.Icon{ViewBox: 24, Stroke: 2, Cap: ui.CapRound, Join: ui.JoinRound, Commands: []ui.PathCommand{ui.MoveTo(6, 9), ui.LineTo(12, 15), ui.LineTo(18, 9)}}
	chevronRightIcon = ui.Icon{ViewBox: 24, Stroke: 2, Cap: ui.CapRound, Join: ui.JoinRound, Commands: []ui.PathCommand{ui.MoveTo(9, 18), ui.LineTo(15, 12), ui.LineTo(9, 6)}}
	chevronUpIcon    = ui.Icon{ViewBox: 24, Stroke: 2, Cap: ui.CapRound, Join: ui.JoinRound, Commands: []ui.PathCommand{ui.MoveTo(18, 15), ui.LineTo(12, 9), ui.LineTo(6, 15)}}
)

// paintChevron draws a 16 point chevron centered in bounds.
func paintChevron(canvas *ui.Canvas, icon *ui.Icon, bounds ui.Rect, color ui.Color) {
	paintChevronSized(canvas, icon, bounds, 16, color)
}

// paintChevronSized draws a chevron of the given size centered in bounds. It is
// what the dropdown arrows of selects, comboboxes and menus use: a text glyph for
// the arrow sits low in its line box, so centering the line is not centering the
// arrow, while the icon's own geometry is centered exactly.
func paintChevronSized(canvas *ui.Canvas, icon *ui.Icon, bounds ui.Rect, size float32, color ui.Color) {
	canvas.DrawIcon(*icon, ui.Rect{X: bounds.X + (bounds.Width-size)/2, Y: bounds.Y + (bounds.Height-size)/2, Width: size, Height: size}, color)
}

// Package gallery owns the shared component showcase used by native and Web
// launchers. Keeping state and paint code here prevents backend-specific demos
// from drifting apart.
//
// The gallery is a browser for the component demos in package componentdemo:
// the catalog page lists every component, and a component page shows all of
// its examples side by side, each one live. Each example is painted on a canvas
// of its own and embedded into the page (see ui.Canvas.Embed), so the demos run
// exactly as they do in a standalone preview and the gallery owns no component
// recipes that could drift from them.
package gallery

import (
	"github.com/vivaui/viva/components"
	"github.com/vivaui/viva/internal/componentdemo"
	"github.com/vivaui/viva/text"
	"github.com/vivaui/viva/ui"
)

const (
	DefaultWidth  = 1180
	DefaultHeight = 880

	toolbarHeight = float32(56)
	pageInset     = float32(32)
	// pageTopGap and pageBottomGap pad the page inside its scroll viewport.
	pageTopGap    = float32(30)
	pageBottomGap = float32(20)
	// toolbarLayer lifts the toolbar above everything a page can paint: example
	// layers stay below 100. A clip culls a label only when none of it is
	// visible and does not crop one that is partly inside, so a line straddling
	// the top of the scroll area would otherwise be drawn in full over the
	// toolbar; the toolbar's opaque surface now covers it.
	toolbarLayer = 100
)

// The gallery names its places in a tree of its own. Its controls are on one
// page; the cards of the catalog are the items of one list; and each live
// example is mounted at a place of its own, so examples that reuse the same
// identities never collide and an event on one finds its way back to it.
//
// The tree is the demos' own (componentdemo.Tree): a demo names its controls in
// it, the gallery mounts the demo at its example's place, and the launchers give
// the same tree to their window.
var (
	tree      = componentdemo.Tree()
	galleryUI = tree.Feature("gallery")

	hitPagePrevious  = galleryUI.ID(tree.Name("page-previous"))
	hitPageNext      = galleryUI.ID(tree.Name("page-next"))
	hitPageCatalog   = galleryUI.ID(tree.Name("page-catalog"))
	hitPageScrollbar = galleryUI.ID(tree.Name("page-scrollbar"))
	hitThemeToggle   = galleryUI.ID(tree.Name("theme-toggle"))
	hitDensityCycle  = galleryUI.ID(tree.Name("density-cycle"))

	// cardScope's item n opens the nth component of the catalog.
	cardScope = tree.Sub(galleryUI, tree.Name("cards"))
	// examplesScope's item n is the place the nth live example is mounted at.
	examplesScope = tree.Sub(galleryUI, tree.Name("examples"))
)

func exampleMount(slot int) ui.Scope { return tree.Sub(examplesScope, ui.Index(slot)) }

// decodeExample finds the live example that owns an identity and the identity
// the example itself painted. ok is false for the gallery's own identities.
func (state *State) decodeExample(id ui.ID) (slot int, local ui.ID, ok bool) {
	for slot := range state.live {
		if local, ok := tree.Local(exampleMount(slot), id); ok {
			return slot, local, true
		}
	}
	return 0, ui.ID{}, false
}

// Tree returns the identity tree the gallery and its demos name their places in.
// A launcher sets it as its window's tree (app.Config.Tree) so identities print
// as paths in logs and in the headless driver's replies.
func Tree() *ui.Tree { return tree }

// State is the complete interaction state for the gallery. Both launchers use
// the same value so their pixels and behavior originate from one implementation.
type State struct {
	// page is 0 for the catalog and n for catalog[n-1].
	page        int
	lightMode   bool
	density     ui.Density
	textEditing bool

	// viewScroll is the page offset. The gallery owns it (wheel and scrollbar
	// drag) instead of the runtime's unbounded canvas offset, so it can never
	// pass the end of the page. scrollMax is the extent measured by the last
	// paint; thumbGrab and thumbDragging track one scrollbar drag.
	viewScroll    float32
	scrollMax     float32
	thumbGrab     float32
	thumbDragging bool

	// live holds the examples of the open component, in catalog order.
	live []liveExample
}

// liveExample is one running example: its demo state, the canvas it paints
// onto, and where the last paint placed it, which routes pointer coordinates
// back into its own space.
type liveExample struct {
	demo   *componentdemo.State
	canvas ui.Canvas
	origin ui.Point
	size   ui.Size
}

// New returns a gallery in its default desktop-light appearance.
func New() *State {
	return &State{lightMode: true}
}

// Theme returns the active appearance and density.
func (state *State) Theme() ui.Theme {
	theme := ui.DarkTheme()
	if state.lightMode {
		theme = ui.LightTheme()
	}
	theme.Metrics = theme.Metrics.WithDensity(state.density)
	return theme
}

// EnableTextEditing connects the open examples' text fields to the platform text
// service (native or browser): their text, selection and composition are edited
// through the editors TextEditor returns. Launchers that have no text service
// leave it off and every field stays a read-only snapshot.
func (state *State) EnableTextEditing() {
	state.textEditing = true
	for index := range state.live {
		state.live[index].demo.EnableTextEditing()
	}
}

// TextEditor returns the application-owned editor behind a text hit of an
// open example, if that example hands its field to the platform text service.
func (state *State) TextEditor(id ui.ID) (*text.Editor, bool) {
	slot, local, ok := state.decodeExample(id)
	if !ok || slot >= len(state.live) {
		return nil, false
	}
	return state.live[slot].demo.Editor(local)
}

// openPage shows the catalog (0) or a component from its top, with fresh
// example state. Revisiting the current page keeps its state and scroll.
func (state *State) openPage(page int) {
	page = min(max(page, 0), len(catalog))
	if page == state.page && (page == 0 || len(state.live) > 0) {
		return
	}
	state.page = page
	state.viewScroll = 0
	state.thumbDragging = false
	state.live = state.live[:0]
	if page == 0 {
		return
	}
	component := &catalog[page-1]
	// Fresh canvases every time: a canvas keeps its animation table across
	// frames, and a stale entry from another page would replay as a transition.
	state.live = make([]liveExample, len(component.Examples))
	for index, example := range component.Examples {
		demo, err := componentdemo.New(component.ID, example.ID)
		if err != nil {
			panic("gallery: catalog lists an example it cannot create: " + err.Error())
		}
		if state.textEditing {
			demo.EnableTextEditing()
		}
		state.live[index].demo = demo
	}
}

// Scroll applies one wheel or trackpad gesture. An example under the pointer
// that scrolls on its own (a scroll area, a message list) takes it first; the
// rest moves the page. The page offset is clamped to the extent the last paint
// measured, so momentum scrolling past the end cannot build up an invisible
// offset that makes the next gesture in the opposite direction appear to do
// nothing. Every gesture is handled: the gallery has no horizontal scrolling
// and owns the offset itself.
func (state *State) Scroll(event ui.ScrollEvent) ui.EventResult {
	if slot, local, ok := state.decodeExample(event.Target); ok && slot < len(state.live) {
		event.Target = local
		if result := state.live[slot].demo.Scroll(event); result.Handled {
			return liftResult(result, slot)
		}
	}
	before := state.viewScroll
	state.viewScroll = min(max(state.viewScroll+event.DeltaY, 0), state.scrollMax)
	return ui.EventResult{Handled: true, Repaint: state.viewScroll != before}
}

// Key hands a key to the example that owns the focused control.
func (state *State) Key(event ui.KeyEvent) ui.EventResult {
	slot, local, ok := state.decodeExample(event.Target)
	if !ok || slot >= len(state.live) {
		return ui.EventResult{}
	}
	event.Target = local
	return liftResult(state.live[slot].demo.Key(event), slot)
}

// Pointer drives the page scrollbar (pressing the thumb, or the track which
// centers the thumb on the pointer, and dragging moves the page) and routes
// the pointer to the example it is over, or that has captured it, in the
// example's own coordinates.
func (state *State) Pointer(event ui.PointerEvent) ui.EventResult {
	if slot, local, ok := state.decodeExample(event.Target); ok && slot < len(state.live) {
		view := &state.live[slot]
		event.X -= view.origin.X
		event.Y -= view.origin.Y
		event.Viewport = view.size
		event.Target = local
		return liftResult(view.demo.Pointer(event), slot)
	}
	switch event.Kind {
	case ui.PointerDown:
		if event.Target != hitPageScrollbar || (event.Button != ui.PointerButtonPrimary && event.Button != ui.PointerButtonNone) {
			return ui.EventResult{}
		}
		thumb := state.scrollGeometry(event.Viewport).VerticalThumb
		state.thumbGrab = thumb.Height / 2
		if event.Y >= thumb.Y && event.Y < thumb.Y+thumb.Height {
			state.thumbGrab = event.Y - thumb.Y
		}
		state.thumbDragging = true
		return state.dragThumb(event)
	case ui.PointerMove:
		if !state.thumbDragging {
			return ui.EventResult{}
		}
		return state.dragThumb(event)
	case ui.PointerUp, ui.PointerExit:
		if !state.thumbDragging {
			return ui.EventResult{}
		}
		state.thumbDragging = false
		return ui.EventResult{Handled: true}
	}
	return ui.EventResult{}
}

// liftResult maps the control an example asks to focus back to the identity it
// has on the page.
func liftResult(result ui.EventResult, slot int) ui.EventResult {
	result.Focus = tree.MountID(exampleMount(slot), result.Focus)
	return result
}

func (state *State) dragThumb(event ui.PointerEvent) ui.EventResult {
	offset := state.scrollGeometry(event.Viewport).VerticalOffsetForThumb(event.Y - state.thumbGrab)
	changed := offset != state.viewScroll
	state.viewScroll = offset
	return ui.EventResult{Handled: true, Repaint: changed}
}

// scrollArea declares the page scroll region: everything under the toolbar, at
// the full window width so the scrollbar sits on the window edge while the
// page keeps its side insets.
func (state *State) scrollArea(canvas *ui.Canvas, theme ui.Theme, size ui.Size) components.ScrollArea {
	viewport := ui.Rect{Y: toolbarHeight, Width: size.Width, Height: max(size.Height-toolbarHeight, 0)}
	pageHeight := state.contentHeight(max(size.Width-pageInset*2, 0))
	return components.BeginScrollArea(canvas, theme, viewport, ui.Size{Width: viewport.Width, Height: pageTopGap + pageHeight + pageBottomGap}).
		Offset(0, state.viewScroll).
		VerticalScrollbar(hitPageScrollbar, "Page position", "")
}

func (state *State) scrollGeometry(size ui.Size) components.ScrollAreaGeometry {
	return state.scrollArea(nil, ui.Theme{}, size).Geometry()
}

// contentHeight is the height of the open page at a page width.
func (state *State) contentHeight(width float32) float32 {
	if state.page == 0 {
		return catalogHeight(width)
	}
	return componentHeight(width, len(state.live))
}

// Paint draws the same responsive gallery on every backend.
func (state *State) Paint(canvas *ui.Canvas, size ui.Size) {
	canvas.SetTree(tree)
	theme := state.Theme()
	width, height := size.Width, size.Height
	canvas.Fill(ui.Rect{Width: width, Height: height}, theme.Background)
	canvas.PushLayer(toolbarLayer)
	page := state.paintToolbar(canvas, theme, width)
	canvas.PopLayer()
	// A page change takes effect here, in the same pass, so the rest of the
	// paint draws the new page rather than a half-changed one.
	state.openPage(page)

	area := state.scrollArea(canvas, theme, size)
	geometry := area.BeginContent()
	// Write the clamped offset back: a resize or a shorter page may have left
	// the stored one past the new end.
	state.viewScroll, state.scrollMax = geometry.OffsetY, geometry.MaxOffsetY
	pageWidth := max(width-pageInset*2, 0)
	content := ui.Rect{
		X: pageInset, Y: geometry.Content.Y + pageTopGap,
		Width: pageWidth, Height: state.contentHeight(pageWidth),
	}
	if state.page == 0 {
		state.paintCatalog(canvas, theme, content, geometry.Viewport)
	} else {
		state.paintComponent(canvas, theme, content, geometry.Viewport)
	}
	area.EndContent()
}

// paintToolbar paints the toolbar and returns the page to show after it: the
// open one, unless a click in this paint chose another.
func (state *State) paintToolbar(canvas *ui.Canvas, theme ui.Theme, width float32) (page int) {
	page = state.page
	canvas.Fill(ui.Rect{Width: width, Height: toolbarHeight}, theme.Card)
	canvas.Fill(ui.Rect{Y: toolbarHeight - 1, Width: width, Height: 1}, theme.Border)

	// Every element is centered on the bar's own centerline (the area above
	// its 1px border) at the active density's control height, rather than at
	// hand-picked offsets that only line up at one density. Buttons use the
	// Small size, whose height is exactly the control height; the default
	// button is four pixels taller and would sit below the centerline.
	control := theme.Metrics.ControlHeight
	top := (toolbarHeight - 1 - control) / 2
	markSize := float32(20)
	paintMark(canvas, theme, ui.Rect{X: pageInset, Y: (toolbarHeight - 1 - markSize) / 2, Width: markSize, Height: markSize})

	// Clicks are read where the control is painted, in the event pass of the
	// click (see ui.Canvas.Clicked).
	if paintIconButton(canvas, theme, hitPagePrevious, ui.Rect{X: 72, Y: top, Width: 28, Height: control}, iconChevronLeft, "Previous page") {
		page = (state.page + len(catalog)) % (len(catalog) + 1)
	}
	if paintIconButton(canvas, theme, hitPageNext, ui.Rect{X: 104, Y: top, Width: 28, Height: control}, iconChevronRight, "Next page") {
		page = (state.page + 1) % (len(catalog) + 1)
	}
	labelX := float32(148)
	title := "Components"
	if state.page != 0 {
		if components.BeginButton(canvas, theme, hitPageCatalog, ui.Rect{X: labelX, Y: top, Width: 96}).Small().Ghost("Components").Clicked() {
			page = 0
		}
		labelX += 108
		title = catalog[state.page-1].Title
	}
	canvas.Label(title, ui.Rect{
		X: labelX, Y: top, Width: width - labelX - 210, Height: control,
	}, ui.TextStyle{Size: theme.Metrics.FontSizeM, Weight: ui.WeightMedium, Color: theme.MutedForeground})

	densityBounds := ui.Rect{X: width - 190, Y: top, Width: 88}
	if components.BeginButton(canvas, theme, hitDensityCycle, densityBounds).Small().Ghost(state.densityName()).Clicked() {
		state.density = (state.density + 1) % 3
	}
	appearance := "Light"
	if state.lightMode {
		appearance = "Dark"
	}
	if components.BeginButton(canvas, theme, hitThemeToggle, ui.Rect{
		X: width - 92, Y: top, Width: 76,
	}).Small().Outline(appearance).Clicked() {
		state.lightMode = !state.lightMode
	}
	return page
}

func (state *State) densityName() string {
	switch state.density {
	case ui.DensityCompact:
		return "Compact"
	case ui.DensityLarge:
		return "Large"
	default:
		return "Regular"
	}
}

// pageHeaderHeight is the height pageHeader takes, including the gap below.
const pageHeaderHeight = float32(82)

// pageHeader paints a page title and subtitle and returns the body below them.
func pageHeader(canvas *ui.Canvas, theme ui.Theme, content ui.Rect, title, subtitle string) ui.Rect {
	metrics := theme.Metrics
	canvas.Label(title, ui.Rect{X: content.X, Y: content.Y, Width: content.Width, Height: 38}, ui.TextStyle{
		Size: metrics.FontSizeL, Weight: ui.WeightSemibold, Color: theme.Foreground,
	})
	canvas.Label(subtitle, ui.Rect{X: content.X, Y: content.Y + 38, Width: content.Width, Height: 22}, ui.TextStyle{
		Size: metrics.FontSizeM, Color: theme.MutedForeground,
	})
	return ui.Rect{X: content.X, Y: content.Y + pageHeaderHeight, Width: content.Width, Height: max(content.Height-pageHeaderHeight, 0)}
}

// wrapLine splits value at a rune boundary so the first part fits width at
// size, using a rough width estimate (0.54 of the size, CJK one em), and returns the
// rest. Text is single-line on every backend, so longer copy is split by hand.
// A line never ends before closing punctuation: the rune before it moves down
// with it, so a lone "。" cannot start a line.
func wrapLine(value string, size, width float32) (line, rest string) {
	advance := float32(0)
	previous := 0
	for index, r := range value {
		glyph := size * 0.54
		if r > 0x2E7F {
			glyph = size // full-width CJK
		}
		if advance+glyph > width {
			if index > 0 && isClosingPunctuation(r) && previous < index {
				index = previous
			}
			return value[:index], value[index:]
		}
		advance += glyph
		previous = index
	}
	return value, ""
}

func isClosingPunctuation(r rune) bool {
	switch r {
	case '。', '，', '、', '；', '：', '！', '？', '）', '」', '』', '》', '”', '’', '.', ',', ';', ':', '!', '?', ')':
		return true
	}
	return false
}

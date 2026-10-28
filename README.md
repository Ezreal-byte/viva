# Viva

Viva is a Go-first, GPU-native desktop framework. Applications draw through
the platform's own window and GPU APIs — Cocoa and Metal, Win32 and Direct3D 11,
XCB and Vulkan — with no WebView, no browser engine, and no bundled runtime. It
is aimed at small binaries, low idle resource usage, and direct access to native
platform capabilities.

Viva is pre-1.0: the API can still change between releases, and macOS is the
reference platform. The Windows and Linux backends track it and are the least
exercised. The status list below is what runs today.

## Quick start

Viva is a Go module, `github.com/vivaui/viva`. The `viva` command builds,
packages and live-reloads applications; nothing in it is linked into the program
you ship.

```sh
brew install vivaui/tap/viva        # macOS, Linux
scoop bucket add vivaui https://github.com/vivaui/homebrew-tap && scoop install viva   # Windows
viva doctor                         # is this machine ready?
viva init hello
cd hello
viva dev .        # hot reload: save a file, the window updates and keeps its state
```

Prebuilt archives (with `SHA256SUMS`) are on each release from `v0.1.3` on; with Go
already installed, `go install github.com/vivaui/viva/cmd/viva@latest` works too.
You do not need Go first: `viva build`, `viva dev` and `viva init` download the
official toolchain once into your user cache when there is none (checked against
go.dev's SHA-256; `VIVA_GO_DOWNLOAD=0` turns it off). A project made by `viva init`
pins the command with a `tool` line in `go.mod`, so `go tool viva` runs the version
that project builds against. See the Build and package guide.

`viva init` writes a runnable project: a window, a test that drives it without
opening one, `viva.toml`, and an `AGENTS.md` describing the programming model.
`viva build` writes a release program to `build/`, and `viva package` writes the
platform's own bundle.

Requirements: Go 1.26 or newer (viva fetches it when missing). On macOS the Xcode command-line tools, because
Viva reaches Cocoa and Metal through cgo; on Linux, X11 with the system XCB and
Vulkan loaders; on Windows, Win32 with Direct3D 11 and DirectWrite. Viva loads
the system libraries at runtime and bundles none of them.

The current release is `v0.1.2`; name it explicitly (`@v0.1.2`) to pin it. Guides,
the API reference, and a component catalog with live previews live in
[`website/`](website/); run it with `npm run dev` there.

Everything below describes this repository: its measurements, its own programs,
and how to work on the framework rather than with it.

## Scope

Viva targets native desktop applications on macOS, Windows, and Linux. A
WebAssembly/WebGPU backend is optional and must not add cost to native builds.
iOS, iPadOS, Android, mobile packaging, and a touch-first mobile runtime are
explicitly out of scope. Desktop performance and native integration will not be
weakened to preserve hypothetical future mobile compatibility.

All three backends use direct operating-system window and GPU APIs with no
WebView and no bundled rendering runtime.

## Current Status

- Native macOS window and event loop.
- Direct Metal clear and present.
- Flat generational retained scene with stable node identities.
- Coalesced O(1) localized rectangle updates with zero hot-path allocations.
- Precompiled Metal shader and instanced rectangle rendering.
- Event-driven C-to-Go pointer dispatch and 32-byte partial Metal uploads.
- Paint-order hover and press interaction state with click activation;
  moving the pointer inside one region performs no renderer work.
- Keyboard focus with Tab traversal, Enter and Space activation, and focus
  rings; one key event repaints at most once and publishes accessibility
  updates only when focus actually changed.
- Event-driven scrolling with per-primitive clipping: a 1000-row list
  virtualizes to its visible window, one scroll gesture repaints exactly once
  with one partial GPU upload, and clipped content never reaches the
  renderer.
- Three native window modes: standard titled, frameless (no traffic lights),
  and custom titlebar (transparent title bar, full-height content, the window
  moves from the regions the page declares). Window close can be intercepted
  by the application.
- Explicit multi-window applications through `app.RunApplication`: every
  window owns independent retained paint, focus, input, text, invalidation, and
  close state, and the native loop exits after the final window closes.
- Status-area icons and popup panels: `Application.AddStatusItem` puts an icon
  in the macOS menu bar, the Windows notification area, or a Linux
  StatusNotifier host; `StatusItemConfig.Menu` makes it open the system's own
  menu (NSMenu, a Win32 popup menu, `com.canonical.dbusmenu`); `WindowPanel` is
  the borderless popup that opens beside it and closes when it loses focus;
  `Window.Hide`/`Show` keep an application running without a visible window.
- `app.StoreSecret`/`LoadSecret`/`DeleteSecret` keep a small secret in the
  system's own store (Keychain, Credential Manager, Secret Service).
- `ui.MonoFamily` draws fixed-pitch text in any installed family and
  `app.MonospaceFamilies` lists them.
- `Config.OnFileDrop` receives files dragged from the file manager, with the hit
  under the pointer.
- A separate retained semantic accessibility tree mirrored into native
  NSAccessibility elements, Windows UI Automation providers (Invoke, Toggle,
  Value, RangeValue, SelectionItem, ExpandCollapse and Text patterns, with
  events) and Linux AT-SPI objects, and into ARIA in the browser build. A
  screen reader's press, focus and set-value requests reach the application
  through `platform.AccessibilityHandler`. See
  [docs/accessibility.md](docs/accessibility.md).
- Event-driven window-resize repainting with exactly one GPU submission per
  resize and zero work while the size is unchanged.
- `Config.OnFrameChange`, `Config.OnMinimize` and `Window.Minimized` report
  where a window is and whether the window manager minimized it.
- Retained Row/Column/Stack/Flex layout with O(1) unchanged-layout checks.
- Paint-order pointer routing, stable keyboard focus, and Tab traversal.
- Native macOS `NSTextInputClient` integration with real Chinese IME candidate
  composition, grapheme-aware editing, and UTF-16 mapping.
- Focus-driven dynamic native text editors on macOS, Windows, and Linux, with
  explicit enable/replace/disable frame updates. macOS and Windows caret timers
  exist only while an editable field is focused. Linux composes through the IBus
  protocol (IBus and Fcitx5's IBus frontend); XIM is still open.
- A separate retained semantic accessibility tree.
- Direct CoreText shaping, system-font fallback, GPU-composited text textures,
  and a bounded measurement cache.
- Experimental Windows shell with Win32 events, D3D11 retained rendering,
  embedded DXBC shaders, DirectWrite measurement, and bounded GDI label
  textures; native builds do not load a shader compiler.
- Experimental Linux/X11 shell with dynamically loaded system XCB, Vulkan,
  Pango, and Cairo; retained rounded rectangles and triangles, layered
  GPU-composited system text, pointer and keyboard input, custom titlebar drag,
  and a blocking event loop with no idle redraw.
- Event-driven rendering with no idle frame loop.
- Theme tokens and a statically typed, source-owned translation of the pinned
  shadcn component catalog, including forms, overlays, menus, calendars,
  scrolling recipes, feedback surfaces, and area, line and bar charts drawn
  from typed slices without a browser runtime, reflection, or a runtime style
  parser.
- A shared native/WASM component browser: a catalog of every component grouped
  by category, and a page per component that runs all of its examples side by
  side, each live (clicks, hover, keyboard, drag, wheel and native text
  editing), with a draggable page scrollbar and hot-reload state preservation.
  The examples are the ones in `internal/componentdemo`, embedded into the page
  with `ui.Canvas.Embed`, so the gallery holds no copy of any component.
- Clean event-loop return on window close or application termination.
- Reproducible binary, startup, CPU, memory, and GPU-submission measurement.

The Windows backend now cross-builds for amd64 and arm64 and passes its platform
tests plus Gallery startup, pointer, and keyboard smoke tests under Wine/DXVK.
That compatibility run does not replace validation on real Windows hardware;
Windows IME, startup, RSS, GPU behavior, and text raster quality remain open.
Platform tests, the conformance scenarios and the new input, drop and device-loss
paths run on a Windows 11 virtual machine (WARP) and a Debian KDE/X11 machine
(lavapipe); `docs/conformance.md` lists what neither covers.
UI Automation is checked against a real Windows 11 machine from an external
client (see docs/accessibility.md). The Linux backend passes Ubuntu arm64 Xvfb/llvmpipe M0 and Debian
amd64 X11/Lavapipe retained-scene smokes, including mixed Chinese/Latin text
and localized scene updates. XIM, native Wayland, smooth (XInput2) scrolling and
hardware-GPU measurements remain open; AT-SPI is checked against libatspi on a
KDE/X11 machine.
Native label renderers are intentionally single-line M2 implementations;
glyph-atlas optimization remains future work. The optional Web preview currently
uses Canvas2D rather than the planned WebGPU backend.

## Run the repository

These commands build and drive the framework's own programs from this checkout;
requirements are the ones in Quick start.

Run the native component showcase:

```sh
go run ./cmd/showcase
```

Run the component gallery in the browser (Canvas2D WebAssembly backend;
same component and paint code as the native build):

```sh
tools/build_web.sh   # viva build --target web, then serves http://127.0.0.1:8090
```

The gallery defaults to light appearance and offers dark mode and three control
densities; both reach every embedded example. Both targets paint the same Go
components. Mouse clicks preserve keyboard focus without showing a focus ring;
Tab navigation exposes the ring. Hover and press change color, never the
control bounds. The catalog grid and the example stages follow the window width.
On macOS, scrolling follows the system's natural-scrolling setting.

`Canvas.Focused()` exposes the real focus owner for both pointer and keyboard
input. `Canvas.FocusVisible()` exposes that ID only when a keyboard focus ring
should be painted. The runtime sets both in one
`SetInteraction(hovered, pressed, focused, focusVisible)` call; paint code reads
them rather than retaining the last clicked control as its own focus state.
A text field in an example hands text painting to the native editor only while
it owns focus, and restores the current Go-owned value as a static label in the
same frame that disables editing on blur. On macOS it supports click-to-place,
drag selection, and Command-A, with rune-indexed `TextSetSelection` events sent
back to the application. Input, Field, Form, Input group (single and multi-line),
Textarea and Input OTP take keyboard input this way; the other text-entry
examples are fixed snapshots of a state (selection, composition, error,
disabled). `text.Editor.Restrict` limits what an editor accepts, which is how
the one-time code takes only digits, six at most. The browser gallery does not
connect a text service yet.

The visual direction and its measured costs are recorded in
`docs/benchmarks/m3-gallery-style-darwin-arm64.md`.

### Website: documentation, API reference and component catalog

The isolated Astro + Svelte site in `website/` is the project's public site, in
Chinese (default) and English:

- `/zh/` and `/en/`: the home page, with a live Go/WASM component and the
  project's measured numbers, each quoted from a benchmark report and checked
  against it by `npm test`, including the two it misses.
- `/<lang>/docs/`: guides for people building applications (getting started,
  the programming model, windows, input, text, drawing, theming, overlays,
  components, system integration, packaging, testing, platforms, performance).
  Their Go code is not copied by hand: every code block is a file under
  `examples/guides/`, which `go vet` and `go test ./...` build and drive
  headlessly, and the website tests fail if a guide differs from its file
  (`npm run sync:examples` refreshes them).
- `/<lang>/api/`: the API reference for `app`, `ui`, `components`, `text` and
  `headless`, generated from the Go source and its doc comments by
  `tools/docs`, so it cannot describe an identifier that does not exist.
- `/<lang>/components/`: the component index, grouped by category.
- `/<lang>/ui/<component>/`: one page per source-owned component recipe, with
  real Go/WASM previews, usage snippets, a source link, keyboard notes and
  platform limits. The docs generator fails when a new recipe is added without
  a matching preview page.

Documentation controls use shadcn-svelte and system fonts; they do not
recreate Go components in HTML/CSS.

With Node.js 22.12 or newer and npm installed, run:

```sh
cd website
npm ci
npm run dev
# http://127.0.0.1:4321/zh/
# http://127.0.0.1:4321/en/
```

Astro handles the documentation; Svelte handles the interactive documentation
controls. Previews start as their examples approach the viewport; changing to the
source tab preserves preview state. Actual Go code and demo labels are unchanged
by documentation language. Editing Go component/example sources triggers a serialized WASM and
source-data rebuild, followed by a page reload. Failed builds retain the last
successful preview and report the error. Resetting a preview only resets its
state; it does not compile code. The production site is a static build:

```sh
cd website
npm run build
# Static output: ../build/docs-site/
npm run preview
npm test && npm run test:site
```

From the repository root, validate the catalog and API data or run an example
natively:

```sh
go run ./tools/docs -check
go run ./cmd/component-preview -component button -example variants
go run ./cmd/component-preview -component input -example default
go run ./examples/guides/hello
```

The standalone native Input example connects the real text editor; the browser
Input intentionally shows visual states only. Native Canvas examples support
macOS plus the experimental Windows and Linux backends; Linux composes through
IBus, and XIM remains unsupported.
They do not run through `viva dev`, whose worker protocol does not yet carry
native text events. The source fingerprint describes Go build inputs, not a
published version. Deployed static sites require rebuilding and redeploying
after source edits. Website packages never enter native release builds.

List and install source-owned component recipes:

```sh
go run ./cmd/viva list
go run ./cmd/viva add --out ./components/ui button card badge
```

Every file of `components/` is a recipe. The installer copies ordinary Go files
into the destination package, together with the recipes the one you named calls
into, so each installed set compiles on its own. The installable copies are
generated from `components/` (`go generate ./cmd/viva`), so they match it. It
does not add a runtime registry, stylesheet parser, background service, or
component bundle to the release application, and importing `github.com/vivaui/viva/components`
without calling a recipe links no symbol of it.

### Building and packaging an application

`viva build` makes a release program (`-trimpath`, stripped, a fixed build ID so
the same sources give the same bytes) for `--target darwin`, `windows`, `linux`
or `web`, and reports its size separately from anything around it. `viva package`
wraps the macOS program in a `.app` bundle. Both read an optional `viva.toml`
next to `go.mod`; this repository's own is the Gallery's:

```toml
[app]
name = "Viva Gallery"          # shown in the menu bar; names the .app
id = "dev.viva.gallery"        # bundle identifier
version = "0.1.0"
executable = "viva-gallery"    # default: the name, lowercased
package = "./cmd/showcase"    # the main package to build
icon = "assets/icon.png"      # PNG, square, 128 px or more (1024 recommended), or .icns

[web]
package = "./cmd/gallery-web" # the browser entry point has its own main package

[build]
tags = []                     # build tags for every target
out = "build"

[mac]
min_system = "12.0"           # the macOS the program is compiled for; default and floor 12.0
category = "public.app-category.developer-tools"

[mac.info]                    # extra Info.plist strings, such as usage descriptions
NSMicrophoneUsageDescription = "Records the demo."
```

```sh
go run ./cmd/viva build                    # build/viva-gallery
go run ./cmd/viva build --target web       # build/web/: program, loader, index.html
go run ./cmd/viva build --arch amd64       # x86_64 macOS build of the same cgo/Metal program
go run ./cmd/viva build --target windows --arch amd64
go run ./cmd/viva package --zip            # build/Viva Gallery.app and a reproducible .zip
go run ./cmd/viva help package             # every command and flag is documented
```

`viva.toml` is decoded strictly: an unknown table or key, a duplicate or a value
of the wrong type is an error that names the line. macOS programs are compiled
for `min_system` rather than for the macOS of the machine building them (the
default toolchain target would be that machine's own, which refuses to start
anywhere older), and `package` refuses a bundle that claims an older system than
its program needs. Both Darwin architectures are supported from either host: the
build keeps cgo and the Objective-C/Metal bridge enabled and lets clang produce
the other Mach-O slice. The macOS runtime tests and benchmarks in this repository
were run on arm64; Intel GPU, IME and text-raster behavior is not part of that
matrix yet.

Signing and notarization are optional: with no identity the
bundle is unsigned, runs on the Mac that built it, and other Macs block it. Give
`--sign-identity` or `APPLE_SIGNING_IDENTITY` to sign it (hardened runtime, secure
timestamp, `[mac] entitlements`), and notarytool credentials to notarize and
staple it as well; `--no-sign` turns it all off. `--target windows`
makes an NSIS installer or a zip, signed with Authenticode when a certificate or
a sign command is configured (`WINDOWS_CERTIFICATE_FILE` and friends), and
`--target linux` a tar.gz or a .deb, which are not signed (`viva help package`).
Two Go module dependencies are kept out of the native application path unless
their feature is used. `go-toml/v2` is used only by the `viva` command, so a
release program does not link it. `purego` is used by the Linux backend (and the
FFI-cost tooling); macOS and Windows programs do not link it. The release
programs also leave the development reload protocol, single-instance sockets,
the Go test runtime that leak counters used to pull in, and PNG icon decoding out
of the link. Reload, single-instance and PNG decoding are opt-in packages; the
test runtime is simply absent. The leak counters themselves remain always on
because they are only two atomics per window or driver.

A static canvas can be presented through the public API:

```go
theme := ui.DarkTheme()
var canvas ui.Canvas
components.Card(&canvas, theme, ui.Rect{X: 24, Y: 24, Width: 320, Height: 180})
if err := app.RunCanvas(app.Config{Title: "My App"}, &canvas); err != nil {
    log.Fatal(err)
}
```

A resizable canvas repaints on window-size changes by setting `Paint`. The
runtime reconciles each immediate paint with the retained scene, so unchanged
repaints perform no renderer work, localized changes upload one contiguous
instance range, and text label textures are rebuilt only when a label actually
changed:

```go
config := app.Config{
    Title: "My App",
    Paint: func(canvas *ui.Canvas, size ui.Size) {
        components.Card(&canvas, theme, ui.Rect{
            X: 24, Y: 24,
            Width: size.Width - 48, Height: size.Height - 48,
        })
    },
}
if err := app.RunCanvas(config, &canvas); err != nil {
    log.Fatal(err)
}
```

A page of controls is declared, not placed by coordinates: package `flex` lays
out rows, columns, grids, scroll areas, popups and the common controls from a
declaration that runs once a frame (see `flex/doc.go` and
`examples/guides/layout`). `ui.Row`, `ui.Column`, `ui.Grid` and `Rect.Cut*` stay
for custom painting that is laid out by hand.

A control is declared where the page is laid out, and its click is handled
there: `f.Button` takes the identity, paints hover, press and focus from the
canvas, and `Clicked()` is true in the one paint that delivers its click, so the
state it changes sits next to the button:

```go
var (
    tree = ui.Tree{}
    page = tree.Feature("editor")
    save = tree.ID(page, "save")
)

config.Paint = func(canvas *ui.Canvas, size ui.Size) {
    bounds := ui.Rect{Width: size.Width, Height: size.Height}
    view.Draw(canvas, theme, bounds, func(f *flex.Frame) {
        f.Column().Fill().Pad(24).Children(func() {
            if f.Button(save, "Save").Clicked() {
                // save()
            }
        })
    })
}
```

Controls that need drag, secondary-button, keyboard, or targeted scrolling use
the typed event callbacks. State remains application-owned, and returning an
empty result from an unchanged event performs no repaint:

```go
config.OnPointer = func(event ui.PointerEvent) ui.EventResult {
    if event.Target == resizeHandle && event.Kind == ui.PointerMove && event.Captured {
        panelWidth = event.X - panelLeft
        return ui.EventResult{Handled: true, Repaint: true}
    }
    if event.Target == menuTarget && event.Kind == ui.PointerDown && event.Button == ui.PointerButtonSecondary {
        menuOpen = true
        return ui.EventResult{Handled: true, Repaint: true, Focus: menuTarget, FocusVisible: true}
    }
    return ui.EventResult{}
}
config.OnKey = func(event ui.KeyEvent) ui.EventResult {
    if event.Target == tabsTarget {
        next, handled := components.TabsKeyDown(selectedTab, tabs, components.TabsHorizontal, components.DirectionLTR, event.Key)
        if handled {
            selectedTab = next
            return ui.EventResult{Handled: true, Repaint: true, Focus: next, FocusVisible: true}
        }
    }
    return ui.EventResult{}
}
config.OnShortcut = func(event ui.KeyEvent) ui.EventResult {
    if event.Modifiers == primaryModifier|ui.ModifierShift && event.Text == "p" {
        commandPaletteOpen = true
        return ui.EventResult{Handled: true, Repaint: true}
    }
    return ui.EventResult{}
}
config.OnScroll = func(event ui.ScrollEvent) ui.EventResult {
    if event.Target == listTarget {
        listOffset = max(0, listOffset+event.DeltaY)
        return ui.EventResult{Handled: true, Repaint: true}
    }
    return ui.EventResult{}
}
```

Pointer presses retain their original target through captured move/up events.
Scroll events carry their window position, modifiers, viewport, and target, so
coordinate-driven controls do not depend on a preceding pointer move.
`OnShortcut` receives Control/Command-modified printable chords before native
text editing. Returning an unhandled result preserves the platform text and IME
path; applications should handle only bindings they own.
`Handled` suppresses the click a press would complete, Enter/Space activation, Escape dismissal,
or default window scrolling. `Focus`, `Blur`, and `FocusVisible` update the
shared input router. Native, WebAssembly, and `viva dev` reload workers transport
the same event values, including pointer and scroll positions, modifiers, key
repeat, both scroll axes, and primary/secondary/middle buttons.

Run the input, text and scene measurement harness:

```sh
./benchmarks/m2_core.sh
```

## Engineering Rules

The performance budgets, dependency policy, architecture boundaries, and
definition of done are enforced by the checked-in benchmarks and tests.
Measurements are evidence; terms such as
“native speed,” “zero idle work,” and “lightweight” are not used as unverified
marketing claims.

## Contributing

Bug reports, platform findings, and documentation fixes are always welcome — see
[CONTRIBUTING.md](CONTRIBUTING.md) for the gate a change has to pass and for
where the line between framework and application behaviour sits. Security
problems go through [SECURITY.md](SECURITY.md), never a public issue.

## Contributors

<a href="https://github.com/vivaui/viva/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=vivaui/viva&max=300&columns=15" alt="Contributors" />
</a>

## License

Viva is licensed under the Apache License 2.0. See [LICENSE](LICENSE) and
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

# Component Gallery Style and Pointer Parity

Measured 2026-10-13 on Apple M2 Pro, Mac14,9, 16 GiB RAM, macOS 15.5
(24F74), darwin/arm64, Go 1.26.5. These are local observations, not budget
compliance claims.

## Direction

The gallery uses neutral white/graphite surfaces, cobalt action and selection
tones, compact geometry, a source-painted geometric mark, and source previews.
It takes clarity and control proportions from desktop interfaces rather than
copying macOS System Settings. A desktop list demonstration informed the clear
list selection treatment. No assets, fonts, dependencies, timers, or workers
were added. Native rendering remains Cocoa/CoreText/Metal; Web remains an
isolated Canvas2D/WASM preview, not WebGPU.

## Pointer and Keyboard Invariants

The old native painter assigned focus only to its own field on pointer down.
It painted an expanded ring before synchronizing with an unfocused input
router, which cleared that focus. Pointer up removed the ring again. The Web
painter did not set pointer focus, so it did not show the same one-frame edge
expansion.

Both painters now focus through the shared input router and track keyboard
focus visibility separately. Pointer down hides the keyboard ring without
moving the control. Tab reveals the ring and advances from the clicked
control. Semantic focus still follows the native router. Mouse press/release
changes colors at fixed bounds. Tests cover native and WASM painters.

Canvas hit regions intersect active clips, preventing scrolled-away controls
from responding over the toolbar. Gallery layout, theme, state, and recipes
are shared; rasterization and system fonts are still platform-dependent.

## Reproduce

```sh
go test ./...
GOOS=js GOARCH=wasm go test \
  -exec="$(go env GOROOT)/lib/wasm/go_js_wasm_exec" \
  ./ui ./components ./internal/gallery ./internal/platform/web
go test ./internal/gallery -run '^$' -bench BenchmarkGalleryPaint -benchmem -count=3
go test ./app -run '^$' -bench BenchmarkPainterSteadyStateRepaint -benchmem -count=3
./benchmarks/m3_showcase_darwin.sh
```

Release builds use `-trimpath -ldflags='-s -w -buildid='` for native and WASM.
Executable bytes exclude the temporary preview app bundle and screenshots.

| Observation | Before | After |
| --- | ---: | ---: |
| Native stripped executable | 1,956,306 B | 1,972,834 B |
| Stripped WASM | 2,772,175 B | 2,797,732 B |
| Idle RSS (`ps`, after 2–3 seconds) | 84,768 KiB | 81,312 KiB |
| Idle CPU (`ps`, single snapshot) | 0.1% | 0.0% |
| Pointer press/release full rebuilds, including startup | 3 | 1 |
| Pointer press/release partial updates | 0 | 2 |

The size delta includes added previews, icons, and responsive gallery behavior,
not just theme colors. RSS/CPU are single samples and not a statistically
validated improvement. `State` grows by two cached strings, a selected-row
integer, and alignment padding; cached captions are rebuilt only on explicit
clicks. No per-widget asynchronous state is introduced.

After warm-up, three default overview paint runs measured 1518–1525 ns/op,
0 B/op, 0 allocs/op. The unchanged runtime repaint benchmark measured
92.30–92.35 ns/op, 0 B/op, 0 allocs/op. The original overview had no paint
benchmark; no timing comparison to that original is claimed.

The focused pointer regression benchmark (one button, warmed storage, one
press/release pair per operation) measured 388.1–399.5 ns/op, 0 B/op,
0 allocs/op with `go test ./app -run '^$' -bench
BenchmarkPainterButtonPointer -benchmem -count=3`.

The native harness reports one initial GPU submission, zero further idle
submissions, and no idle text updates. An isolated click run reports three
submissions total, one full rebuild, two partial updates, and one changed
label texture. The scripted resize crosses the narrow-layout breakpoint and
therefore intentionally rebuilds structure once.

## Limits

The observed native first frame remains above the provisional 50 ms budget
(174–376 ms in these launches). Idle RSS remains above 15 MiB. Per-label
CoreText textures and GPU resources remain existing costs, and this visual
work does not solve those budgets. No goroutine count, installer size, or
cross-platform pixel equivalence was measured. Windows/Linux appearance has
not been validated.

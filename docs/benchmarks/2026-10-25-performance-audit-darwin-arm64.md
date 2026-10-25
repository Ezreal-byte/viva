# Performance audit: where the next gains are

Measured 2026-10-25 on Apple M2 Pro (Mac14,9), 16 GiB RAM, macOS 15.5
(24F74), darwin/arm64, Go 1.26.5 with CGO enabled, Go source at `c883da0`
(unchanged by this audit). Every number below is a local observation from one sitting, taken
with the checkout's own benchmarks, not a budget claim. `-count=1` runs vary by
roughly 5–15% on this machine; treat differences smaller than that as noise.
What has not been measured is listed at the end.

This document answers one question: **where does Viva still spend time that it
does not have to?** It is an audit, not a change. Where an experiment required
editing code, the edit was reverted before this document was written, so the
tree still builds exactly as released; the measured deltas are upper bounds for
the proposed optimization, not landed results. §10 records what was implemented
afterwards and re-measures it against a worktree of the same commit.

---

## Conclusion: the five things worth doing first

| # | Opportunity | Evidence (measured) | Expected effect | Effort | Risk |
| - | --- | --- | --- | --- | --- |
| 1 | Skip the per-paint hit bookkeeping (labels, semantics, accessibility, regions) when the paint declared the same hits | A 4,000-control page repaints in 747.8 µs; with hit copies localized (a safe first step) and the unchanged semantics + regions work removed, 178 µs (−72%) | 2–3× faster repaint for pages with hundreds to thousands of controls; no gain for small pages | M | Medium: change detection must not miss a focus, label or value change |
| 2 | Let the renderer insert/remove an instance in the middle of a page without a full rebuild | Hover highlight insert: 107 µs vs 71 µs for the same page with a color-only hover; profile puts 27–37% in `render.Store.rebuild` | ~1.5× on hover/highlight repaints of dense pages, and no full instance re-upload for a one-shape change | M | Medium: ordering and `instanceBySlot` invariants |
| 3 | Stop re-tessellating bordered overlays during a fade | Modal open/close: 40 µs; 41% of samples in `earcut.isEar` and 65% cumulative under `Presence.apply` (the fade vectorizes every `RoundedBorder` each frame) | ~1.5–2× on animated dialogs/popovers; the 8.4 µs fade-vectorize benchmark would hit a cache | S–M | Low: cache keyed by geometry |
| 4 | Stop converting a text line to a string before the bidi early-out | Arrow-key cursor move: 1,149 ns, 8 allocs, 384 B on mixed text; the allocation is the whole logical line, for a line that is not bidi | 0 allocs per cursor move; less GC during editing | S | Low |
| 5 | Look up grapheme properties once per rune, not three times | Mixed CJK/Latin wrap: 84.5 µs; 38% of samples in `uniprops` property lookups (`propOf`, `tuple`, `flags`), 72% in `grapheme.Boundary` | ~2× on wrapping/segmentation of non-ASCII text (Latin also gains from an ASCII fast path) | S–M | Low: table generation + differential tests |

The rest of this document gives the measurements, the phase attribution, and the
secondary items (hit testing, memory duplication, missing measurements).

---

## Method

Baseline (`-benchmem`, `-count=1`, from the repository root):

```sh
go test ./ui -run '^$' -bench . -benchmem -benchtime=2000x
go test ./internal/scene ./internal/render ./internal/semantic ./internal/input \
    -run '^$' -bench . -benchmem
go test ./app -run '^$' -bench . -benchmem -benchtime=300x
go test ./text -run '^$' -bench . -benchmem
go test ./components ./flex ./headless ./markdown \
    -run '^$' -bench . -benchmem -benchtime=500x
go test ./internal/grapheme ./internal/linebreak ./internal/textcache \
    ./internal/textmetrics -run '^$' -bench . -benchmem -benchtime=2000x
go test ./internal/platform/darwin -run '^$' -bench . -benchmem -benchtime=500x
```

Profiles were taken with `-cpuprofile` / `-memprofile` on the same benchmarks
(`-benchtime` raised until there were enough samples) and read with
`go tool pprof -top`. Phase attribution was done by temporarily commenting out
one call in `painter.publish` at a time (`syncRegions`, `syncSemantics`) and
re-running; the edits were reverted. Steady-state allocation was measured with
long `-benchtime` (100,000 iterations) so setup work is amortized away, because
`-benchmem` at a few hundred iterations otherwise reports setup allocations as
per-op allocations.

Profiles behind the attribution tables:

```sh
go test ./app -run '^$' -bench 'BenchmarkPainterLabelledPage/4k$' \
    -benchtime=1500x -cpuprofile labelled.cpu
go test ./app -run '^$' -bench 'BenchmarkPainterIconPageHoverToggle$' \
    -benchtime=15000x -cpuprofile iconhover.cpu
go test ./app -run '^$' -bench 'BenchmarkPainterModalToggle$' \
    -benchtime=5000x -cpuprofile modal.cpu
go test ./ui -run '^$' -bench 'BenchmarkWrapLinesCJKAndPath$' \
    -benchtime=20000x -cpuprofile wrap.cpu
go test ./text -run '^$' -bench 'BenchmarkCursorMoveMixedText$' \
    -benchtime=100000x -memprofile cursor.mem
go tool pprof -top -nodecount=25 <profile>
```

Struct sizes were read with `unsafe.Sizeof` in throwaway tests.

---

## Baseline highlights

Full suites were run; the numbers that matter for this audit:

| Benchmark | ns/op | allocs/op | Note |
| --- | ---: | ---: | --- |
| `app/PainterSteadyStateRepaint` | 246–329 | 0 | unchanged window: already at target |
| `app/PainterButtonPointer` | 995 | 0 | pointer move over a button |
| `app/PainterVirtualizedScroll` | 255 (100k iters) | 0 | 1,000-row list, visible window only; a 300-iteration run reports ~2 µs because setup is amortized over few iterations |
| `app/PainterIconPageHoverToggle` | 105,437 | 0 | 60 icons, one highlight appears/disappears |
| `app/PainterModalToggle` | 40,413 | 0 | dialog opens/closes each iteration |
| `app/PainterLabelledPage/1k` | 186,541 | 0 | 1,000 controls + labels |
| `app/PainterLabelledPage/4k` | 747,790 | 0 | 4,000 controls + labels |
| `input/HitTest10KMiss` | 12,793 | 0 | pointer over empty space on a 10k-region page |
| `input/FocusNext10K` | 8.5 | 0 | Tab traversal |
| `scene/SceneNoChanges` | 2.0 | 0 | drain with nothing queued |
| `scene/SceneLocalizedUpdate` | 18.1 | 0 | one node changed |
| `scene/SceneCreate10K` | 503,738 | 3.29 MB | includes slice growth |
| `render/StoreLocalizedUpdate` | 41.3 | 0 | one instance changed |
| `render/StoreAppendTail` | 657.7 | 3,041 B | one append after the last instance |
| `semantic/SemanticNoChanges` | 2.0 | 0 | |
| `semantic/SemanticLocalizedUpdate` | 22.5 | 0 | |
| `text/CursorMoveMixedText` | 1,149 | 8 / 384 B | one Left key through mixed text |
| `text/ReplaceComposition` | 58.1 | 2 / 32 B | composition update |
| `text/TypingWithUndoHistory` | 574.7 | 17 / 400 B | one keystroke, undo recorded |
| `ui/WrapLinesCJKAndPath` | 84,486 | 0 | ~330-byte CJK + path string, cached ellipsis |
| `ui/WrapLines` | 1,464 | 0 | short Latin paragraph |
| `ui/DrawIconDirect` / `DrawIconCached` | 3,859 / 314.7 | 16 B / 40 B | cache hit is 12× faster |
| `ui/GradientIconTessellate` / `Replay` | 16,617 / 270.2 | 65 / 59 KB vs 0 | tessellation cache works |
| `components/*` paint (14 cases) | 8–1,566 | 0 | declarations allocate nothing |
| `flex/DeclaredPage` | 15,393 | 0 | |
| `flex/TenThousandRowTable` | 3,838 | 0 | virtualization works |
| `gallery/GalleryPaint/table` | 16,287 | 0 | |
| `darwin/TextEngineWidthHit` | 31.5 | 0 | width cache hit |
| `darwin/TextEngineWidthMiss` | 10,068 | 2 / 50 B | shaping a new string |
| `darwin/MeasureTextMixed` | 19,436 | 1 / 48 B | full mixed-script measurement |
| `darwin/StageBitmapUpload/tile` | 6,168 | 1 / 320 B | one tile upload |
| `darwin/StageBitmapUpload/whole` | 732,029 | 1 / 8 B | whole-window bitmap |

The picture is not "Viva is slow". Idle repaint, layout, component declaration,
localized scene/render updates, focus traversal and text-width cache hits are
already at or near the framework's targets. The costs that remain cluster in
**per-repaint reconciliation on large pages**, **structural changes in the
renderer**, and **animation-time tessellation**.

---

## 1. Large pages: bookkeeping costs more than the paint

`BenchmarkPainterLabelledPage` repaints the same 4,000-control page over and
over. Nothing changes between iterations — no focus change, no label change, no
layout change — so a correct incremental pipeline should converge to "reuse what
is already published". It does not: every repaint rebuilds and re-compares
everything.

Phase attribution on the 4,000-control page (temporary edits, reverted; the
"index" row is an earlier prototype that only replaced `for _, hit := range`
copies with index access):

| Configuration | 4k | 1k |
| --- | ---: | ---: |
| Baseline | 747.8 µs | 186.5 µs |
| Index iteration instead of 232-byte `Hit` copies (prototype) | 645 µs (−14%) | 163 µs (−13%) |
| Prototype, `syncRegions` skipped | 548 µs | 137 µs |
| Prototype, `syncSemantics` skipped | 273 µs | 66 µs |
| Prototype, both skipped | 178 µs (−72%) | 44 µs (−73%) |
| Prototype, `nameFromLabels` skipped only | 516 µs (−20%) | 130 µs |

So on a large static page roughly **72% of the frame is framework bookkeeping
after the application's paint function has already finished**. Breaking that
down further:

- `nameFromLabels` alone is ~130 µs at 4k. It clears a map, inserts every
  label (an existence check plus an assignment each), then looks every control
  up in it — roughly 12,000 map operations per repaint, to produce a mapping
  that did not change since the previous frame.
- `syncSemantics` builds a 184-byte `platform.AccessibilityNode` for every
  control and compares the whole slice with the previous frame every repaint.
- `syncRegions` rebuilds the 48-byte `input.Region` slice, sorts it (already
  sorted, so an O(n) check), hands it to `Router.SetRegions` (which copies it
  again and scans for the focused slot), then `applyAutofocus` makes two more
  passes over the hits.
- `ui.Hit` is 232 bytes, and the painter's loops copy it by value; 4,000
  controls + 4,000 labels means ~1.9 MB of struct copies per repaint for the
  copies alone. Index iteration recovers ~14% at 4k.

### Proposed change A (primary): a published-paint gate

Give `Canvas` a cheap **hit digest** accumulated as hits are declared: a
128-bit hash (two independent 64-bit accumulators) over the scalar fields and
the label/value/hint bytes, plus the hit count. `painter` keeps the digest from
the last publish. When count, digest, focus, focus visibility, pending focus
request, trapped scope and the active text selection are all unchanged, reuse
the last published regions, semantic changes and accessibility payload instead
of rebuilding them.

Design notes that matter:

- Hash string bytes, not string headers. Paint functions recreate label
  strings (`fmt.Sprintf`) every frame at a stable address often enough that
  pointer comparison is not sound.
- Include the focused ID and the active editor's selection/version: the
  accessibility payload stamps both.
- 128 bits of collision resistance is enough in practice (2^-128), but note
  that gating regions also gates pointer hit testing: a collision would leave
  accessibility and input stale until something else changes. Keep a
  verification build tag that always does the full rebuild and compares, so the
  gate can be proven in tests rather than argued about.
- Hashing 4k hits should cost ~15–30 µs if words are hashed eight bytes at a
  time; against ~470 µs of bookkeeping that is the right trade. If that overhead
  is unacceptable, gate only `syncSemantics` (the measured 372 µs) and leave
  regions as they are.
- Keep a narrow fast path for pages under a threshold (say 64 hits): for small
  pages the hashing and the rebuild are both noise, and skipping the gate keeps
  the code simpler to reason about.

Estimate: 645 µs → ~200–280 µs at 4k, ~163 µs → ~60–90 µs at 1k, on top of the
index-iteration prototype. No change for normal pages. Verify with the existing
labelled-page benchmark plus new tests that change one label, one value, one
focus, and one text selection while the hit digest is otherwise equal, asserting
the payload still updates.

### Proposed change B (cheap, independent): name labels without a map

Labels are declared next to the control they name in practice (the label is
painted before or after its input). A single ordered pass can pair a label hit
with a control hit carrying the same ID when they are adjacent, and only the
unpaired labels need the map fallback. At 4k that removes most of the
~130 µs without touching the semantics contract; the "first label wins" rule
stays by taking the first adjacent or mapped label in paint order.

### Proposed change C (already prototyped): index iteration

Replace `for _, hit := range canvas.Hits()` with index access over
`hits := canvas.Hits()` in `syncRegions`, `syncSemantics` and `applyAutofocus`,
and merge `applyAutofocus`'s two passes into one. Measured −14%; zero behavior
change.
Also worth doing regardless of A: `Router.SetRegions` can compare first and
return when the incoming slice equals the retained one (48-byte records, memcmp
fast) instead of copying and rescanning.

### Why not cache at the layout level?

Layout is already value-type code and already allocation-free; the cost here is
not layout. The expensive part is mirroring the same declarative result into
three retained structures every frame. Caching that mirror is the right layer.

---

## 2. Structural changes: one new shape rebuilds the page

`BenchmarkPainterIconPageHoverToggle` paints 60 icons and 60 rounded rectangles;
hover adds or removes one highlight rectangle at index 7. The scene already
localizes this: `TestInsertingAShapeMidPageTouchesOnlyThatShape` asserts exactly
one scene change. The renderer does not.

- Measured: 107 µs per toggle.
- Same page, hover changes one icon's **color** instead of inserting a
  highlight: 71 µs. The 36 µs difference (34%) is the structural change.
- CPU profile of the toggle: `render.Store.rebuild` 27–37%,
  `painter.applyClipping`/closure 11–16%, `syncShapes`/`drawnBy` ~18%,
  `stampState` ~9%.

Why it rebuilds: `Store.Apply` only keeps the append fast path when every new
shape's previous sibling is the old last instance. A mid-page insert fails that
test and falls back to `rebuild()`, which clears `instanceBySlot`, walks every
node through the error-checked `live()` lookup, and re-emits every 152-byte
`Rectangle`. `publish` then sees `Update{Rebuild: true}` and the native backend
packs and re-sends **every** instance through cgo (`updateRectangles` sets
`first = 0, count = len(rectangles)`), instead of one instance.

### Proposed change

Teach the store a genuine mid-order insert/delete:

1. Find the insertion index (the new node's previous sibling's instance, or
   the first instance when it is first).
2. `copy(rectangles[at+1:], rectangles[at:])`, write the new instance at `at`,
   and bump `instanceBySlot` for the shifted suffix. Deletion is the mirror
   image.
3. Report an `Update` covering `[at, oldLast]`, and extend the native update
   format with an explicit `move` operation (destination index, source range)
   so Metal/D3D11/Vulkan `memmove` their instance buffer and only upload the
   new suffix, instead of repacking the whole page.

The Go-side memmove of 152-byte records is roughly an order of magnitude
cheaper than the current rebuild (which also does a slice append and a
per-node lookup); skipping the native full re-upload is the larger win for big
pages. Also grow `render`'s slot arrays geometrically in `ensureSlots`: today
it grows by exactly the missing count, which allocates one 168-byte node plus a
4-byte map slot per structural change (the scene already appends, which grows
geometrically).

Risk is in ordering invariants, so extend
`TestInsertingAShapeMidPageTouchesOnlyThatShape` to assert the `Update` range
covers only the moved suffix, and keep `storeMatchesCanvas` in the loop.

Estimate: icon hover 107 µs → ~75–85 µs (the color-only floor), and a dense
page's one-shape change stops scaling with page size: a one-shape change would
cost one shape, not the whole page.

---

## 3. Overlay fades re-tessellate their borders every frame

`BenchmarkPainterModalToggle` is 40.4 µs. Its profile is not layout or scene
work:

- `ui.(*earcutScratch).isEar` 41% flat, `Presence.apply` 65% cumulative,
  `components.Dialog.End` 24%.
- `BenchmarkVectorizeRoundedBorderDuringFade` (the isolated case) is 8.4 µs
  with zero allocations.

The cause is `vectorizeBorderBackings`: while a fading overlay is not fully
opaque, every `RoundedBorder` in it is converted from two rounded rectangles
into a vector ring so the backing cannot show through the fill. That rebuilds
arc sweeps and ear-clips the ring **on every frame of the fade**, although for a
pure fade the geometry does not change at all — only alpha does. Scale and slide
transforms do change geometry, and re-tessellation there is legitimate.

### Proposed change

Cache the triangulated ring keyed by `(bounds, radius, border width, layer,
clip)`; on a pure fade, reuse the triangles and only multiply alpha. For a
scale/slide, either accept re-tessellation (it is bounded and the overlay is
small) or store ring triangles in local space and transform their vertices when
appending. The cache belongs with the other scratch state on `Canvas` (bounded,
cleared by `Trim`) or on the presence entry.

Estimate: modal/popover animation ~40 µs → ~25 µs; the isolated vectorize
benchmark ~8.4 µs → sub-µs on a hit. Medium priority: 40 µs fits comfortably in
an 8.33 ms frame, but it is pure waste and it scales with the number of bordered
surfaces in the overlay.

---

## 4. Text: cheap wins and a strategic gap

### Cursor movement allocates a line copy for nothing

`text.Editor.visualStep` starts by building
`line := string(editor.runes[lineStart:lineEnd])` and only then asks
`bidi.NeedsBidi(line)`. For left-to-right lines — almost all of them — that
string is allocated and copied purely to be discarded. Measured:
`BenchmarkCursorMoveMixedText` is 1,149 ns with **8 allocations / 384 B** per
iteration (about one per arrow step through the line).

Fix: add a rune-slice bidi predicate (`bidi.NeedsBidiRunes`, or an inline scan
for RTL/format-control ranges) and do the early-out before any conversion. In
the rare bidi case, keep the string conversion but reuse `starts`, `visual` and
`odd` buffers on the `Editor` instead of allocating three slices per step.

Expected: 0 allocs on cursor movement; removes GC pressure from the most
latency-sensitive interaction (holding an arrow key).

### Grapheme properties: three lookups per rune

Mixed CJK/Latin wrapping is 84.5 µs for ~330 bytes (`WrapLinesCJKAndPath`).
Profile: `grapheme.Boundary` 72% cumulative, and inside it
`uniprops.tuple`/`flags`/`propOf` together 38%. `propOf` performs three
independent Unicode table lookups per rune (grapheme class, InCB, extended
pictographic). ASCII text takes three lookups too, even though a plain ASCII
letter is `GBOther`/`InCBNone`/not pictographic.

Fix, in order of payoff:

1. ASCII fast path in `propOf` (controls and CR/LF handled explicitly; the rest
   return one constant). This speeds Latin text in every wrap, caret and
   linebreak pass.
2. One generated range table returning `(gb, incb, pict)` in a single binary
   search, replacing the three lookups for non-ASCII.
3. In `breakPoint`'s final weak-break loop, iterate clusters with
   `grapheme.Next` once instead of calling `Boundary` at every byte offset.

Estimate: ~2× on mixed-script wrapping; the existing grapheme benchmarks
(ASCII 2.1 µs, Mixed 7.5 µs for their fixed strings) make the improvement easy
to verify by differential tests.

### Typing allocates 17 objects per keystroke

`TypingWithUndoHistory` is 574.7 ns / 17 allocs / 400 B per keystroke. Undo
history reasonably copies the changed range, but 17 allocations per character is
worth an audit: the editor already caches its string, so the remaining
allocations are likely history records, rune-slice growth and selection structs
in `record`/`merge`. Target: a handful of allocations, not zero (history must
retain data), and no per-key allocation for the common short insert.

### Strategic: per-label textures, no glyph atlas

The README already says it: native label renderers are single-line, and a glyph
atlas is future work. A text-heavy page pays one texture (and one upload) per
changed label; `StageBitmapUpload/tile` is 6.2 µs per tile, and the whole-window
bitmap path is 732 µs. Shaping itself is not the problem — a width-cache hit is
31.5 ns and a missing mixed-script measure is 19.4 µs — repetition is. A glyph
atlas (rasterize once, draw with UVs, evict textures) is the single largest
remaining text-side design project. It is not a micro-optimization and should be
its own ADR with a size/memory budget, but it belongs on the roadmap.

---

## 5. Hit testing and region publication

- `HitTest10KMiss` is 12.8 µs — 1.27 ns per region over 480 KB of `Region`
  records. That is close to memory bandwidth, so only a spatial index or a
  per-layer/bounds pre-filter can improve it. A uniform grid (or interval lists
  per row band) for pages above a threshold (say 512 regions), with the current
  linear scan kept for small pages, would turn a miss into tens of nanoseconds.
  Priority is low: this runs only on pointer events, and 12.8 µs is well inside
  a frame; it matters as insurance for very large non-virtualized pages.
- `Router.SetRegions` copies the whole region slice and rescans for the focused
  slot on every paint. An equality fast path (memcmp of 48-byte records) and
  reusing the previous focused slot when the regions are unchanged would remove
  most of that cost; together with the §1 gate it disappears entirely on
  unchanged pages.
- The `Hit` struct (232 B) is the common currency of paint, regions and
  semantics. Moving the text-field-only tail (`TextBounds`, `TextSize`,
  `TextColor` and the `Placement` interface — 52 B) into a side table indexed by
  hit would cut it to roughly 168–180 B, improving every one of the copies above.
  This is a wider refactor; do it only if §1 still leaves copying visible in
  profiles.

---

## 6. Memory: the renderer keeps a second copy of every shape

Measured struct sizes (`unsafe.Sizeof`):

| Struct | Bytes |
| --- | ---: |
| `ui.Hit` | 232 |
| `ui.Rectangle` / `ui.Triangle` / `ui.Text` / `ui.Image` | 60 / 104 / 96 / 120 |
| `scene.node` / `scene.Change` | 168 / 152 |
| `render.node` / `render.Rectangle` | 168 / 152 |
| `semantic.entry` / `semantic.Change` | 136 / 120 |
| `platform.AccessibilityNode` | 184 |
| `input.Region` | 48 |

Retained across frames, one shape costs: a `scene.node` (168 B), a `render.node`
(168 B), a `render.Rectangle` (152 B), the canvas `Rectangle` it was declared
as (60 B), a `shapeIDs` entry (8 B) and an `instanceBySlot` entry (4 B) — about
560 B, with the geometry present three times (`scene.node`, `render.node`,
`render.Rectangle`). The per-frame `clipped` copy (60–104 B) and `shapeRef`
(12 B) are transient on top of that. `SceneCreate10K` allocates 3.29 MB for 10k
nodes including growth.

Two changes are worth considering:

1. `render.node` only needs tree links, kind and the instance index; the
   geometry can live in `Rectangle` alone. That removes 168 B per shape and one
   full geometry copy per changed shape in `upsert`.
2. Make `scene.Change` kind-specific (a rectangle change needs 64 B of payload,
   not the 152 B that also carries triangle points, shade and shadow). It is
   transient, but a theme change queues a change per node.

Neither is urgent at today's page sizes; both matter for the stated "memory
scales with active nodes" goal on shape-heavy pages, and both reduce cache
pressure in `applyClipping`/`syncShapes`/`rebuild`. Item 1 landed after this
audit; §10 has the before/after measurement, and item 2 is still open.

---

## 7. Measurement gaps and hygiene

The audit found these non-code issues:

- **A checked-in benchmark fails its own assertion.**
  `internal/platform/darwin.BenchmarkSelectionUTF16Range` expects
  `{Start:4, End:15}` but `utf16Range` returns `{Start:4, End:16}` for
  `"Viva 原生界面 👩‍💻 A中😀B"`. The result is correct (offset 18 is the start of
  😀, rune 16; `TestUTF16RangeUsesRuneBoundaries` passes), so the expectation
  is stale. Today `go test ./internal/platform/darwin -bench .` fails, which
  means the darwin benchmark suite cannot be used as a gate.
- **No startup/first-frame benchmark.** The framework's target list includes
  empty-window first-frame time, idle RSS/CPU and goroutine counts, but there is
  no checked-in benchmark for them; the 330–370 ms and 34–54 MiB figures in the
  docs are old and above target. `BenchmarkBitmapStartup/initial` (2.1 µs, 28
  allocs) covers only the bitmap prepass. Add a headless first-frame benchmark
  (app construction + first paint + frame build) and a `viva build` timing
  harness; keep real-window hardware numbers on the test machines.
- **No automated idle/GPU-submission assertion in CI.** The driver test covers
  no frames scheduled while settled; `render.Update` counters make GPU
  submission assertions possible. Record "submissions while idle = 0" as a
  measured line, not only prose.
- **Benchmarks have no regression gate.** `benchmarks/history.sh` needs a
  manual machine label and stores nothing in the repo; `benchstat` is not wired
  into CI. A noise-tolerant CI job for the pure-Go subsystems (scene, render,
  semantic, input, text, ui wrap) would catch the next regression.
- **No comparable Linux/Windows physical-hardware run exists.** The recorded
  Linux and Windows numbers are lavapipe/WARP compatibility runs and correctly
  stay out of these tables; a physical run on the two test machines is still
  open.

Suggested new benchmarks to lock in the work above: large-page repaint with a
single changed label (not a no-op repaint), mid-page insert/delete
(`render` and `app`), overlay fade with geometry cache hit, cursor movement
allocation count, and non-ASCII wrap through `WrapRows`.

---

## 8. What is already at target (do not regress)

- Idle: `TestDriverAddsNoGoroutinesAndIdlesWhenSettled` and
  `TestIdlePainterSchedulesNoFrames` pass; a settled window repaints in 246 ns
  with zero allocations, and scene/semantic drains with nothing queued are 2 ns.
- Component declaration and layout: measured component paint cases and `flex`
  pages allocate nothing; `DeclaredPage` is 15 µs for a full page.
- Localized updates: scene 18 ns, render 41 ns, focus traversal 8.5 ns, small
  hit tests nanoseconds; virtualization keeps a 10,000-row table at 3.8 µs and
  a 1,000-row scroll at 255 ns.
- Text measurement cache: 31.5 ns on a hit vs 10 µs on a miss — keep texts
  stable and measured at their drawn weight.
- Icon and path caches: cached icon replay 270–315 ns vs 3.9 µs direct;
  `DrawPathCurveFill` 2.1 µs with scratch reuse; tessellation replay 270 ns vs
  16.6 µs.

Any optimization above must keep these numbers flat; these rows are the
comparison set for the next history record.

---

## 9. Suggested order

1. §1 B + C (label pairing, index iteration, `SetRegions` fast path) — small,
   independent, measurable on the existing 4k benchmark.
2. §1 A (published-paint gate) — the largest single win on large pages.
3. §4 cursor allocation + §4 grapheme properties — small, contained, benefit
   every app.
4. §2 renderer mid-order insert — biggest one-shape-change win, needs the
   native `move` operation and new invariants tests.
5. §3 fade tessellation cache — contained animation win.
6. §5 hit-test index, §6 memory de-duplication — scalability work, do after the
   bookkeeping is incremental so the profiles show what is left.
7. §7 measurement work in parallel: fix the failing benchmark, add the
   first-frame/idle benchmarks, wire a CI regression job.

---

## Limitations

- Single machine, single sitting, `-count=1`: 5–15% run-to-run noise; medians
  across repeated runs were not collected for every table row.
- The phase-ablation numbers come from temporary edits that skipped whole
  phases; they measure the ceiling of what incrementality can recover, not a
  ready implementation, and skipping `syncSemantics` also skips publishing
  accessibility, so it is not a valid runtime configuration as-is.
- No real window was opened. Startup, idle RSS/CPU, GPU submission counts and
  window-resize behavior were not re-measured; the existing docs' figures for
  those are older than this audit.
- Linux and Windows backends were not measured. Their renderer update paths
  differ, so the §2 native `move` work must be designed against all three, not
  just Metal.
- The memory numbers are `unsafe.Sizeof` of the structs plus the allocation
  totals the benchmarks report; they do not include allocator size classes,
  map overhead or native-side resources.
- The Web/WASM backend, the development reload worker and the native window
  creation path were not profiled; this audit covers the native runtime path.
- Binary size is outside this audit. It is a separate axis with its own
  measurements in the other reports under this directory.

---

## 10. Follow-up: what landed, re-measured

Implemented after the audit on the same machine and the same base commit
(`c883da0`, worktree changes uncommitted at the time of measurement). Before and
after were measured back to back: "before" from a detached worktree of
`c883da0`, "after" from this tree, both with the same `GOCACHE` and the same
benchmark flags. Numbers are two runs each; treat differences under ~10% as
noise.

```sh
go test ./app -run '^$' -bench 'BenchmarkPainterLabelledPage|BenchmarkPainterIconPageHoverToggle|BenchmarkPainterModalToggle|BenchmarkPainterSteadyStateRepaint' -benchtime=300x -count=2
go test ./text -run '^$' -bench BenchmarkCursorMoveMixedText -benchtime=5000x -count=2
go test ./ui -run '^$' -bench 'BenchmarkVectorizeRoundedBorderDuringFade|BenchmarkWrapLinesCJKAndPath' -benchtime=1000x -count=2
```

| Benchmark | Before | After | Change |
| --- | --- | --- | --- |
| `BenchmarkPainterLabelledPage/1k` | 183.6 µs | 66.0–72.6 µs | −62% |
| `BenchmarkPainterLabelledPage/4k` | 738.5 µs | 255.6–278.8 µs | −62 to −65% |
| `BenchmarkPainterSteadyStateRepaint` | 276–290 ns | 184–216 ns | within noise of the audit's 223 ns |
| `BenchmarkPainterModalToggle` | 40.0–40.3 µs | 13.3–14.2 µs | −66% |
| `BenchmarkPainterIconPageHoverToggle` | 104.4–105.0 µs | 91.9 µs | −12% |
| `BenchmarkCursorMoveMixedText` | 1,494–1,779 ns, 8 allocs, 384 B | 507–527 ns, 0 allocs, 0 B | −66% time, allocations gone |
| `BenchmarkVectorizeRoundedBorderDuringFade` | 12.3–14.0 µs | 0.72–0.74 µs | −94% |
| `BenchmarkWrapLinesCJKAndPath` | 90.2–92.1 µs | 73.4–74.7 µs | −19% |

The modal and fade numbers are larger than the audit's own before-numbers (40.4
and 8.4 µs) measured earlier with a different `-benchtime`; the before/after pair
in this table is the like-for-like comparison.

### §1 A+B+C — published-paint gate (largest win)

- `app/painter.go`: after painting, compare the new canvas hit list with the one
  published last frame (`bytes.Equal` over the `ui.Hit` bytes, plus focus, text
  editor and trap state). When nothing changed, skip `syncRegions`, the
  region sort, `router.SetRegions`, `applyAutofocus`, `syncSemantics`,
  accessibility diffing and `syncTextInput`'s label rebuild. Any hit-count,
  bounds, label, value, role, focus-scope, selection or composition change opens
  the gate; a pending programmatic focus bypasses it; the semantic-tree rebuild
  in `trim` marks the gate dirty.
- Comparing the raw `ui.Hit` bytes is deliberate: a per-hit struct key measured
  ~315 µs on the 4k page, `bytes.Equal` measured ~28–32 µs. The published slice
  keeps the previous frame's strings reachable, so an equal byte image is an
  equal hit; padding or a re-allocated but equal string can only cause a safe
  false miss.
- `syncRegions` and `syncSemantics` now iterate the canvas hit list by index
  instead of copying 232-byte `Hit` values.
- The gate retains one copy of the hit list (`publishedHits`, 232 B per hit:
  ~0.9 MB for a 4,000-hit page), which scales with the active node count rather
  than history and is trimmed by `trimPainterCapacity` like the other painter
  slices. That is the memory price of skipping the per-hit mirror.
- A repaint whose hits changed pays one extra scan of the hit list (the memcmp
  that detects the change: ~13 µs at 4k, measured), which is under 3% of the
  full mirror it is about to run. The second check after `syncTextInput` only
  re-reads focus and editor state, so the memcmp is not repeated.
- `internal/input`: `SetRegions` returns immediately when the new region slice
  equals the current one.
- §1 B is done for the common orders. `syncSemantics` now pairs a label with
  its control as it walks the hit list: the words patch the control just
  appended when the label follows it, or wait in a small pending table when the
  label comes first (how a form paints a title, then its field). Only labels
  that never meet their control, a second label for one control, or more than
  eight pending labels fall back to `nameFromLabels`. A page that changes one
  label on 4k controls went from 739 to 668 µs; at 1k from 183 to 153 µs.
  Tests cover label-before, label-after, another control in between, and
  first-label-wins.

### §2 — renderer mid-order insert (partial)

`internal/render`: a batch that only inserts shapes now splices them into the
instance list at the position its previous sibling names, renumbers the slots
after that point and publishes the moved suffix (`Update{First: at, Count:
len-at}`). All three backends already accept a growing suffix update
(`first+count == total`), so no native change was needed. Deletions still
rebuild, because every backend's retained store is sized to the old count
(`totalCount < _rectangleCount` is rejected); shrinking remains a native
follow-up for Metal, D3D11 and Vulkan.

Measured on the hover benchmark that alternates an insert (fast now) with a
delete (still a rebuild): 104.7 → 91.9 µs (−12%). There is no dedicated store
benchmark for the splice: a steady-state insert benchmark either grows the scene
without bound (every iteration adds a shape) or has to rebuild the store inside
the loop, so the app-level toggle benchmark is the guard. `store_test.go`'s mid-insert
test now asserts the spliced update instead of a rebuild; the painter test that
wipes renderer state still gets a rebuild because the store is empty (a store
that lost its state is a resync, not an insert).

### §3 — border-ring tessellation cache

`ui.Canvas` keeps up to eight `ringCacheEntry` values keyed by the two rounded
rectangles (bounds and radius) that define a ring. A fade reuses the cached
triangles and only re-applies color, layer, clip and `RectsBefore`; the key does
not include color, so the animation hits the cache every frame. Entry count and
triangle count are bounded, so a page that cycles through many borders does not
keep them all.

### §4 — text allocations

- `internal/bidi.NeedsBidiRunes` answers the early-out on the editor's rune
  slice, so `text.visualStep` no longer converts the line to a string just to
  discover it is left-to-right. The visual-step scratch slices now live on the
  `Editor` and are reused.
- `internal/grapheme`: ASCII answers its grapheme class inline (CR, LF, other
  C0/DEL are Control, the rest Other) instead of three table lookups, and
  `uniprops.GraphemeProps` reads one tuple for the three non-ASCII properties.
  A differential test asserts `NeedsBidiRunes` agrees with `NeedsBidi`.
- Measurement caveat: most of the audit's "72% in `grapheme.Boundary`" for
  `BenchmarkWrapLinesCJKAndPath` came from the benchmark's own per-cluster
  measurer (`clusterWrapper`), not from `WrapLines` itself. The landed fixes help
  the real boundary calls (and any cluster-aware measurer) but cannot remove
  that benchmark's measurement cost; the measured −19% is the honest result.

### §6 — memory: one copy of each shape's geometry

`internal/render.node` no longer stores bounds, color, radius, points, edge
mask, shade and shadow: it keeps the tree links, kind and liveness only
(`unsafe.Sizeof` 168 → 56 B). The flattened `Rectangle` is the single geometry
copy; a live shape reaches it through `instanceBySlot`, and a shape inserted by
the batch carrying `freshInstance` is read from that batch's changes. A
rebuild therefore no longer needs the node payload:

- deletions compact the instance list in place (one linear pass, no
  allocation);
- a deletion followed by a root-tail append — the common delete shape — is
  compacted and then appended, still without a walk;
- arbitrary inserts (a shape under a group, or in the middle) walk the tree for
  the new paint order and rewrite the list from the back. An existing instance
  only ever moves to a later position after compaction, so the descending write
  never clobbers an instance it has not read yet. No output array is allocated:
  the list grows through append.

Measured on 10,000 rectangles, both sides with the same probe (a temporary test,
removed after measuring; the "before" side ran in a detached worktree of
`c883da0`): retained bytes = `cap(nodes)×sizeof(node)` +
`cap(rectangles)×sizeof(Rectangle)` + `cap(instanceBySlot)×4` (+
`cap(order)×8` after the change), plus a `runtime.ReadMemStats` heap delta with
the inputs kept alive.

| Metric | Before | After |
| --- | ---: | ---: |
| Retained (struct math) | 371.1 B/shape | 240.0 B/shape (−35%) |
| Heap delta after GC | 3.81 MB | 2.50 MB (−34%) |
| `render.node` | 168 B | 56 B |
| `BenchmarkStoreDeleteRebuild` | 139–147 µs | 70–76 µs, same 3 allocs |

Across the whole per-shape cost in the audit's table (scene node 168 B + render
node + rectangle 152 B + canvas rectangle 60 B + `shapeIDs` 8 B +
`instanceBySlot` 4 B) this is roughly 560 → 460 B per active shape; the renderer
share drops by about a third. The extra `order` buffer (8 B/shape) is reused and
is included in the table above. `BenchmarkStoreDeleteRebuild` got twice as fast
because the compaction pass replaces the old full re-flatten; the arbitrary-
insert rebuild is the slowest of the three paths (walk plus descending write),
but it is a structural fallback, not a per-frame path, and has no dedicated
benchmark. The initial paint of a new store also takes the append path, which
moved `BenchmarkPainterFirstFrame` from 164–173 µs to 138–142 µs.

Not measured: real-window RSS and GPU-side memory on the three platforms. The
numbers above are retained Go heap for the render store; the charter still wants
target-machine RSS before this is called a finished memory result.

### §7 — measurement fixes

- `BenchmarkSelectionUTF16Range` no longer fails: the expected UTF-16 end offset
  for rune index 14 of `"Viva 原生界面 👩‍💻 A中😀B"` is 16, not 15 (the surrogate
  pair and the emoji before it each add a unit). It was the only failing
  benchmark in the repository.
- `benchmarks/ci.sh` plus `go run ./tools/perfhistory compare` now compare a
  pull request with its base revision on the same runner. A new `benchmarks`
  job in `.github/workflows/ci.yml` runs it for the pure-Go scene, render,
  input, text, wrap and painter benchmarks (platform, GPU and window benchmarks
  stay out so a shared runner can reproduce the result). The gate is
  noise-tolerant: 20% and at least 100 ns slower, or any new allocation — a
  first local run flagged a 280 → 323 ns micro-benchmark, which is exactly the
  noise a shared runner adds, hence the higher floor. The compare path was
  exercised locally against a worktree of `c883da0`: 17 of 18 candidate
  benchmarks had a base result and the final run passed.
- New benchmarks for the paths this audit changed: a label-changing large-page
  repaint (`BenchmarkPainterLabelledPageChanged`, the gate-open cost: 739 µs at
  4k, i.e. the full mirror, versus 255 µs for a no-op repaint), the renderer's
  delete fallback (`BenchmarkStoreDeleteRebuild`; 139–147 µs before the §6
  geometry change, 70–76 µs after, 3 allocs for a 10k instance page), non-ASCII
  `WrapRows`
  (`BenchmarkWrapRowsCJKAndPath`, 85.9 µs), and a test that arrow keys on a
  left-to-right line allocate nothing (`AllocsPerRun`).
### Consciously left open

- §2 deletions: an incremental shrink needs the retained instance store resized
  in Metal, D3D11 and Vulkan, and a mistake there is a rendering regression, not
  a slow path. It cannot be exercised in this session (no real windows on the
  three platforms), so the documented rebuild fallback stays.
- §5 hit-test index: 12.8 µs per pointer event on a 10,000-region page is about
  0.15% of one core at 120 Hz, while a grid adds per-region memory. That loses
  against the charter's first priority and is not worth the invariant surface.
- §6 item 2 (`scene.Change` kind-specific) is not done: it is transient, and
  shrinking it means a tagged union or per-kind change types through the scene
  and render APIs. §6 item 1 landed as the section above; real-window RSS on the
  target machines is still the missing measurement for it.

- A headless first-frame benchmark landed as `BenchmarkPainterFirstFrame`
  (construct, first paint, scene and frame build; 200 labelled controls:
  164–173 µs, 913 KB, 127 allocs before the §6 geometry change; 138–142 µs,
  913 KB, 126 allocs after, because the initial flatten takes the append path
  instead of a tree walk). Empty-window application construction, a
  `viva build` timing harness, an idle CPU/GPU-submission assertion beyond the
  existing driver test, and physical Linux/Windows runs remain open.

### Verification state

- `go test ./app ./internal/... ./ui ./text` passes.
- Repo-wide `go test ./...` has two environment-only failures in this sandbox:
  `cmd/viva` (`TestInitOutsideACheckoutUsesTheModulePath` rejects the checkout
  path when the session resolves it through a symlink) and
  `tools/appbundle` (`hdiutil` cannot run under the sandbox). Both are outside
  the changed packages and fail the same way on `c883da0`.
- §6 memory numbers are retained Go heap measured headlessly (probe removed
  after measuring), not RSS of a live window; see the section above.
- Not measured: Windows and Linux hosts, real windows, GPU submission counts,
  idle RSS/CPU. The mid-order insert contract was written against all three
  backends' existing suffix rules; it was only executed on Metal through the
  unit tests and benchmarks above.

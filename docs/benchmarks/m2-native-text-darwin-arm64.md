# M2 Native Text and IME: Darwin arm64

## Scope

This checkpoint validates one focused, single-line text editor through the full
macOS path:

```text
NSEvent -> NSTextInputClient -> Go text.Editor -> CoreText bitmap -> Metal
```

The renderer uses the system font and fallback chain. It does not bundle fonts,
a Unicode database, a shader compiler, or a third-party text engine. The stable
Go editor remains the source of truth for committed text, selection, and marked
composition; AppKit mirrors that state only to serve native input-method queries.

## Reference Environment

- Hardware: Apple M2 Pro
- Operating system: macOS, arm64
- Go: 1.26.5
- Rendering: CAMetalLayer and a precompiled Metal library
- Shaping/rasterization: CoreText and CoreGraphics
- Build: `-trimpath -ldflags='-s -w -buildid='`

## Reproducible Harness

Run:

```sh
./benchmarks/m2_text_darwin.sh
```

The harness creates a real Cocoa window, performs two marked-text replacements
followed by one committed Chinese/emoji replacement, closes the window, and
checks exact counters.

Observed on 2026-10-13:

```text
stripped_executable_bytes=1818818
gop_first_frame_ms=300.779
gop_gpu_submissions=4
gop_gpu_draw_calls=8
gop_rectangle_instances=12
gop_gpu_uploaded_bytes=96
gop_gpu_full_rebuilds=1
gop_gpu_partial_updates=0
gop_pointer_events=0
gop_text_events=3
gop_text_texture_rebuilds=5
```

Four submissions are expected: one initial presentation and one presentation
for each of the three text mutations. There is no idle submission loop. Two draw
calls per submission are expected because the demo has one rectangle batch and
one text texture.

## Real Input Method Validation

The macOS Pinyin input method was also driven through physical-key-equivalent
events rather than the deterministic test hook. AppKit displayed its native
candidate panel at the rectangle reported by `firstRectForCharacterRange`, then
committed `你好` into the Go editor.

- Composition evidence: `docs/images/m2-native-ime-composition.png`
- Commit evidence: `docs/images/m2-native-ime-commit.png`

The same run remained at 0.0% idle CPU. It closed with 18 native text events and
19 GPU submissions: one initial submission plus one for each event. No redraws
were observed while it was idle.

## Current Limits

- The native adapter currently exposes one focused single-line editor.
- Text mutation rebuilds one bounded text texture. A retained glyph atlas is a
  later optimization for many labels and editors.
- Cursor blinking is intentionally absent until it can be scheduled without a
  permanent application heartbeat.
- The measured physical footprint with a live IME session was approximately
  39 MiB, above the provisional 15 MiB target.
- First-frame measurements remain approximately 300-620 ms in this development
  launch environment, above the provisional 50 ms target.
- Core grapheme movement is a compact approximation and still needs Unicode
  conformance work.

These misses remain explicit engineering work; they are not represented as
meeting the final runtime budgets.

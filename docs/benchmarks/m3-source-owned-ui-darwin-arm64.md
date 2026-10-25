# M3 Source-Owned UI Showcase: Darwin arm64

## Scope

This checkpoint validates the first visible UI stack built from typed Go theme
tokens and source-oriented component recipes. The showcase includes navigation,
cards, metrics, status badges, a chart, activity rows, buttons, and a live native
text input. It uses Cocoa, CoreText, CAMetalLayer, and Metal directly; it contains
no HTML, CSS, JavaScript, WebView, browser engine, or runtime component registry.

## Reproduce

```sh
go run ./cmd/showcase
./benchmarks/m3_showcase_darwin.sh
```

Visual evidence: `docs/images/m3-native-showcase.png`.

Observed on Apple M2 Pro / macOS arm64 / Go 1.26.5:

```text
stripped_executable_bytes=1836418
gop_first_frame_ms=367.102
gop_gpu_submissions=1
gop_gpu_draw_calls=55
gop_rectangle_instances=51
gop_gpu_uploaded_bytes=1632
gop_gpu_full_rebuilds=1
gop_gpu_partial_updates=0
gop_pointer_events=0
gop_text_events=0
gop_text_texture_rebuilds=0
```

The static showcase presents exactly once and performs no idle GPU submissions.
Its idle CPU was 0.0%. The stripped executable remains below the provisional
4 MiB target at approximately 1.84 MB.

## Source Ownership

The development-only component command lists and copies selected recipes:

```sh
go run ./cmd/gop list
go run ./cmd/gop add --out ./components/ui button card badge
```

Generated components import only the compact `gop/ui` primitives. Existing files
are not overwritten unless `--force` is provided. The CLI and unselected recipe
templates are not imported by release applications.

## Current Limits

- Static labels currently use one CoreText texture and draw call per label.
  Retained atlas packing and batched glyph instances are required before this is
  the final many-label renderer.
- Rectangle corners are sharp. Rounded geometry should be introduced as a
  measured render primitive rather than a general CSS-like style mechanism.
- Component functions paint visual state; runtime interaction/state composition
  still needs a stable public orchestration API.
- First frame remains above the provisional 50 ms target in the development
  launch environment.
- The current per-label texture path raised this showcase's physical footprint
  to approximately 54 MiB. Atlas packing is required to recover the memory
  budget before the text renderer can be considered complete.

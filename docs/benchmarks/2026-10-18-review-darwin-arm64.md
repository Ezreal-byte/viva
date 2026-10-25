# Review checkpoint: sizes, allocations and idle behavior (2026-10-18)

> **Update 2026-10-22:** this document records the baseline before optional
> feature isolation and the Trim work. Current sizes are 3,006,242 B for the
> macOS hello program, 3,965,106 B for the macOS showcase, 3,488,256 B for the
> Windows hello program and 3,678,437 B for the Linux one. The remaining
> numbers were taken after the optional-feature isolation and Trim work landed.

Replaces the size and allocation figures of the M1 and M3 documents, which
describe a much smaller framework. Nothing here is a design target; every figure
is a measurement, and what was not measured is listed at the end.

## Environment

- Hardware: Apple M2 Pro, macOS 15.5 (Darwin 24.5.0).
- Go 1.26.5, `darwin/arm64`, CGO enabled for the macOS build only.
- Build flags: `-trimpath -ldflags='-s -w -buildid='` (`-H=windowsgui` for Windows).
- Source: this repository after the value-type layout and Linux IBus changes.

## Stripped executables

| Program | Platform | Bytes | Against the 4 MiB target |
| --- | --- | ---: | --- |
| `examples/guides/hello` | macOS arm64 | 2,951,330 | within |
| `examples/guides/hello` | Windows amd64 | 3,274,752 | within |
| `examples/guides/hello` | Linux amd64 | 3,498,213 | within |
| `cmd/showcase` (the component gallery) | macOS arm64 | 3,596,178 | within |
| Motty (a real product, with SSH, SFTP, terminal, AI) | macOS arm64 | 16,154,194 | not applicable |

The Linux build grew by 61,440 bytes (1.8%) with the IBus client and the D-Bus
dialer it uses; the macOS and Windows builds do not link them. The macOS
showcase grew by 256 bytes with the new `ui` primitives. The M1 and M3 documents
report 1.77 MB and 1.84 MB; the difference is the framework that was added since
(component catalog, native text editing, status items, window controls, secret
store, file drops, ...), not this change.

## Allocations and time (`go test -bench`, `-benchmem`)

| Benchmark | ns/op | allocs/op |
| --- | ---: | ---: |
| Unchanged repaint (`BenchmarkPainterSteadyStateRepaint`) | 224 | 0 |
| Pointer move over a button (`BenchmarkPainterButtonPointer`) | 970 | 0 |
| Hover toggling across a page of icons (`BenchmarkPainterIconPageHoverToggle`) | 121,194 | 0 |
| Modal open and close (`BenchmarkPainterModalToggle`) | 1,274 | 0 |
| Hit test on 10,000 regions, miss (`BenchmarkHitTest10KMiss`) | 8,182 | 0 |
| Focus traversal over 10,000 regions | 7 | 0 |
| Scene drain, unchanged 10,000 nodes | 2.0 | 0 |
| Scene localized update of one node | 14.7 | 0 |
| Component declaration, all 332 `BenchmarkPaint` cases | 235 to 3,537 | 0 |
| `ui.IDBlock.At` and `Index` | 0.3 | 0 |
| `ui.Rect` edge cuts, three in a row | 0.3 | 0 |

Component declarations allocate nothing in any of the 332 cases; the slowest is
the calendar at about 3.5 µs. `Button` click-and-paint allocates once per op in
`BenchmarkClickAndPaint/button` (23 B), the switch animated frame once (16 B).

## Idle behavior

`TestDriverAddsNoGoroutinesAndIdlesWhenSettled` (new) drives a window through
hover, click, a transition, a resize and a background `Invalidate`, and checks
that no goroutine was started at any step, that the transition settles, that a
settled window paints nothing in a further minute of virtual time, and that
closing leaves the goroutine count where it began. It exercises the same painter
as a window, through the headless driver; it does not open a real window.

## Fuzzing

New targets, run for 25 to 30 seconds each on this machine:

| Target | Executions | Result |
| --- | ---: | --- |
| `text.FuzzEditorOperations` (insert, IME marked text, delete, move, UTF-16 conversion, restricted fields) | 2,227,978 | passes |
| `ui.FuzzWrapLines` | 943,839 | passes |
| `input.FuzzRouterKeepsFocusValid` | 7,248,291 | passes |
| `ibus.FuzzDispatchNeverPanics` | 11,090,196 | passes |

The seed corpora, and fixed-seed random runs of the editor, also run in every
`go test`. Before they passed they found two defects, both fixed in the same
change: `Canvas.WrapLines` panicked on text that is not valid UTF-8, and
`text.Editor` kept a composition range past the end of the text when the user
deleted after an input method had cleared its marked text.

## Not measured

- Startup time to the first frame, idle resident memory and CPU of a real
  window, and GPU submissions: they need a real window, which this review did
  not open on this Mac. The M1 and M3 figures (about 330 to 370 ms to the first
  frame, 34 to 54 MiB resident) are old and above their targets; they have not
  been refreshed.
- Linux and Windows timing or memory: the test machines run Direct3D on WARP and
  Vulkan on lavapipe, which say nothing about GPU performance.
- Allocation figures on Linux and Windows: the benchmarks above ran on macOS.

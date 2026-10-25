# M2: Native text selection and blur handoff

## Scope and environment

Measured October 13, 2026 (Asia/Shanghai) on Apple M2 Pro, macOS 15.5
(24F74), Go 1.26.5 darwin/arm64. Native UI checks used the Metal Gallery.
The baseline is the interrupted implementation with pointer selection events
already added, before this continuation's Command-A and blur fixes. This is
not a comparison against a released version.

The application owns the text and rune-indexed selection. AppKit hit testing
produces UTF-16 positions, snaps to composed-character boundaries, and sends
`TextSetSelection` to that owner. Command-A uses `performKeyEquivalent:` and
`selectAll:` rather than depending on text key bindings to deliver a command
key. Repeated identical selections do not emit another event. Disabling or
closing the native editor cancels dragging and its caret timer.

`Canvas.Focused()` now reports actual pointer or keyboard focus;
`Canvas.FocusVisible()` reports only the focus-ring target. All repository
call sites and source templates use the new API, with no compatibility entry
point. Development protocol v2 carries the visibility bit explicitly and
rejects v1. Gallery hides the static Email label only while that field owns
actual focus. A blur frame restores the current value and disables the native
editor together, without deriving ownership from the last click.

## Automated checks

```sh
go test ./...
go vet ./...
go test -race ./app ./internal/text ./internal/gallery ./cmd/showcase ./internal/reloadwire ./internal/devreload
go test ./internal/platform/darwin -run TestNativeTextSelection -count=1 -v
```

All passed. The native test compiles the actual Objective-C implementation
into a test-only executable with an event-recording view. It covers UTF-16
surrogate pairs, combining accents, ZWJ emoji, flags, outside-edge dragging,
both drag directions, Command-A modifier filtering, unchanged selections,
composition cancellation, and disable/close cleanup. A loop of 1,000 unchanged
selections produces zero selection events and zero texture rebuilds. Its
observed native view instance size is 848 bytes; the fix adds no native ivars.

App and Gallery tests cover background clicks, another control, Tab, refocus,
Go-owned text/selection retention, and an unchanged blurred repaint producing
no rectangle update, label update, or editor update. Codec tests reject old
protocol versions, truncated payloads, and invalid visibility bits.

Windows and Linux amd64/arm64, plus js/wasm, passed whole-repository
`go test -exec=true ./...` and `go vet ./...`. Those are compile checks, not
execution on those platforms. This change does not claim Windows pointer
selection, native multiline editing, or Web text-input parity.

The native UI checks run a standalone Gallery build, not `gop dev`.
The development host currently forwards paint/click/focus state but does not
transport `TextInputProvider` or text events. Native adapter edits also require
a host restart rather than a worker-only swap. Editor parity in development
mode remains an explicit follow-up, not a result of the focus protocol update.

## Timing, allocations, and artifact size

```sh
go test ./app -run '^$' -bench BenchmarkPainterButtonPointer -benchmem -count=3
go test ./internal/text ./internal/platform/darwin -run '^$' -bench 'CursorMoveMixedText|SelectionUTF16Range' -benchmem -count=3
go build -trimpath -ldflags='-s -w -buildid=' -o build/showcase ./cmd/showcase
```

| Measurement | Baseline | Completed fix |
| --- | ---: | ---: |
| Stripped Gallery executable | 2,790,754 bytes | 2,790,770 bytes |
| Canvas structure, `unsafe.Sizeof` | 184 bytes | 184 bytes |
| Button down/up benchmark, median of 3 | 570.6 ns/op | 576.0 ns/op |
| Mixed-text cursor movement, median of 3 | 794.2 ns/op | 783.6 ns/op |
| UTF-16 selection conversion, median of 3 | Not captured | 80.92 ns/op |

All three listed microbenchmarks report 0 B/op and 0 allocs/op for their
specific fixtures. This is not a claim that the complete native selection or
texture-rendering path is allocation-free. Three timing samples do not establish
statistical significance; the approximately 1% pointer difference is reported
rather than claimed as an improvement.

The focus-visibility boolean occupies existing Canvas padding. Concurrent
animation work subsequently added a `motionState` and changed the live
checkout's Canvas measurement to 232 bytes. That later measurement is not
attributed to this fix, and the executable sizes above refer to the captured
pre-animation validation binaries. No new runtime dependency is introduced.
Installer size and cross-platform binary deltas were not measured.

## Native UI verification

The final validation build visibly passed Command-A highlight/replacement,
click-to-place insertion, Left/Backspace, partial mouse-drag selection and
replacement, background blur, and Tab blur. The edited value remained visible
after both kinds of blur. Automated Unicode fixtures passed; the UI automation
driver omitted non-ASCII characters in one typing attempt, so that attempt is
not counted as Chinese IME or emoji input verification.

After blur, two `ps -o rss=,%cpu=,time=` samples three seconds apart reported
0.0% CPU and unchanged accumulated CPU time (2.59 seconds); the first RSS sample
was 50,208 KiB for the full Gallery, not the minimal-window target. GPU counters
were not sampled across that idle interval. The no-work claim is bounded to
the native timer/selection tests and unchanged-frame regression, not an
unmeasured whole-process GPU or memory guarantee.

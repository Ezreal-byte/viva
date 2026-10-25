# Transitions and Frame Scheduling: Darwin arm64

Reference machine: Apple M2 Pro, macOS 15.5, Go 1.26.5, release-mode Go
benchmarks (`go test -bench`), no other workload pinned.

## Cost when nothing animates

An application that never calls `Canvas.Animate` pays one frame bracket per
repaint and never reads the clock.

```sh
go test ./ui -run '^$' -bench FrameBracket -benchmem
```

```text
BenchmarkFrameBracketWithoutAnimation   2.0 ns/op   0 B/op   0 allocs/op
```

Existing painter benchmarks, built from the tree with and without the frame
hooks and run interleaved (6 runs each, 2 s, minimum shown):

```text
                                      before      after
PainterSteadyStateRepaint ns/op        138.0      140.4
PainterButtonPointer      ns/op        595.1      599.2
allocs/op                                0          0
```

The difference is inside run-to-run noise (about 5 ns).

## Cost while a transition runs

```sh
go test ./ui ./components ./app -run '^$' -bench 'Animate|Easing|SwitchAnimatedFrame|AnimatedSwitchFrame' -benchmem
```

```text
BenchmarkAnimateSettled64             452.7 ns/op   0 allocs   (7 ns per settled slot)
BenchmarkAnimateTransitioning64      3081.0 ns/op   0 allocs   (48 ns per moving slot)
BenchmarkEasingInOut                   43.6 ns/op   0 allocs
BenchmarkSwitchAnimatedFrame          200.5 ns/op   0 allocs   (paint one switch, animated)
BenchmarkPainterAnimatedSwitchFrame   378.8 ns/op   0 allocs   (full repaint, scene diff, store)
```

`TestMidFlightFramesDoNotAllocate` additionally asserts exactly zero allocations
per mid-flight repaint through the painter, scene, and renderer store.
`BenchmarkPainterAnimatedSwitchFrame` reports 6 B/op because it toggles the
switch every 20 frames; a memory profile attributes it to
`semantic.Tree.DrainChanges` on the toggle frame, where the switch's checked
state changes. It is not a per-frame cost.

## Real window, real event loop

A scratch program (not committed) painted one `Switch` with `app.RunCanvas`,
toggled its state, and recorded every `Paint` call.

```text
150 ms theme transition: 17 paints, 0.5 ms .. 152.6 ms after the toggle,
                         ~9.5 ms apart (8.33 ms timer + event-loop latency)
  3 s transition       : 323 paints, last at +3004 ms, then stopped
idle after settling    : 0 paints in 1.5 s
```

Three screenshots taken while the 3 s transition ran and as it ended showed the
thumb at three distinct positions and the track fading from the muted input
color to the primary color, ending in the final state.

No parked goroutine, timer, or GPU submission exists while idle: the timer is
armed only by a repaint that left a transition in flight, and the wake-up goes
through `platform.Waker`, not a channel reader.

## Binary size

`go build -trimpath -ldflags="-s -w"` of `cmd/showcase` from identical trees that
differ only in this change (Switch recipe, painter hooks, `RunCanvas` wiring,
darwin `Waker` binding):

```text
stripped showcase   before 2,790,930 B   after 2,807,474 B   (+16,544 B)
stripped pulse      before 2,029,266 B   after 2,045,778 B   (+16,512 B)
```

Both deltas are one 16 KiB Mach-O page; the symbol tables show the real change
is far smaller. Unstripped `pulse` (an app that never reaches `RunCanvas`) grew
by 224 B, unstripped `showcase` by 1,488 B. Symbols added to `showcase` that
belong to this feature total about 4.1 KB (transition table and `Animate` 1.3 KB,
easing solver 0.8 KB, `Color.Mix` 0.2 KB, frame timer 0.7 KB, `Waker` 0.9 KB,
the rest closures and cgo stubs), partly offset by renumbered cgo stubs. Both
binaries stay well under the 4 MiB target.

## Not measured

- Windows and Linux frame cadence: only compiled (`GOOS=windows`/`linux go vet`)
  and unit-tested.
- Latency from input to the first moving frame under heavy GPU load.
- ProMotion pacing with the window occluded; `nextDrawable` can time out (1 s)
  while the window is covered, which skips presents but not transition time.

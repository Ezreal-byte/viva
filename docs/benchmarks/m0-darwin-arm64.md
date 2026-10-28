# M0 Native Pulse Baseline: macOS ARM64

Date: 2026-10-12

This is an implementation baseline, not a generalized performance claim.

## Environment

- Hardware architecture: Apple ARM64.
- Operating system: macOS 15.5 (Darwin 24.5.0).
- Go: 1.26.5, `darwin/arm64`, CGO enabled.
- Compiler: Apple clang 17.0.0.
- macOS SDK: 15.5.
- Build flags: `-trimpath -ldflags='-s -w -buildid='`.
- GPU path: Cocoa + CAMetalLayer + Metal.

## Observed Results

| Metric | Result | Budget | Goal | Status |
| --- | ---: | ---: | ---: | --- |
| Stripped executable | 1,697,506 bytes | <= 4 MiB | <= 4 MiB | Pass |
| Settled CPU | 0.0% | effectively 0% | effectively 0% | Pass |
| Idle GPU submissions | 0 after initial present | zero | zero | Pass |
| Total GPU submissions during unchanged run | 1 | one initial present | one initial present | Pass |
| Physical footprint after real Metal present | 26.3–26.8 MiB | <= 32 MiB | <= 15 MiB | Pass |
| Peak physical footprint | 31.3–32.1 MiB | not yet set | not yet set | Baseline |
| First scheduled frame | 123.075–373.643 ms | <= 400 ms warm | <= 50 ms warm | Pass |
| Normal termination status | 0 | 0 | 0 | Pass |

Status is judged against the budget. The goal column is the long-term target: a
result below it is reported as a gap, not treated as a regression, until the
fixed platform cost has been separated from framework-owned cost.

RSS reported by `ps` was roughly 55–61 MiB, but this includes shared mappings and
is not used as the primary resident-memory metric. The table uses macOS
`vmmap -summary` physical footprint.

The original 15.0 MiB observation occurred before a Metal command was actually
submitted and was therefore rejected as an invalid GPU-runtime baseline.

## Rendering Behavior

The application performs one explicit clear/present after the window is shown.
It has no display link, animation timer, polling loop, or permanent redraw
callback. Five settled one-second process samples reported 0.0% CPU, and the
diagnostic counter reported one total Metal submission before termination.

## Interpretation

M0 proves that a direct Go/Cocoa/Metal path can remain far below the binary-size
budget and completely quiet while idle. It meets the startup and
physical-footprint budgets.

The goals stay at 15 MiB and 50 ms and are not met: the fixed AppKit/Metal cost
has not been separated from framework-owned memory yet. The budgets sit at the
measured envelope so that a regression is still caught; the next investigation
should separate fixed AppKit/Metal process cost from framework-owned memory,
then bring the goals down towards 15 MiB and 50 ms.

The startup measurement begins in a native image constructor before Go runtime
startup and ends when the first Metal command buffer is scheduled. It does not
claim that the frame has reached the display hardware.

## Reproduction

Run:

```sh
./benchmarks/m0_darwin.sh
```

The harness builds a stripped executable, waits for the first Metal submission,
allows the process to settle, captures process and `vmmap` metrics, requests
normal Cocoa termination, and reports the final submission count and exit code.

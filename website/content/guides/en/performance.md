---
title: Performance and size
summary: The budgets Viva is built to, what has actually been measured, where it misses, and what to avoid in your own code.
order: 15
group: more
---

Viva is built for small binaries and low idle cost. Those are **design targets**, not claims. Every result below comes from a report in the repository, on the machine and build named in it. Treat them as baselines for that machine rather than promises about yours.

## Budgets

For a minimal native application:

| Budget | Target |
| --- | --- |
| Stripped executable | at most 4 MiB (review above 8 MiB) |
| Warm start to first interactive frame | at most 50 ms |
| Idle resident memory | at most 15 MiB |
| Idle CPU, GPU submissions and presents | effectively zero |
| Framework goroutines while idle | at most 2 |
| An unchanged frame | no layout, no scene rebuild, no GPU submit |

When goals conflict, the order is low memory, then runtime performance, then small artifacts, then correctness, then ergonomics and cross-platform consistency, and last the optional browser target.

## What has been measured

M0 is the minimal macOS window: a Metal clear and present with no components. Measured on Apple silicon, macOS 15.5, Go 1.26.5, built with `-trimpath -ldflags='-s -w -buildid='`.

| Metric | Result | Target | Verdict |
| --- | --- | --- | --- |
| Stripped executable | 1,697,506 bytes | at most 4 MiB | pass |
| Settled CPU | 0.0% | effectively zero | pass |
| Idle GPU submissions | 0 after initial present | zero | pass |
| Physical footprint after a real Metal present | 26.3–26.8 MiB | at most 15 MiB | **miss** |
| First scheduled frame | 123.075–373.643 ms | at most 50 ms warm | **miss** |

The report is explicit about both misses. An earlier 15 MiB figure was taken before any Metal command was submitted, and was rejected as a baseline. The fixed cost of AppKit and Metal has not yet been separated from Viva's own, so the footprint target stands but is not met. The full reports, with machine details and reproduction commands, are in the repository's `docs/benchmarks/`.

The retained scene (M1) drains an unchanged 10,000-node scene in 2.012 ns with no allocations, and updates one node of 10,000 in 12.39 ns with no allocations.

## What to avoid in your own code

Viva can only keep the idle cost at zero if the application does too.

- **Do not repaint on a timer.** Call `Invalidator.Invalidate` when state changes, and let the window sleep otherwise.
- **Do not allocate in `Paint`.** It runs on every repaint, and the framework's own steady-state paths are written to allocate as little as possible, close to nothing. Reuse buffers and keep per-frame work proportional to what changed.
- **Keep callbacks short.** They run on the window thread. Do slow work on another goroutine and publish the result through `OnWake`.
- **Link only what you use.** Optional subsystems live in their own packages and build tags, so an unused one adds no code to your binary.

## Reproducing

The `benchmarks/` directory holds the scripts behind the macOS and Linux baselines, and every report names the command that produced it. `viva build` reports a program's size on its own, separate from anything around it.

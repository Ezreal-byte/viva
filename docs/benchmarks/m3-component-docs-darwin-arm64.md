# Component documentation boundary — macOS arm64

Date: 2026-10-14. Reference machine: Apple M2 Pro, macOS 15.5,
Go 1.26.5, arm64. Native builds use `-trimpath -ldflags='-s -w -buildid='`.

## Linkage and artifact sizes

| Artifact | Bytes |
| --- | ---: |
| `cmd/pulse`, before documentation tooling | 2,045,778 |
| `cmd/pulse`, after documentation tooling | 2,045,778 |
| `cmd/component-preview`, six explicit examples | 2,691,794 |
| Browser component-preview WASM | 3,005,140 |

Both pulse binaries have SHA-256
`43f906d44356251c66380588ae880cdc33c9c247882f2264474d92d543f5ee51`.
`go list -deps ./cmd/pulse` contains neither `internal/componentdemo`,
`tools/docs` nor `internal/platform/web`. Astro/Svelte exist only in the
`website/` npm project; shadcn-svelte/Bits UI/Tailwind and the typed Chinese/English
dictionary are likewise website-only. No Go module dependencies were added.

These are executable/WASM bytes, not installer or compressed transfer sizes.
The all-example native preview intentionally links all six demonstrations;
ordinary applications do not. No startup/RSS comparison is claimed here:
the minimal native executable is byte-identical, and this change adds no
persistent structure or worker to its runtime. Separate backend timing,
memory and idle-GPU measurements remain governed by the existing harnesses.

## Shared example paint benchmark

```sh
go test ./internal/componentdemo -run '^$' -bench '^BenchmarkPaint$' \
  -benchtime=50ms -count=1
```

All 19 cases at 320px and 760px width measured **0 B/op, 0 allocs/op**
for warmed, reduced-motion static paint. Representative timings:

| Case | Width | ns/op |
| --- | ---: | ---: |
| Button / default | 320 | 198.6 |
| Button / variants | 760 | 587.0 |
| Dialog / confirm | 760 | 581.9 |
| Dialog / destructive | 320 | 584.6 |

These are Go command-generation microbenchmarks, not rendering or
input-to-present latency. They exclude native text snapshot conversion;
nonempty editor snapshots allocate a UTF-8 string. See the package README for
the native-text benchmark and explicit clock ownership.

## Reproduce the isolation check

```sh
go build -trimpath -ldflags='-s -w -buildid=' -o build/pulse ./cmd/pulse
go list -deps ./cmd/pulse
go test ./internal/componentdemo ./tools/docs ./cmd/component-preview ./cmd/gop
cd website
npm ci
npm run check
npm test
npm run build
```

The site is development tooling and an optional static deployment. Its WASM
preview uses Canvas2D and does not claim Metal/D3D performance or browser IME
support. Native Input editing is validated separately from its visual-only
browser example.

# Windows Backend Compatibility Check: Wine amd64

## Scope

This checkpoint exercises the experimental Win32 / D3D11 backend through
Wine 11.18 and DXVK on an Apple M2 Pro. It verifies Windows ABI layouts,
DirectWrite measurement, bounded GDI label rasterization, D3D11 texture upload,
retained updates, embedded DXBC identity, and application teardown. It is a
compatibility check, not Windows performance evidence.

The text boundary deliberately keeps DirectWrite for measurement and uses a
top-down GDI DIB only on a label cache miss. Coverage is converted to
premultiplied BGRA before the immutable texture is created. Unchanged frames and
geometry-only updates do not rasterize labels. Direct2D, WIC, and a runtime
shader compiler are absent from the application strings.

## Environment

- Host: Apple M2 Pro, macOS 15.5 (24F74), arm64.
- Go: 1.26.5.
- Guest ABI: Windows amd64, `CGO_ENABLED=0`.
- Runtime: Wine 11.18, DXVK 1.10.3 macOS build, MoltenVK.
- Build: `-trimpath -ldflags='-s -w -buildid='`.
- Date: 2026-10-13.

## Results

- All Windows package tests passed under Wine, including a mixed Latin/Chinese
  measurement case and a nonempty premultiplied GDI raster result.
- Unchanged renderer benchmark: 2.818 ns/op, 0 B/op, 0 allocs/op.
- Mixed DirectWrite measurement benchmark: 536,100 ns/op, 702 B/op,
  25 allocs/op. Wine timings are not representative of Windows hardware.
- Gallery startup: 1 GPU submission, 1 full rebuild, 45 label texture rebuilds.
- Synthetic pointer press/release: 3 GPU submissions, 1 full rebuild,
  3 partial updates, 2 pointer events, and 1 changed label texture after startup.
- Synthetic Tab input: 2 GPU submissions, 2 full rebuilds, 1 key event, and no
  label texture rebuild after startup.
- Stripped amd64 Gallery: 2,170,880 B, below the provisional 4 MiB target.
- The text-path change increased the stripped amd64 Gallery from 2,166,784 B
  to 2,170,880 B: +4,096 B.

Raw logs are retained in `build/research/windows-platform-tests-gdi.log`,
`build/research/windows-gallery-gdi-dxvk.log`,
`build/research/windows-gallery-gdi-click.log`, and
`build/research/windows-gallery-gdi-key.log`.

## Remaining Evidence

- Run the same binaries on Windows 11 amd64 and arm64 hardware.
- Compare DirectWrite measurement bounds with GDI raster bounds across Latin,
  Chinese, emoji, combining marks, bidi text, and high DPI.
- Measure first interactive frame, idle RSS/CPU, goroutine count, and D3D11
  submissions on Windows rather than Wine.
- Implement and validate TSF/IME, clipboard, UI Automation, and native text
  editing before treating the backend as M2-complete.

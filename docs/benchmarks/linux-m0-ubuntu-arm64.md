# Linux M0 Native Pulse: Ubuntu arm64

## Scope

This checkpoint exercises the experimental Linux M0 shell through XCB and
Vulkan. The application dynamically loads the system `libxcb.so.1` and
`libvulkan.so.1`, creates one standard titled X11 window, clears one swapchain
image, presents once, and then blocks in `xcb_wait_for_event`. It does not use a
WebView, polling loop, runtime shader compiler, or bundled GPU stack.

This is a virtual-machine compatibility and idle-behavior result. Mesa
llvmpipe is a software Vulkan implementation, so its startup, resident memory,
thread count, and rendering cost are not representative of a desktop GPU.

## Environment

- Host: Apple M2 Pro, macOS 15.5 (24F74), arm64.
- Guest: QEMU Ubuntu 24.04 arm64, Linux 6.8.0-146-generic, glibc 2.39.
- Display: Xvfb, 1280x800x24.
- Vulkan: Mesa llvmpipe 25.2.8, Vulkan API 1.4.318.
- Go: 1.26.5 on the build host.
- Build: `CGO_ENABLED=0 -trimpath -ldflags='-s -w -buildid='`.
- Date: 2026-10-13.

## Results

- Linux XCB and Vulkan ABI tests pass in the arm64 guest.
- Ten first-frame samples range from 52.776 ms to 104.398 ms, with a median of
  60.260 ms. This includes window, instance, device, swapchain, clear, and first
  present; the VM/software-renderer result does not meet or refute the 50 ms
  physical-hardware target.
- After a 3 second settle, process CPU time did not advance during a 10 second
  sample: `0.000%` measured CPU.
- After 5 idle seconds and a graceful X11 close, diagnostics still report one
  GPU submission, zero resize events, and one Go goroutine.
- Settled RSS is 93,796 KiB with 13 OS threads and 6 file descriptors. llvmpipe
  owns most of this cost; a hardware Vulkan run is required before comparing
  Linux against the 15 MiB idle target.
- Stripped pulse sizes are 1,900,773 B for Linux arm64 and 1,908,965 B for
  Linux amd64, both below the provisional 4 MiB target.
- The ELF directly requires only `libdl.so.2`, `libpthread.so.0`, and
  `libc.so.6`; XCB and Vulkan remain dynamically selected system libraries.

Run `DISPLAY=:99 ./benchmarks/m0_linux.sh` on a Linux X11 system with Vulkan and
`xdotool` to rebuild the pulse, run platform tests, capture one startup sample,
measure settled CPU/RSS/thread/file-descriptor state, and close through the X11
window protocol.

## Current Boundary

- The original measurements above cover the M0 arm64 path. A 2026-10-15
  compatibility follow-up on Debian 12 amd64, Linux 5.15, X.Org 1.21.1.7,
  Go 1.26.5, and Mesa Lavapipe 22.3.6 exercised retained rounded rectangles,
  triangles, layered Pango/Cairo text, pointer updates, and custom chrome.
- Linux platform tests and the repository-wide test suite pass natively on that
  amd64 system. A two-click retained-scene probe changed the first captured
  frame and restored the second frame byte-for-byte; the process emitted no
  runtime error output.
- Mixed fixed-pitch text now places printable ASCII at its declared column
  origin, centers non-ASCII clusters only within their declared cell span, and
  shares one pixel-snapped baseline across styled runs. Pixel-level tests cover
  the column origin and vertical centering.
- With `CGO_ENABLED=0 -trimpath -ldflags='-s -w -buildid='`, the Linux amd64
  minimal pulse is 2,228,453 B both before and after the optional scene/text
  work. The retained scene/text example is 2,179,301 B. The latter directly
  links only `libdl`, `libpthread`, and `libc`; XCB, Vulkan, Pango, and Cairo are
  selected dynamically.
- On the same AMD EPYC 9554 host, three mixed Chinese/Latin samples measured
  23.675–27.038 µs and 724 B/32 allocations per Pango measurement, and
  136.611–144.131 µs and about 34.5 KiB/73 allocations per Cairo rasterization.
  Rasterization occurs only when retained label content or scale changes; it is
  not an unchanged-frame or idle path.
- One settled Lavapipe scene sample used 108,960 KiB RSS. This shared
  container/software-renderer observation is compatibility evidence, not an
  authoritative memory or latency benchmark.
- XCB is the first Linux window adapter. Wayland is not yet implemented, and no
  fallback or lowest-common-denominator abstraction has been added in advance.
- XIM/IBus composition, accessibility, fractional scaling, and close
  interception remain incomplete.
- Physical AMD, Intel, and NVIDIA Vulkan hardware measurements remain required.

---
title: Platforms
summary: How Viva is built on macOS, Windows, Linux and the browser.
order: 14
group: more
---

Viva keeps one public API over three native backends. Each backend uses the operating system's own windowing, GPU and text services, and native builds link system frameworks and the Go runtime.

## macOS

Cocoa, `CAMetalLayer` and Metal for drawing, with CoreText for text.

- Native text input, including Chinese input-method composition with candidate windows.
- A separate semantic tree mirrored into `NSAccessibility` elements.
- Status-area icons and popup panels, with the system's own `NSMenu` available for a status item.
- Packaging to `.app`, `.zip` and `.dmg`. Programs are compiled for macOS 12 by default.

## Windows

Win32 events, Direct3D 11 with embedded precompiled shaders, DirectWrite for measurement and bounded GDI textures for labels. Native builds do not load a shader compiler.

- Windows for amd64 and arm64.
- Native windowing, input and system text.
- Packaging to an NSIS installer or a `.zip`.

## Linux

X11 through dynamically loaded system XCB, Vulkan, Pango and Cairo, with a blocking event loop that does no idle redraw.

- X11 with the system's XCB and Vulkan libraries.
- Pango and Cairo for text.
- Packaging to a `tar.gz` with an install script, or a `.deb`.
- File dialogs, URL opening and notifications use installed desktop utilities; see [System integration](/en/docs/system-integration/).

## Browser (optional)

A secondary target: the same Go components compiled to WebAssembly and drawn with Canvas2D. The component catalog's previews use it.

- The browser build uses the same application, layout and component code as the native backends.
- Clipboard and file pickers use the browser's asynchronous APIs.
- Text editing is available in the Input example, and the same Go editor backs it as the native preview.
- WebGPU is a planned backend.

The browser backend is built separately and never makes a native artifact larger, slower or more complicated.

## Not supported

iOS, iPadOS, Android and other mobile systems are out of scope, and a touch-first runtime is not a goal. Desktop APIs are not weakened to leave a mobile door open.

# Linking `github.com/vivaui/viva/components`: Darwin arm64

Reference machine: Apple M2 Pro, macOS 15.5, Go 1.26.5, `-trimpath
-ldflags="-s -w -buildid="`, native build. Before is the committed tree; after
is that tree with the package-level icon initializers of `components` replaced
by composite literals. Linux and Windows were not measured.

Programs: **a** opens a window with `app` and `ui` only; **b** adds
`import _ "github.com/vivaui/viva/components"`; **c** instead calls `components.BeginButton(...).Default(...)`;
**hello** is `examples/guides/hello`, which uses `flex` and so links the package.

Executable file size is quantized: Mach-O segments are 16 KiB pages and the ad-hoc
signature has a hash per page, so a 32-byte change in `__rodata` that crosses a page
boundary shows as 16,512 bytes of file. The section sizes below are what changed.

```text
                           __text      __rodata    file bytes
before  a (baseline)      1,397,144     62,465     3,420,370
before  b (blank import)  1,406,216     62,625     3,436,898   b-a: +9,072 text, +160 rodata
after   b (blank import)  1,397,144     62,497     3,436,882   b-a:     0 text,  +32 rodata
before  c (one recipe)    1,414,472     62,657     3,453,426
after   c (one recipe)    1,405,400     62,529     3,436,914
before  hello             1,547,656     65,857     3,718,034
after   hello             1,538,584     65,697     3,701,506   -9,072 text, -160 rodata
```

After the change a blank import links no symbol of `github.com/vivaui/viva/components` (`go tool nm`
lists none, down from 24); the 32 bytes of `__rodata` are the package's entry in
the program's package tables. The 9 KB of code that was `github.com/vivaui/viva/components.init`
is gone from every program, including the ones that use `flex`.

Reproduce the symbol check with `go test ./components -run TestBlankImport`.

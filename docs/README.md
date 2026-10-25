# Documentation in this directory

These are **repository documents**, not user documentation. They record how the
framework is built, measured, and verified — they assume you have a checkout and
can run its programs.

| If you want to | Read |
| --- | --- |
| Use Viva in an application | the guides in [`website/content/guides`](../website/content/guides), or run the site: `cd website && npm run dev` |
| Look up an API | the API reference on that site, generated from the Go source and its comments |
| Build the framework itself | [../CONTRIBUTING.md](../CONTRIBUTING.md) and [../README.md](../README.md) |
| Check platform behaviour | [conformance.md](conformance.md) |
| See what was measured | [benchmarks/](benchmarks/) |

## What is here

- [conformance.md](conformance.md) — the scenarios that need a real window on a
  real platform, and the ones that still need eyes.
- [accessibility.md](accessibility.md) — the semantic tree and native
  accessibility.
- [system-integrations.md](system-integrations.md) — tray, menus, notifications,
  file drops, and the rest of the native surface.
- [initial-frame.md](initial-frame.md) — what the first frame does and does not
  do.
- [benchmarks/](benchmarks/) — every measurement, with the machine, the build
  mode, and the method recorded alongside the numbers.

## Writing a document here

A document that makes a claim about speed, size, or memory must also record the
reference machine, the operating system, the build mode, and how the number was
taken. Numbers without that context do not belong in this directory.

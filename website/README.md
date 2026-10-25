# viva documentation website

Astro owns the static documentation and routes; Svelte islands own interactive
controls. This development website is not part of the native UI or release-linked
runtime. Buttons, tabs and search inputs use source-owned shadcn-svelte components
from the official Vega registry. There are no remote fonts, scripts or analytics;
typography uses the operating system's sans-serif and monospace fonts.

Use Node 22.12+ and npm. From this directory:

```sh
npm ci
npm run dev
npm run check
npm test
npm run build
npm run test:site
npm run preview
```

`dev` and `build` run the Go generator first. Generated inputs are
`../build/docs-data/catalog.json` and `../build/docs-data/public/`; the static
output is `../build/docs-site/`. The development plugin is maintained separately
in `viva-preview.mjs`. Catalog data is read afresh for each route render rather
than cached at module scope. Package versions are pinned in the lockfile.
`check` runs both Astro and Svelte diagnostics; `test` includes the separately
maintained preview-plugin tests, interaction-policy tests and i18n completeness
tests. Run `npm run generate` before tests in a clean checkout. After building,
`test:site` checks both locales' rendered routes, links, flat examples and unique IDs.

## Data and URLs

`src/lib/types.ts` preserves the generator's PascalCase `Page`, `Nav`, `Demo`,
`Section`, and `Capability` fields, and lowercase catalog and coverage fields.
No extra generator fields are required. Page membership comes exclusively from
the generated catalog; the website only checks that page IDs are valid, unique
URL slugs. Examples also come exclusively from generated data. All examples are
stacked vertically with independent previews and reset buttons; the production
component source is expanded once below them.

- `/` redirects to `/zh/ui/button/`; `/zh/` and `/en/` open the corresponding Button page.
- `/zh/ui/<component>/` and `/en/ui/<component>/` document each generated page.
- `/zh/coverage/` and `/en/coverage/` list source components; pending entries are not page links.
- `/preview/index.html?component=<id>&example=<id>&theme=light|dark&source=<revision>` is
  the Go WASM runner; the source fingerprint versions the preview HTML request.
- `/sources/components/<id>.go` is the source-owned component recipe.
- `/sources/internal/componentdemo/<id>.go` is the shared Go example source, not
  a promise of an individual case file or the native command runner.
- `/coverage.json` and `/source-manifest.json` are served unchanged.

Revision labels mean **Go source fingerprint**, not a Git tag or a website asset
hash. Browser previews do not establish native platform support. All generated
platform details, section text and per-example web limitations remain visible.
The page never imitates a Go component using website CSS.

## Languages and documentation controls

`src/lib/i18n.ts` owns the supported locales, routes and typed message dictionaries.
Chinese is the default. The visible 中文 / EN links preserve the current component
or coverage page and work without JavaScript. The URL is the language authority;
there is no hidden browser-language redirect or server session.

Navigation, example titles/descriptions, guidance and interaction feedback are
localized. API names, shell commands, actual Go source and labels
drawn by the Go demo are intentionally unchanged. The first deeply documented
components use hand-written English copy; every additional source-owned recipe
receives bounded generic English guidance and explicit Chinese verification text,
so expanding the catalog cannot silently publish mixed-language prose. Tests cover
dictionary parity, all generated content and source preservation.

Official shadcn component source lives under `src/lib/components/ui/` with registry
URLs and original registry-response digests in `shadcn-sources.json`. `components.json`
records the registry configuration. Bits UI supplies keyboard/accessibility behavior;
Tailwind CSS, tailwind-variants and cn supply styling/composition. These dependencies
are confined to this website, not the Go runtime. The site uses a small local typed
dictionary rather than adding a client i18n framework.

## Interaction boundary

An IntersectionObserver activates each example's iframe when it approaches the
viewport and then disconnects. Already visited examples remain mounted until
leaving the page. Theme changes replace activated iframes; reset replaces only
its own example.
Messages
are accepted only from that exact active `contentWindow` and the page's exact
origin, with `viva-preview-ready` or `viva-preview-error` types. A page `load` event
is not readiness. One 60-second timeout permits explicit retry; there is no
polling or persistent heartbeat. Old readiness messages cannot revive a failed
or replaced iframe. Each example has an independent lifecycle and load timeout.

The native command uses its own generated example ID and appends `-dark` for
dark mode. Copy reports success or failure and
never executes a command. Source text is generated code, rendered as escaped
text (or build-time Shiki output), never assembled with untrusted HTML.

Search uses `/` or Ctrl/Command+K. Arrow keys move through filtered navigation.
Escape clears search or closes the mobile
navigation disclosure. There is no keyboard trap. Themes are stored locally;
storage denial leaves the toggle functional. Without JavaScript, documentation,
navigation, production source, coverage and native commands remain readable, but
interactive controls and WASM previews require JavaScript.

All production URLs are local. Serve the complete built directory with its
generated WASM and source assets to work without an internet connection.
`file://` is not a supported execution environment for the WASM runner.

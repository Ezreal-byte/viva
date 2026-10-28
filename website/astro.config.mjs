import { defineConfig } from 'astro/config';
import svelte from '@astrojs/svelte';
import vivaPreview from './viva-preview.mjs';
import tailwindcss from '@tailwindcss/vite';
import { fileURLToPath } from 'node:url';
import { writeFile } from 'node:fs/promises';
import { locales } from './src/lib/i18n.ts';

// Cloudflare Pages answers this file before it looks at the assets, so the root
// is a redirect at the edge instead of the meta-refresh document that
// src/pages/index.astro emits for hosts that ignore it. publicDir belongs to the
// generator, so the file is written into the build output here.
const pagesRedirects = () => ({
  name: 'pages-redirects',
  hooks: {
    'astro:build:done': async ({ dir }) => {
      const [fallback] = locales;
      await writeFile(new URL('_redirects', dir), `/  /${fallback}/  302\n`);
    },
  },
});

export default defineConfig({
  output: 'static',
  trailingSlash: 'always',
  // Readers move between documentation pages constantly; a full document load
  // per click is what makes the site feel slower than a client-routed one.
  // Hover prefetching and the client router in BaseLayout turn a click into a
  // DOM swap instead of a new page load.
  prefetch: { prefetchAll: true, defaultStrategy: 'hover' },
  publicDir: '../build/docs-data/public',
  outDir: '../build/docs-site',
  markdown: { shikiConfig: { themes: { light: 'github-light', dark: 'github-dark' }, defaultColor: false } },
  integrations: [svelte(), pagesRedirects()],
  devToolbar: { enabled: false },
  vite: {
    plugins: [vivaPreview(), tailwindcss()],
    resolve: { alias: { $lib: fileURLToPath(new URL('./src/lib', import.meta.url)) } },
  },
});

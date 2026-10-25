import { defineConfig } from 'astro/config';
import svelte from '@astrojs/svelte';
import vivaPreview from './viva-preview.mjs';
import tailwindcss from '@tailwindcss/vite';
import { fileURLToPath } from 'node:url';

export default defineConfig({
  output: 'static',
  trailingSlash: 'always',
  publicDir: '../build/docs-data/public',
  outDir: '../build/docs-site',
  markdown: { shikiConfig: { themes: { light: 'github-light', dark: 'github-dark' }, defaultColor: false } },
  integrations: [svelte()],
  devToolbar: { enabled: false },
  vite: {
    plugins: [vivaPreview(), tailwindcss()],
    resolve: { alias: { $lib: fileURLToPath(new URL('./src/lib', import.meta.url)) } },
  },
});

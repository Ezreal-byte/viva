import assert from 'node:assert/strict';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { setTimeout as pause } from 'node:timers/promises';
import { createPreviewRebuilder, isPreviewSource, previewAssetPath } from './viva-preview.mjs';

test('only Go preview inputs trigger regeneration', () => {
  for (const filename of ['components/button.go', 'internal/componentdemo/input.go', 'tools/docs/content.go', 'go.mod', 'cmd/viva/templates/input.go.txt']) {
    assert.equal(isPreviewSource(fileURLToPath(new URL(`../${filename}`, import.meta.url))), true);
  }
  for (const filename of ['components/button_test.go', 'website/src/pages/index.astro', 'build/docs-data/catalog.json', 'components-other/button.go']) {
    assert.equal(isPreviewSource(fileURLToPath(new URL(`../${filename}`, import.meta.url))), false);
  }
});

test('preview assets resolve inside the current generated directory only', () => {
  const directory = fileURLToPath(new URL('../build/docs-data/public/preview/', import.meta.url));
  assert.equal(previewAssetPath('/preview/preview-next.wasm?source=next', directory), fileURLToPath(new URL('../build/docs-data/public/preview/preview-next.wasm', import.meta.url)));
  assert.equal(previewAssetPath('/preview/', directory), null);
  assert.equal(previewAssetPath('/sources/components/button.go', directory), null);
  assert.equal(previewAssetPath('/preview/%2e%2e/catalog.json', directory), null);
  assert.equal(previewAssetPath('/preview/%2Fetc%2Fpasswd', directory), null);
});

test('source changes coalesce and a running build never overlaps its replacement', async () => {
  let builds = 0;
  let reloads = 0;
  const completions = [];
  const rebuilder = createPreviewRebuilder({ delay: 1, rebuild: () => { builds++; return new Promise(resolve => completions.push(resolve)); }, reload: () => reloads++, reportError: assert.fail });
  try {
    rebuilder.changed();
    rebuilder.changed();
    await pause(20);
    assert.equal(builds, 1);
    rebuilder.changed();
    await pause(20);
    assert.equal(builds, 1);
    completions.shift()();
    await pause(20);
    assert.equal(builds, 2);
    assert.equal(reloads, 0);
    completions.shift()();
    await pause(20);
    assert.equal(reloads, 1);
  } finally { rebuilder.close(); }
});

test('failure does not reload or retry until another source edit', async () => {
  let errors = 0;
  let builds = 0;
  let activeSignal;
  const rebuilder = createPreviewRebuilder({ delay: 1, rebuild: async signal => { activeSignal = signal; builds++; throw new Error('compile error'); }, reload: assert.fail, reportError: () => errors++ });
  rebuilder.changed();
  await pause(20);
  assert.equal(errors, 1);
  assert.equal(builds, 1);
  rebuilder.close();
  assert.equal(activeSignal.aborted, true);
  rebuilder.changed();
  await pause(20);
  assert.equal(builds, 1);
});

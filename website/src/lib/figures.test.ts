import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { figures, homepageTiles, tiles } from './figures.ts';
import { home } from './home.ts';

const docs = new URL('../../../docs/', import.meta.url);
const read = (path: string) => readFileSync(new URL(path, docs), 'utf8');

test('every number on the home page is still stated by the report it quotes', () => {
  for (const figure of figures) {
    assert.ok(read(figure.document).includes(figure.evidence), `${figure.document} no longer contains "${figure.evidence}"`);
  }
});

test('a figure that the report calls a miss is shown as one, and nothing else is', () => {
  const m0 = read('benchmarks/m0-darwin-arm64.md');
  for (const figure of figures) {
    const row = m0.split('\n').find(line => line.includes(figure.evidence));
    if (row?.includes('| Miss |')) assert.ok(figure.miss, `${figure.id} is a miss in its report`);
    if (row?.includes('| Pass |')) assert.ok(!figure.miss, `${figure.id} passes in its report`);
  }
});

test('every figure appears in exactly one tile, and tiles agree with their figures', () => {
  const used = tiles.flatMap(tile => tile.figures);
  assert.deepEqual([...used].sort(), figures.map(figure => figure.id).sort());
  for (const tile of tiles) {
    for (const id of tile.figures) assert.equal(figures.find(figure => figure.id === id)?.miss, tile.miss);
  }
});

test('both languages describe exactly the tiles the page shows', () => {
  for (const locale of ['zh', 'en'] as const) {
    assert.deepEqual(Object.keys(home[locale].measures).sort(), tiles.map(tile => tile.id).sort());
    for (const measure of Object.values(home[locale].measures)) {
      assert.ok(measure.label.trim() && measure.value.trim() && measure.detail.trim());
    }
  }
});

test('the landing page shows no figure its report calls a miss', () => {
  for (const tile of homepageTiles) {
    assert.ok(!tile.miss, `${tile.id} is a miss and belongs in the performance guide, not the landing page`);
  }
});

test('the performance guide quotes the same numbers in both languages', () => {
  for (const locale of ['zh', 'en']) {
    const guide = readFileSync(new URL(`../../content/guides/${locale}/performance.md`, import.meta.url), 'utf8');
    for (const id of ['binary', 'footprint', 'firstFrame', 'unchanged', 'update']) {
      const figure = figures.find(entry => entry.id === id)!;
      const quoted = figure.evidence.replace(' after initial present', '').replace(/ ns\/op$/, ' ns');
      assert.ok(guide.includes(quoted), `${locale}/performance.md is missing ${quoted}`);
    }
  }
});

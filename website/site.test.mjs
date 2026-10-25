import assert from 'node:assert/strict';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import test from 'node:test';

const catalog = JSON.parse(readFileSync(new URL('../build/docs-data/catalog.json', import.meta.url), 'utf8'));
const output = new URL('../build/docs-site/', import.meta.url);
const readPage = path => readFileSync(new URL(`${path}index.html`, output), 'utf8');

const api = JSON.parse(readFileSync(new URL('../build/docs-data/api.json', import.meta.url), 'utf8'));
const guideDirectory = new URL('./content/guides/en/', import.meta.url);
const guideSlugs = readdirSync(guideDirectory).filter(name => name.endsWith('.md')).map(name => name.slice(0, -3));
const lang = locale => (locale === 'zh' ? 'zh-CN' : 'en');

test('the root redirects to the Chinese home page', () => {
  assert.ok(readPage('').includes('/zh/'));
});

for (const locale of ['zh', 'en']) {
  test(`${locale} home page states its numbers, its misses and its platforms`, () => {
    const html = readPage(`${locale}/`);
    assert.ok(html.includes(`<html lang="${lang(locale)}"`));
    for (const language of ['zh', 'en']) assert.ok(html.includes(`href="/${language}/"`));
    for (const text of ['1,697,506', '26.3–26.8 MiB', '123–374 ms', 'macOS', 'Windows', 'Linux']) assert.ok(html.includes(text), `missing ${text}`);
    assert.ok(html.includes(`href="/${locale}/docs/getting-started/"`));
    assert.ok(html.includes('class="preview-canvas"'), 'the live component preview is missing');
    assert.equal([...html.matchAll(/<h1[ >]/g)].length, 1);
  });

  test(`${locale} documentation has an index and every guide`, () => {
    const index = readPage(`${locale}/docs/`);
    for (const slug of guideSlugs) {
      assert.ok(index.includes(`href="/${locale}/docs/${slug}/"`), `index lacks ${slug}`);
      const html = readPage(`${locale}/docs/${slug}/`);
      assert.ok(html.includes(`<html lang="${lang(locale)}"`));
      for (const language of ['zh', 'en']) assert.ok(html.includes(`href="/${language}/docs/${slug}/"`), `${slug}: no ${language} switch`);
      assert.equal([...html.matchAll(/<h1[ >]/g)].length, 1, `${slug}: one h1`);
    }
  });

  test(`${locale} API reference covers every package and type`, () => {
    const index = readPage(`${locale}/api/`);
    for (const pkg of api.packages) {
      assert.ok(index.includes(`href="/${locale}/api/${pkg.name}/"`), `index lacks ${pkg.name}`);
      const page = readPage(`${locale}/api/${pkg.name}/`);
      for (const type of pkg.types) {
        assert.ok(page.includes(`href="/${locale}/api/${pkg.name}/${type.name}/"`), `${pkg.name} lacks a link to ${type.name}`);
        assert.ok(existsSync(new URL(`${locale}/api/${pkg.name}/${type.name}/index.html`, output)));
      }
    }
  });

}

function htmlFiles(directory) {
  const found = [];
  for (const entry of readdirSync(directory)) {
    const path = new URL(`${entry}${statSync(new URL(entry, directory)).isDirectory() ? '/' : ''}`, directory);
    if (statSync(path).isDirectory()) found.push(...htmlFiles(path));
    else if (entry.endsWith('.html')) found.push(path);
  }
  return found;
}

test('every site-local link and image on every page resolves, including its #anchor', () => {
  const cache = new Map();
  const ids = path => {
    if (!cache.has(path)) cache.set(path, new Set([...readFileSync(new URL(path, output), 'utf8').matchAll(/\sid="([^"]+)"/g)].map(match => match[1])));
    return cache.get(path);
  };
  let checked = 0;
  for (const file of htmlFiles(new URL('zh/', output)).concat(htmlFiles(new URL('en/', output)))) {
    const html = readFileSync(file, 'utf8');
    const source = file.pathname.slice(output.pathname.length);
    for (const match of html.matchAll(/(?:href|src)="(\/[^"]*)"/g)) {
      const [target, anchor] = match[1].split('#');
      const path = target.split('?')[0].slice(1);
      const resolved = path.endsWith('/') || path === '' ? `${path}index.html` : path;
      assert.ok(existsSync(new URL(resolved, output)), `${source}: missing ${match[1]}`);
      if (anchor && resolved.endsWith('.html')) assert.ok(ids(resolved).has(decodeURIComponent(anchor)), `${source}: no #${anchor} in ${resolved}`);
      checked++;
    }
  }
  assert.ok(checked > 10000);
});

test('no page repeats an id', () => {
  for (const file of htmlFiles(new URL('zh/', output)).concat(htmlFiles(new URL('en/', output)))) {
    const ids = [...readFileSync(file, 'utf8').matchAll(/\sid="([^"]+)"/g)].map(match => match[1]);
    const repeated = ids.filter((id, index) => ids.indexOf(id) !== index);
    assert.deepEqual(repeated, [], `${file.pathname.slice(output.pathname.length)} repeats ${repeated.slice(0, 3)}`);
  }
});

for (const locale of ['zh', 'en']) {
  for (const page of catalog.pages) {
    test(`${locale}/${page.ID} has localized links, every example and unique accessible IDs`, () => {
      const html = readPage(`${locale}/ui/${page.ID}/`);
      assert.ok(html.includes(`<html lang="${locale === 'zh' ? 'zh-CN' : 'en'}"`));
      for (const language of ['zh', 'en']) assert.ok(html.includes(`href="/${language}/ui/${page.ID}/"`));
      assert.doesNotMatch(html, /<select\b/);
      assert.equal([...html.matchAll(/class="example-section"/g)].length, page.Examples.length);
      assert.equal([...html.matchAll(/class="example-demo"/g)].length, page.Examples.length);
      assert.equal([...html.matchAll(/class="code-block usage-block"/g)].length, page.Examples.length);
      for (const example of page.Examples) {
        assert.ok(html.includes(`id="example-${example.ID}"`));
      }
      const ids = [...html.matchAll(/\sid="([^"]+)"/g)].map(match => match[1]);
      assert.equal(new Set(ids).size, ids.length);
      for (const match of html.matchAll(/href="(\/[^"?#]+)"/g)) {
        const path = match[1].slice(1);
        assert.ok(existsSync(new URL(path.endsWith('/') ? `${path}index.html` : path, output)), `Missing link: ${path}`);
      }
    });
  }

  test(`${locale} components index links every component and switches language`, () => {
    const html = readPage(`${locale}/components/`);
    assert.ok(html.includes(`<html lang="${lang(locale)}"`));
    for (const language of ['zh', 'en']) assert.ok(html.includes(`href="/${language}/components/"`));
    for (const page of catalog.pages) assert.ok(html.includes(`href="/${locale}/ui/${page.ID}/"`), `index lacks ${page.ID}`);
    assert.ok(html.includes('class="component-index-grid"'));
    assert.ok(html.includes(locale === 'zh' ? '按钮' : 'Button'));
  });
}

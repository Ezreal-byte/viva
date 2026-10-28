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
  // Pages answers this at the edge; the page with the meta refresh is only a
  // fallback for hosts that ignore _redirects.
  const redirects = readFileSync(new URL('./_redirects', output), 'utf8');
  assert.match(redirects, /^\/\s+\/zh\/\s+302$/m, 'no edge redirect for the root');
  assert.ok(readPage('').includes('/zh/'));
});

test('navigation is client-routed and the stored theme survives it', () => {
  const html = readPage('zh/docs/getting-started/');
  // Without the client router every click loads a new document: measured at
  // 0.3-1.1 s per move against about 30 ms for a DOM swap on the same machine.
  assert.ok(html.includes('ClientRouter'), 'no client router script');
  // The router swaps in the new page's markup, which carries data-theme="light",
  // so a dark-theme reader would see the page flash white without this hook.
  assert.ok(html.includes('astro:after-swap'), 'no theme restore after a swap');
});

for (const locale of ['zh', 'en']) {
  test(`${locale} header offers a site-wide search`, () => {
    const html = readPage(`${locale}/`);
    assert.ok(html.includes('search-trigger'), 'no search trigger in the header');
    assert.ok(html.includes('SearchDialog'), 'the search dialog island is not in the header');
  });

  test(`${locale} search index covers components, guides and API`, () => {
    const index = JSON.parse(readFileSync(new URL(`./search/${locale}.json`, output), 'utf8'));
    const kinds = [...new Set(index.entries.map(entry => entry.kind))].sort();
    assert.deepEqual(kinds, ['api', 'component', 'guide']);
    assert.ok(index.entries.some(entry => entry.kind === 'component' && entry.href === `/${locale}/ui/button/`), 'no Button component');
    assert.ok(index.entries.some(entry => entry.kind === 'guide' && entry.href === `/${locale}/docs/getting-started/`), 'no getting-started guide');
    for (const entry of index.entries) assert.match(entry.href, /^\/[a-z]{2}\//, `bad href ${entry.href}`);
  });
}

test('unknown paths fail with a 404 page instead of the root redirect', () => {
  // Pages treats a project without a top-level 404.html as a single-page
  // application: every unmatched path answers with the root document and a 200,
  // so a broken link looks like a working redirect. Astro emits this file.
  assert.ok(existsSync(new URL('404.html', output)), 'no 404.html for unknown paths');
  const html = readFileSync(new URL('404.html', output), 'utf8');
  assert.ok(html.includes('<html lang="zh-CN"'));
  assert.ok(html.includes('404'));
  assert.ok(!html.includes('http-equiv="refresh"'), 'the 404 page is the root redirect');
  for (const language of ['zh', 'en']) {
    assert.ok(html.includes(`href="/${language}/"`), `404 page has no ${language} home link`);
    assert.ok(html.includes(`href="/${language}/components/"`), `404 page has no ${language} catalog link`);
  }
});

test('theme tokens stay namespaced so an extension cannot repaint the page', () => {
  // A browser extension that themes scrollbars declares its own
  // `:root { --background: url(...) }`. Extension styles load after the page's,
  // both selectors are `:root` at the same specificity, so a bare `--background`
  // let it replace the whole page background with its artwork. Every token this
  // site declares carries the product prefix for that reason, and these names
  // are the ones a page-level stylesheet is most likely to collide with.
  const cssDirectory = new URL('_astro/', output);
  const css = readdirSync(cssDirectory)
    .filter(name => name.endsWith('.css'))
    .map(name => readFileSync(new URL(name, cssDirectory), 'utf8'))
    .join('\n');
  assert.ok(css.includes('--viva-background:'), 'no namespaced page background token');
  for (const name of ['background', 'text', 'border', 'surface', 'surface-hover', 'muted', 'radius', 'strong-border', 'doc-secondary', 'brand', 'accent', 'accent-soft', 'error', 'success', 'mono', 'header-height']) {
    assert.doesNotMatch(css, new RegExp(`(?<![\\w-])--${name}\\s*:`), `the bare token --${name} is declared again`);
    assert.ok(!css.includes(`var(--${name})`), `something still reads the bare token --${name}`);
  }
});

for (const locale of ['zh', 'en']) {
  test(`${locale} home page states its numbers, its platforms and where the rest lives`, () => {
    const html = readPage(`${locale}/`);
    assert.ok(html.includes(`<html lang="${lang(locale)}"`));
    for (const language of ['zh', 'en']) assert.ok(html.includes(`href="/${language}/"`));
    for (const text of ['1,697,506', 'macOS', 'Windows', 'Linux']) assert.ok(html.includes(text), `missing ${text}`);
    assert.ok(html.includes(`href="/${locale}/docs/getting-started/"`));
    // The count is derived from the catalog, so a hardcoded number or a missing
    // stat fails here instead of drifting.
    assert.ok(html.includes(`<dd>${catalog.pages.length}</dd>`), 'the component count is missing');
    // The figures that have not reached their report's target live in the
    // performance guide; the landing page still has to lead a reader to them.
    assert.ok(html.includes(`href="/${locale}/docs/performance/"`), 'no link to the measurements that are not on the page');
    assert.ok(!html.includes('26.3–26.8 MiB'), 'a performance-guide figure is on the landing page');
    assert.equal([...html.matchAll(/<h1[ >]/g)].length, 1);
  });

  test(`${locale} footer carries the site map, the repository and the licence`, () => {
    const footer = readPage(`${locale}/`).slice(readPage(`${locale}/`).indexOf('class="site-footer"'));
    assert.ok(footer.includes('https://github.com/vivaui/viva'), 'no repository link');
    assert.ok(footer.includes('blob/main/LICENSE'), 'no licence link');
    assert.ok(footer.includes(`href="/${locale}/docs/"`), 'no guides link');
    assert.ok(footer.includes(`href="/${locale}/components/"`), 'no catalog link');
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

  test(`${locale} catalog index hands the sidebar navigation, not page bodies`, () => {
    const html = readPage(`${locale}/components/`);
    // The sidebar island needs stable IDs, names and categories. Passing whole
    // pages duplicated every example, section and capability into the HTML, and
    // each page also embeds the full component list, so the index shipped a
    // megabyte of payload that the browser had to parse to show its navigation.
    assert.equal([...html.matchAll(/&quot;Category&quot;/g)].length, catalog.pages.length);
    assert.ok(html.includes(`&quot;${catalog.pages[0].ID}&quot;`));
    for (const field of ['Summary', 'Examples', 'Sections', 'Capabilities', 'InstallCommand']) {
      assert.ok(!html.includes(`&quot;${field}&quot;`), `sidebar island receives ${field}`);
    }
  });
}

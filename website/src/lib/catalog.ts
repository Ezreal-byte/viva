import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import type { Catalog, Nav, Page } from './types';

export function readCatalog(): Catalog {
  const filename = resolve(process.cwd(), '../build/docs-data/catalog.json');
  let catalog: Catalog;
  try {
    catalog = JSON.parse(readFileSync(filename, 'utf8')) as Catalog;
  } catch (error) {
    throw new Error(`Cannot read ${filename}. Run npm run generate first.`, { cause: error });
  }
  if (!Array.isArray(catalog.pages) || !catalog.pages.length || !Array.isArray(catalog.coverage)) {
    throw new Error('The generated catalog must contain nonempty pages and a coverage array.');
  }
  const seen = new Set<string>();
  for (const page of catalog.pages) {
    if (typeof page.ID !== 'string' || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(page.ID) || seen.has(page.ID)) {
      throw new Error('Generated page IDs must be nonempty, unique, lowercase URL slugs.');
    }
    seen.add(page.ID);
  }
  return catalog;
}

// The sidebar island needs only stable IDs, names and categories. Handing it
// whole pages serialized every example, section and capability into the
// island's props, and each page embeds the full component list, so the catalog
// index shipped a megabyte of duplicated payload. Keep the island input flat.
export function navigationFor(catalog: Catalog, page?: Page): Nav[] {
  const documented = new Set(catalog.pages.map((entry) => entry.ID));
  const entries: Nav[] = page ? page.All : catalog.pages;
  return entries
    .filter((entry) => documented.has(entry.ID))
    .map(({ ID, Title, Category }) => ({ ID, Title, Category }));
}

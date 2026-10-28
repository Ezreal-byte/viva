import { getCollection } from 'astro:content';
import { readAPI } from './api';
import { readCatalog } from './catalog';
import { guideSlug } from './content';
import { categoryLabel, componentLabel, componentPath, type Locale } from './i18n';
import { apiPath, docsPath } from './routes';

export type SearchKind = 'component' | 'guide' | 'api';

export interface SearchEntry {
  title: string;
  sub: string;
  href: string;
  kind: SearchKind;
}

// One flat list per locale, built at build time: components from the generated
// catalog, guides from their front matter, API packages and types from the
// generated api.json. The page fetches it the first time the dialog opens.
export async function searchIndex(locale: Locale): Promise<SearchEntry[]> {
  const components: SearchEntry[] = readCatalog().pages.map(page => ({
    title: componentLabel(locale, page.ID, page.Title),
    sub: categoryLabel(locale, page.Category),
    href: componentPath(locale, page.ID),
    kind: 'component',
  }));

  const guides: SearchEntry[] = (await getCollection('guides'))
    .filter(entry => entry.id.startsWith(`${locale}/`))
    .sort((left, right) => left.data.order - right.data.order)
    .map(entry => ({
      title: entry.data.title,
      sub: entry.data.summary,
      href: docsPath(locale, guideSlug(entry)),
      kind: 'guide',
    }));

  const api: SearchEntry[] = readAPI().packages.flatMap(pkg => [
    { title: pkg.name, sub: pkg.synopsis, href: apiPath(locale, pkg.name), kind: 'api' as const },
    ...pkg.types.map(type => ({
      title: type.name,
      sub: `${pkg.name} · ${type.synopsis}`,
      href: apiPath(locale, pkg.name, type.name),
      kind: 'api' as const,
    })),
  ]);

  return [...components, ...guides, ...api];
}

import { getCollection, type CollectionEntry } from 'astro:content';
import type { Locale } from './i18n';
import { siteText } from './site';
import { docsPath, type NavGroup } from './routes';

export type Guide = CollectionEntry<'guides'>;

export function guideSlug(entry: Guide): string {
  return entry.id.split('/')[1];
}

export async function guidesFor(locale: Locale): Promise<Guide[]> {
  const all = await getCollection('guides');
  return all.filter(entry => entry.id.startsWith(`${locale}/`)).sort((left, right) => left.data.order - right.data.order);
}

const groupOrder = ['start', 'concepts', 'guides', 'distribute', 'more'] as const;

// The documentation sidebar: guides grouped the way a reader moves through them.
export function guideNavigation(guides: Guide[], locale: Locale, currentID?: string): NavGroup[] {
  const text = siteText(locale);
  const headings = { start: text.groupStart, concepts: text.groupConcepts, guides: text.groupGuides, distribute: text.groupDistribute, more: text.groupMore };
  return groupOrder
    .map(group => ({
      heading: headings[group],
      links: guides.filter(entry => entry.data.group === group).map(entry => ({ href: docsPath(locale, guideSlug(entry)), label: entry.data.title, current: entry.id === currentID })),
    }))
    .filter(group => group.links.length > 0);
}

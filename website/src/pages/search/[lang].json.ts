import type { APIRoute } from 'astro';
import { isLocale, locales } from '../../lib/i18n';
import { searchIndex } from '../../lib/search';

export function getStaticPaths() {
  return locales.map(lang => ({ params: { lang } }));
}

// A file per locale, generated once: the header dialog downloads it the first
// time it opens instead of shipping the whole index on every page.
export const GET: APIRoute = async ({ params }) => {
  if (!isLocale(params.lang)) return new Response('Not found', { status: 404 });
  return new Response(JSON.stringify({ entries: await searchIndex(params.lang) }), {
    headers: { 'content-type': 'application/json; charset=utf-8' },
  });
};

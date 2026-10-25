import { readFileSync, statSync } from 'node:fs';
import { resolve } from 'node:path';
import type { Locale } from './i18n';
import { apiPath } from './routes';

export interface DocInline { text: string; italic?: boolean; code?: boolean; href?: string }
export interface DocBlock { kind: 'paragraph' | 'heading' | 'code' | 'list'; text?: DocInline[]; items?: DocBlock[][]; code?: string; ordered?: boolean }
export interface APIValue { names: string[]; decl: string; doc: DocBlock[] }
export interface APIFunc { name: string; receiver?: string; decl: string; synopsis: string; doc: DocBlock[] }
export interface APIType { name: string; decl: string; synopsis: string; doc: DocBlock[]; constants: APIValue[]; variables: APIValue[]; constructors: APIFunc[]; methods: APIFunc[] }
export interface APIPackage { name: string; importPath: string; synopsis: string; doc: DocBlock[]; constants: APIValue[]; variables: APIValue[]; functions: APIFunc[]; types: APIType[] }
export interface APIData { packages: APIPackage[] }

// api.json is parsed once per build, and again whenever the generator rewrites it,
// so a long static build does not parse megabytes of JSON once per page.
let cache: { modified: number; data: APIData } | undefined;
export function readAPI(): APIData {
  const filename = resolve(process.cwd(), '../build/docs-data/api.json');
  let modified: number;
  try {
    modified = statSync(filename).mtimeMs;
  } catch (error) {
    throw new Error(`Cannot read ${filename}. Run npm run generate first.`, { cause: error });
  }
  if (cache?.modified !== modified) {
    const data = JSON.parse(readFileSync(filename, 'utf8')) as APIData;
    if (!Array.isArray(data.packages) || !data.packages.length) throw new Error('The generated API data must contain packages.');
    cache = { modified, data };
  }
  return cache.data;
}

export function packageNamed(data: APIData, name: string): APIPackage {
  const found = data.packages.find(entry => entry.name === name);
  if (!found) throw new Error(`Unknown API package: ${name}`);
  return found;
}

// A generated link such as "ui/Canvas#method-Hovered" or "app#func-RunCanvas"
// becomes a route under /<locale>/api/. External URLs pass through unchanged.
export function apiHref(locale: Locale, href: string): string {
  if (/^[a-z]+:\/\//i.test(href)) return href;
  const [page, anchor] = href.split('#');
  return apiPath(locale, ...page.split('/')) + (anchor ? `#${anchor}` : '');
}

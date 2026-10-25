import type { Locale } from './i18n';

// Every route the website serves, in one place. The URL is the language authority.
export const homePath = (locale: Locale): string => `/${locale}/`;
export const docsPath = (locale: Locale, slug?: string): string => slug ? `/${locale}/docs/${slug}/` : `/${locale}/docs/`;

export type Section = 'home' | 'docs' | 'components' | 'api';

export const apiPath = (locale: Locale, ...parts: string[]): string => `/${locale}/api/${parts.map(part => `${part}/`).join('')}`;

export interface NavLink {
  href: string;
  label: string;
  current?: boolean;
}

export interface NavGroup {
  heading: string;
  links: NavLink[];
}

export interface Heading {
  depth: number;
  slug: string;
  text: string;
}

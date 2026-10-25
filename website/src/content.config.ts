import { defineCollection } from 'astro:content';
import { glob } from 'astro/loaders';
import { z } from 'astro/zod';

// Guides are written for the website, once per language. The website holds no
// second copy of them that could drift from the guides' source files.
const guides = defineCollection({
  loader: glob({ pattern: '*/*.md', base: './content/guides' }),
  schema: z.object({
    title: z.string(),
    summary: z.string(),
    order: z.number().int().positive(),
    group: z.enum(['start', 'concepts', 'guides', 'distribute', 'more']),
  }),
});

export const collections = { guides };

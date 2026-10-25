// Fills the Go code blocks of the guides from examples/guides/, so the code a guide
// shows is the code `go test ./examples/...` runs.
//
// A guide marks a block with `<!-- example: hello/main.go -->` followed by a go fence,
// or writes `{{example:hello/main.go}}` and lets this script expand it. The website
// tests fail if a marked block differs from its file; run `npm run sync:examples`.
import { readFileSync, readdirSync, writeFileSync } from 'node:fs';

const examples = new URL('../examples/guides/', import.meta.url);
const guides = new URL('./content/guides/', import.meta.url);

export function block(path) {
  const source = readFileSync(new URL(path, examples), 'utf8').replace(/\n+$/, '');
  return `<!-- example: ${path} -->\n\`\`\`go\n${source}\n\`\`\``;
}

export function expand(markdown) {
  return markdown
    .replace(/\{\{example:([^}]+)\}\}/g, (_, path) => block(path.trim()))
    .replace(/<!-- example: (\S+) -->\n```go\n[\s\S]*?\n```/g, (_, path) => block(path));
}

if (import.meta.url === `file://${process.argv[1]}`) {
  for (const locale of readdirSync(guides)) {
    for (const name of readdirSync(new URL(`${locale}/`, guides)).filter(file => file.endsWith('.md'))) {
      const file = new URL(`${locale}/${name}`, guides);
      const before = readFileSync(file, 'utf8');
      const after = expand(before);
      if (after !== before) writeFileSync(file, after);
    }
  }
}

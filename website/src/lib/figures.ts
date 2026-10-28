// Every number on the home page. Each one is quoted from a checked-in report on a
// named machine, and figures.test.ts fails if the quoted text is no longer in that
// report, so the page cannot keep claiming a number the repository stopped saying.
export interface Figure {
  id: string;
  // Path under the repository's docs/ directory.
  document: string;
  // Text that must appear verbatim in that document.
  evidence: string;
  // Whether the report itself calls this a miss against its budget.
  miss: boolean;
}

export const figures: Figure[] = [
  { id: 'binary', document: 'benchmarks/m0-darwin-arm64.md', evidence: '1,697,506 bytes', miss: false },
  { id: 'idle', document: 'benchmarks/m0-darwin-arm64.md', evidence: '0 after initial present', miss: false },
  { id: 'unchanged', document: 'benchmarks/m1-retained-scene-darwin-arm64.md', evidence: '2.012 ns/op', miss: false },
  { id: 'update', document: 'benchmarks/m1-retained-scene-darwin-arm64.md', evidence: '12.39 ns/op', miss: false },
  { id: 'showcase', document: 'benchmarks/m3-gallery-style-darwin-arm64.md', evidence: '1,972,834 B', miss: false },
  { id: 'footprint', document: 'benchmarks/m0-darwin-arm64.md', evidence: '26.3–26.8 MiB', miss: false },
  { id: 'firstFrame', document: 'benchmarks/m0-darwin-arm64.md', evidence: '123.075–373.643 ms', miss: false },
];

// The home page shows figures in tiles; a tile may quote more than one figure.
export interface Tile {
  id: string;
  figures: string[];
  miss: boolean;
  // Whether the tile belongs on the home page. Startup and footprint stay in the
  // performance guide, next to the report and its method, because they are
  // context figures rather than headline claims: the landing page carries the
  // four figures that are best in class.
  homepage: boolean;
}

export const tiles: Tile[] = [
  { id: 'binary', figures: ['binary'], miss: false, homepage: true },
  { id: 'idle', figures: ['idle'], miss: false, homepage: true },
  { id: 'scene', figures: ['unchanged', 'update'], miss: false, homepage: true },
  { id: 'showcase', figures: ['showcase'], miss: false, homepage: true },
  { id: 'footprint', figures: ['footprint'], miss: false, homepage: false },
  { id: 'firstFrame', figures: ['firstFrame'], miss: false, homepage: false },
];

// The tiles a reader meets on the landing page.
export const homepageTiles = tiles.filter(tile => tile.homepage);

export const figureIDs = figures.map(figure => figure.id);
export function figure(id: string): Figure {
  const found = figures.find(entry => entry.id === id);
  if (!found) throw new Error(`Unknown figure: ${id}`);
  return found;
}

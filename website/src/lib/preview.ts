import type { Nav, PreviewState, PreviewUpdate, Theme } from './types.ts';

export const previewTimeout = 60_000;

export function previewPath(component: string, example: string, theme: Theme, revision: string, locale: 'zh' | 'en'): string {
  const parameters = new URLSearchParams({ component, example, theme, source: revision, locale });
  return `/preview/index.html?${parameters.toString()}`;
}

export function filterNavigation(items: Nav[], query: string): Nav[] {
  const words = query.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean);
  return items.filter((item) => {
    const searchable = `${item.ID} ${item.Title} ${item.Category}`.toLocaleLowerCase();
    return words.every((word) => searchable.includes(word));
  });
}

export interface PreviewWheel {
  deltaX: number;
  deltaY: number;
}

// previewWheel accepts an unused wheel gesture from the active preview so the
// page can scroll even while the pointer is over the iframe.
export function previewWheel(
  event: Pick<MessageEvent, 'origin' | 'source' | 'data'>,
  origin: string,
  activeSource: MessageEventSource | null,
): PreviewWheel | null {
  if (!activeSource || event.origin !== origin || event.source !== activeSource) return null;
  if (!event.data || typeof event.data !== 'object' || Array.isArray(event.data)) return null;
  if (event.data.type !== 'viva-preview-wheel') return null;
  const deltaX = Number(event.data.deltaX);
  const deltaY = Number(event.data.deltaY);
  if (!Number.isFinite(deltaX) || !Number.isFinite(deltaY)) return null;
  if (deltaX === 0 && deltaY === 0) return null;
  return { deltaX, deltaY };
}

// previewMeasurement accepts the content height the preview paints, so the
// documentation stage can size itself to the component instead of a guess.
export function previewMeasurement(
  event: Pick<MessageEvent, 'origin' | 'source' | 'data'>,
  origin: string,
  activeSource: MessageEventSource | null,
): number | null {
  if (!activeSource || event.origin !== origin || event.source !== activeSource) return null;
  if (!event.data || typeof event.data !== 'object' || Array.isArray(event.data)) return null;
  if (event.data.type !== 'viva-preview-height') return null;
  const height = Number(event.data.height);
  if (!Number.isFinite(height) || height < 80 || height > 800) return null;
  return Math.round(height);
}

export function previewUpdate(
  event: Pick<MessageEvent, 'origin' | 'source' | 'data'>,
  origin: string,
  activeSource: MessageEventSource | null,
  state: PreviewState,
): PreviewUpdate | null {
  if (!activeSource || event.origin !== origin || event.source !== activeSource) return null;
  if (!event.data || typeof event.data !== 'object' || Array.isArray(event.data)) return null;
  if (event.data.type === 'viva-preview-ready' && state === 'loading') {
    return { state: 'ready', message: '' };
  }
  if (event.data.type === 'viva-preview-error' && state !== 'error') {
    const message = typeof event.data.message === 'string' && event.data.message.trim()
      ? event.data.message.slice(0, 1000)
      : 'The Go WebAssembly preview could not start. Retry the preview.';
    return { state: 'error', message };
  }
  return null;
}

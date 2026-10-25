import assert from 'node:assert/strict';
import { test } from 'node:test';
import { filterNavigation, previewMeasurement, previewPath, previewTimeout, previewUpdate, previewWheel } from './preview.ts';

const activeSource = {} as MessageEventSource;
const staleSource = {} as MessageEventSource;
const origin = 'http://127.0.0.1:4321';

test('preview paths encode generated IDs and use only the local runner', () => {
  assert.equal(previewPath('button', 'default', 'light', '573501b437e2', 'zh'), '/preview/index.html?component=button&example=default&theme=light&source=573501b437e2&locale=zh');
  assert.equal(previewPath('input', 'mixed text&case=1', 'dark', 'next revision', 'en'), '/preview/index.html?component=input&example=mixed+text%26case%3D1&theme=dark&source=next+revision&locale=en');
  assert.notEqual(previewPath('button', 'default', 'light', 'old', 'zh'), previewPath('button', 'default', 'light', 'new', 'zh'));
  assert.equal(previewTimeout, 60_000);
});

test('only the active preview can report its painted content height', () => {
  const data = { type: 'viva-preview-height', height: 246.4 };
  assert.equal(previewMeasurement({ source: activeSource, origin, data }, origin, activeSource), 246);
  assert.equal(previewMeasurement({ source: staleSource, origin, data }, origin, activeSource), null);
  assert.equal(previewMeasurement({ source: activeSource, origin: 'https://example.com', data }, origin, activeSource), null);
  assert.equal(previewMeasurement({ source: activeSource, origin, data: { type: 'viva-preview-ready', height: 246 } }, origin, activeSource), null);
  assert.equal(previewMeasurement({ source: activeSource, origin, data: { type: 'viva-preview-height', height: 20 } }, origin, activeSource), null);
  assert.equal(previewMeasurement({ source: activeSource, origin, data: { type: 'viva-preview-height', height: 'x' } }, origin, activeSource), null);
});

test('only the active preview can forward an unused wheel gesture', () => {
  const data = { type: 'viva-preview-wheel', deltaX: 2, deltaY: 30 };
  assert.deepEqual(previewWheel({ source: activeSource, origin, data }, origin, activeSource), { deltaX: 2, deltaY: 30 });
  assert.equal(previewWheel({ source: staleSource, origin, data }, origin, activeSource), null);
  assert.equal(previewWheel({ source: activeSource, origin: 'https://example.com', data }, origin, activeSource), null);
  assert.equal(previewWheel({ source: activeSource, origin, data: { type: 'viva-preview-ready' } }, origin, activeSource), null);
  assert.equal(previewWheel({ source: activeSource, origin, data: { type: 'viva-preview-wheel', deltaX: 0, deltaY: 0 } }, origin, activeSource), null);
  assert.equal(previewWheel({ source: activeSource, origin, data: { type: 'viva-preview-wheel', deltaX: 'x', deltaY: 1 } }, origin, activeSource), null);
});

test('only the current iframe from the exact origin can signal readiness', () => {
  const data = { type: 'viva-preview-ready' };
  assert.deepEqual(previewUpdate({ source: activeSource, origin, data }, origin, activeSource, 'loading'), { state: 'ready', message: '' });
  assert.equal(previewUpdate({ source: staleSource, origin, data }, origin, activeSource, 'loading'), null);
  assert.equal(previewUpdate({ source: activeSource, origin: 'https://example.com', data }, origin, activeSource, 'loading'), null);
  assert.equal(previewUpdate({ source: activeSource, origin: 'http://localhost:4321', data }, origin, activeSource, 'loading'), null);
  assert.equal(previewUpdate({ source: null, origin, data }, origin, null, 'loading'), null);
});

test('late ready events cannot resurrect timed-out or failed previews', () => {
  const event = { source: activeSource, origin, data: { type: 'viva-preview-ready' } };
  assert.equal(previewUpdate(event, origin, activeSource, 'error'), null);
  assert.equal(previewUpdate(event, origin, activeSource, 'ready'), null);
});

test('runtime errors after ready are retained as text and bounded', () => {
  const data = { type: 'viva-preview-error', message: '<img src=x onerror=alert(1)>' };
  assert.deepEqual(previewUpdate({ source: activeSource, origin, data }, origin, activeSource, 'ready'), { state: 'error', message: data.message });
  assert.equal(previewUpdate({ source: activeSource, origin, data }, origin, staleSource, 'ready'), null);
  const update = previewUpdate({ source: activeSource, origin, data: { ...data, message: 'x'.repeat(2000) } }, origin, activeSource, 'loading');
  assert.equal(update?.message.length, 1000);
});

test('malformed and unknown messages have no effect', () => {
  for (const data of [null, undefined, 'viva-preview-ready', [], {}, { type: 'ready' }, { type: 42 }]) {
    assert.equal(previewUpdate({ source: activeSource, origin, data }, origin, activeSource, 'loading'), null);
  }
  assert.ok(previewUpdate({ source: activeSource, origin, data: { type: 'viva-preview-error', message: 42 } }, origin, activeSource, 'loading')?.message);
});

test('navigation search matches name, ID and category without inventing links', () => {
  const items = [
    { ID: 'button', Title: 'Button', Category: 'Actions' },
    { ID: 'checkbox', Title: 'Checkbox', Category: 'Forms' },
    { ID: 'input', Title: 'Text input', Category: 'Forms' },
  ];
  assert.deepEqual(filterNavigation(items, '  FORM  '), [items[1], items[2]]);
  assert.deepEqual(filterNavigation(items, 'form input'), [items[2]]);
  assert.deepEqual(filterNavigation(items, 'BUTTON'), [items[0]]);
  assert.deepEqual(filterNavigation(items, 'unknown'), []);
  assert.deepEqual(filterNavigation(items, ''), items);
});

<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from './Icon.svelte';
  import type { Locale } from '../lib/i18n';

  type Entry = { title: string; sub: string; href: string; kind: 'component' | 'guide' | 'api' };

  let { locale, labels }: {
    locale: Locale;
    labels: { label: string; placeholder: string; empty: string; components: string; guides: string; api: string };
  } = $props();

  let open = $state(false);
  let query = $state('');
  let active = $state(0);
  let entries = $state<Entry[]>([]);
  let loading = $state(false);
  let failed = $state(false);
  let dialog = $state<HTMLDialogElement | null>(null);
  let field = $state<HTMLInputElement | null>(null);

  const groups = $derived([
    { kind: 'component' as const, heading: labels.components },
    { kind: 'guide' as const, heading: labels.guides },
    { kind: 'api' as const, heading: labels.api },
  ]);

  // Ranked: a title that starts with the query, then one that contains it, then a
  // match in the subtitle or category. Ties keep the shorter title, which is the
  // more general entry.
  const matches = $derived.by(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return [] as Entry[];
    return entries
      .map(entry => {
        const title = entry.title.toLowerCase();
        const sub = `${entry.sub} ${entry.kind}`.toLowerCase();
        const rank = title.startsWith(needle) ? 0 : title.includes(needle) ? 1 : sub.includes(needle) ? 2 : -1;
        return { entry, rank };
      })
      .filter(match => match.rank >= 0)
      .sort((left, right) => left.rank - right.rank || left.entry.title.length - right.entry.title.length)
      .slice(0, 24)
      .map(match => match.entry);
  });

  $effect(() => {
    const element = dialog;
    if (!element) return;
    if (open && !element.open) {
      element.showModal();
      field?.focus();
    } else if (!open && element.open) {
      element.close();
    }
  });

  async function load() {
    if (entries.length || loading) return;
    loading = true;
    try {
      const response = await fetch(`/search/${locale}.json`);
      if (!response.ok) throw new Error(String(response.status));
      const data = await response.json();
      if (Array.isArray(data.entries)) entries = data.entries;
    } catch {
      failed = true;
    } finally {
      loading = false;
    }
  }

  function show() {
    open = true;
    load();
  }

  function move(delta: number) {
    if (!matches.length) return;
    active = (active + delta + matches.length) % matches.length;
    document.getElementById(`search-result-${active}`)?.scrollIntoView({ block: 'nearest' });
  }

  function choose(index = active) {
    const entry = matches[index];
    if (!entry) return;
    open = false;
    location.assign(entry.href);
  }

  onMount(() => {
    const shortcut = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        show();
      }
    };
    window.addEventListener('keydown', shortcut);
    return () => window.removeEventListener('keydown', shortcut);
  });
</script>

<button class="search-trigger" type="button" aria-label={labels.label} title={labels.label} onclick={show}>
  <Icon name="search" size={15} />
  <span class="search-trigger-label">{labels.label}</span>
  <kbd aria-hidden="true">⌘K</kbd>
</button>

<dialog
  bind:this={dialog}
  class="search-dialog"
  aria-label={labels.label}
  onclose={() => (open = false)}
  onclick={event => {
    if (event.target === dialog) open = false;
  }}
>
  <div class="search-field">
    <Icon name="search" size={16} />
    <input
      bind:this={field}
      bind:value={query}
      type="text"
      autocomplete="off"
      spellcheck="false"
      placeholder={labels.placeholder}
      aria-label={labels.label}
      aria-controls="search-results"
      aria-activedescendant={matches.length ? `search-result-${active}` : undefined}
      oninput={() => (active = 0)}
      onkeydown={event => {
        if (event.key === 'ArrowDown') {
          event.preventDefault();
          move(1);
        } else if (event.key === 'ArrowUp') {
          event.preventDefault();
          move(-1);
        } else if (event.key === 'Enter') {
          event.preventDefault();
          choose();
        } else if (event.key === 'Escape') {
          event.preventDefault();
          open = false;
        }
      }}
    />
    <kbd aria-hidden="true">esc</kbd>
  </div>

  <div class="search-results" id="search-results" role="listbox" aria-label={labels.label}>
    {#if failed}
      <p class="search-note">{labels.empty}</p>
    {:else if loading}
      <p class="search-note" role="status">{labels.placeholder}</p>
    {:else if !query.trim()}
      <p class="search-note">{labels.placeholder}</p>
    {:else if !matches.length}
      <p class="search-note">{labels.empty}</p>
    {:else}
      {#each groups as group (group.kind)}
        {@const items = matches.filter(entry => entry.kind === group.kind)}
        {#if items.length}
          <div class="search-group">
            <h2>{group.heading}</h2>
            <ul>
              {#each items as entry (entry.href)}
                {@const index = matches.indexOf(entry)}
                <li>
                  <a
                    id="search-result-{index}"
                    href={entry.href}
                    role="option"
                    aria-selected={index === active}
                    class:active={index === active}
                    onmouseenter={() => (active = index)}
                    onclick={() => (open = false)}
                  >
                    <span class="search-title">{entry.title}</span>
                    <span class="search-sub">{entry.sub}</span>
                  </a>
                </li>
              {/each}
            </ul>
          </div>
        {/if}
      {/each}
    {/if}
  </div>
</dialog>

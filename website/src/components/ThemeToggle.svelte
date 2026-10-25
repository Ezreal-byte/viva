<script lang="ts">
  import { onMount } from 'svelte';
  import type { Theme } from '../lib/types';
  import Icon from './Icon.svelte';
  import { Button } from '$lib/components/ui/button';
  import { messages, type Locale } from '../lib/i18n';

  let { locale }: { locale: Locale } = $props();
  const text = $derived(messages[locale]);

  let theme = $state<Theme>('light');
  let mounted = $state(false);

  onMount(() => {
    theme = document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light';
    mounted = true;
  });

  function toggle() {
    theme = theme === 'light' ? 'dark' : 'light';
    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme;
    try {
      localStorage.setItem('viva-docs-theme', theme);
    } catch {}
    window.dispatchEvent(new CustomEvent('viva-theme-change', { detail: { theme } }));
  }
</script>

<Button variant="ghost" size="sm" class="theme-toggle" onclick={toggle} aria-label={text.darkTheme} aria-pressed={theme === 'dark'} disabled={!mounted}>
  <Icon name={theme === 'light' ? 'moon' : 'sun'} />
  <span class="theme-label">{theme === 'light' ? text.dark : text.light}</span>
</Button>

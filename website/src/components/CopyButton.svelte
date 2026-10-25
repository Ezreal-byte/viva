<script lang="ts">
  import { copyText } from '../lib/clipboard';
  import Icon from './Icon.svelte';
  import { Button } from '$lib/components/ui/button';
  import { messages, type Locale } from '../lib/i18n';

  let { text, label, prominent = false, locale }: { text: string; label?: string; prominent?: boolean; locale: Locale } = $props();
  const wording = $derived(messages[locale]);
  const copyLabel = $derived(label ?? wording.copySource);
  let pending = $state(false);
  let result = $state('');
  let failed = $state(false);

  $effect(() => {
    text;
    result = '';
    failed = false;
  });

  async function copy() {
    const snapshot = text;
    pending = true;
    result = '';
    try {
      await copyText(snapshot);
      if (text === snapshot) {
        failed = false;
        result = wording.copiedNotice;
      }
    } catch {
      if (text === snapshot) {
        failed = true;
        result = wording.copyFailed;
      }
    } finally {
      pending = false;
    }
  }
</script>

<div class="copy-control">
  <Button variant={prominent ? 'default' : 'outline'} size="sm" onclick={copy} disabled={pending || !text} aria-label={copyLabel} title={copyLabel}>
    <Icon name={result && !failed ? 'check' : 'copy'} />
  </Button>
  <span class="copy-result" class:copy-failed={failed} role="status" aria-live="polite" aria-atomic="true">{result}</span>
</div>

<script lang="ts">
  import { onMount } from 'svelte';
  import { copyText } from '../lib/clipboard';
  import { messages, type Locale } from '../lib/i18n';

  let { locale }: { locale: Locale } = $props();

  const copyIcon = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M8 8h12v12H8zM16 8V4H4v12h4"/></svg>';
  const checkIcon = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m5 12 4 4L19 6"/></svg>';

  onMount(() => {
    const text = messages[locale];
    const blocks = [...document.querySelectorAll<HTMLPreElement>('pre')].filter(pre =>
      !pre.closest('.code-block') && !pre.closest('.code-copy-wrap') && pre.textContent?.trim(),
    );
    for (const pre of blocks) {
      const wrap = document.createElement('div');
      wrap.className = 'code-copy-wrap';
      pre.parentNode?.insertBefore(wrap, pre);
      wrap.appendChild(pre);

      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'code-copy-button';
      button.setAttribute('aria-label', text.copyCode);
      button.title = text.copyCode;
      button.innerHTML = copyIcon;

      let reset: ReturnType<typeof setTimeout> | undefined;
      button.addEventListener('click', async () => {
        clearTimeout(reset);
        button.disabled = true;
        try {
          await copyText(pre.innerText);
          button.innerHTML = checkIcon;
          button.dataset.state = 'copied';
          button.setAttribute('aria-label', text.copied);
          button.title = text.copied;
        } catch {
          button.setAttribute('aria-label', text.copyFailed);
          button.title = text.copyFailed;
        } finally {
          button.disabled = false;
        }
        reset = setTimeout(() => {
          button.innerHTML = copyIcon;
          button.setAttribute('aria-label', text.copyCode);
          button.title = text.copyCode;
          delete button.dataset.state;
        }, 1600);
      });
      wrap.appendChild(button);
    }
  });
</script>

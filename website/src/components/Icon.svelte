<script lang="ts">
  let { name, size = 16 }: { name: keyof typeof paths; size?: number } = $props();
  const paths = {
    search: 'm21 21-4.4-4.4M19 11a8 8 0 1 1-16 0 8 8 0 0 1 16 0',
    sun: 'M12 3V1m0 22v-2M3 12H1m22 0h-2M5.6 5.6 4.2 4.2m15.6 15.6-1.4-1.4m0-12.8 1.4-1.4M4.2 19.8l1.4-1.4M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0',
    moon: 'M20.8 13A9 9 0 0 1 11 3.2 9 9 0 1 0 20.8 13Z',
    menu: 'M4 6h16M4 12h16M4 18h16',
    close: 'm6 6 12 12M6 18 18 6',
    arrow: 'M7 17 17 7M7 7h10v10',
    chevron: 'm9 5 7 7-7 7',
    copy: 'M8 8h12v12H8zM16 8V4H4v12h4',
    check: 'm5 12 4 4L19 6',
    reset: 'M3 10a9 9 0 1 1 2.6 8.4M3 4v6h6',
    warning: 'm12 3 10 18H2L12 3Zm0 6v5m0 3h.01',
  };
</script>

<svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
  <path d={paths[name]} />
</svg>

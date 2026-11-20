<script lang="ts">
  export let name: string;
</script>

<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
  {#if name === 'scan'}
    <path d="M8 3H4a1 1 0 0 0-1 1v4m13-5h4a1 1 0 0 1 1 1v4M3 16v4a1 1 0 0 0 1 1h4m8 0h4a1 1 0 0 0 1-1v-4M2 12h20" />
    <path d="M8 8V6h6l2 2v2m0 4v4H8v-4" />
  {:else if name === 'events'}
    <rect x="3" y="3" width="5" height="7" rx="1" />
    <rect x="3" y="14" width="5" height="7" rx="1" />
    <path d="M12 4h9m-9 5h9m-9 6h9m-9 5h9" />
  {:else if name === 'threats'}
    <path d="m12 2 8 3.5v5.4c0 4.6-3.2 8.1-8 11.1-4.8-3-8-6.5-8-11.1V5.5Z" />
    <path d="M12 7v6m0 4h.01" />
  {:else if name === 'settings'}
    <path d="m9.5 3-.7 2.2-2 .9-2.1-.5-2 3.4 1.6 1.7v2.6L2.7 15l2 3.4 2.1-.5 2 .9.7 2.2h5l.7-2.2 2-.9 2.1.5 2-3.4-1.6-1.7v-2.6L21.3 9l-2-3.4-2.1.5-2-.9-.7-2.2Z" />
    <circle cx="12" cy="12" r="3.25" />
  {/if}
</svg>

<style>
  svg { display: block; width: 100%; height: 100%; }
</style>

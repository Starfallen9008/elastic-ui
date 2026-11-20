<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';

  export let anchor: HTMLElement;
  export let id: string;
  export let title: string;
  const dispatch = createEventDispatcher<{ close: void }>();
  let panel: HTMLDivElement;
  let positioned = false;
  let placement = 'above';

  function close(restoreFocus = false) {
    if (restoreFocus) anchor?.focus({ preventScroll: true });
    dispatch('close');
  }

  function position() {
    if (!anchor?.isConnected || !panel) return;
    const rect = anchor.getBoundingClientRect();
    const width = window.innerWidth;
    const height = window.innerHeight;
    // An offscreen trigger must not leave an orphaned overlay on screen.
    if (rect.bottom < 0 || rect.top > height) { close(); return; }
    const above = rect.top - 16;
    const below = height - rect.bottom - 16;
    placement = above >= panel.scrollHeight || above >= below ? 'above' : 'below';
    panel.style.maxHeight = `${Math.max(80, placement === 'above' ? above : below)}px`;
    // offset sizes exclude the entrance scale and keep the anchor gap stable.
    panel.style.left = `${Math.max(8, Math.min(rect.left, width - panel.offsetWidth - 8))}px`;
    panel.style.top = `${Math.max(8, placement === 'above' ? rect.top - panel.offsetHeight - 8 : rect.bottom + 8)}px`;
    positioned = true;
  }

  onMount(() => {
    panel.showPopover(); // Top layer avoids clipping and transformed page ancestors.
    position();
    panel.focus({ preventScroll: true });
    const outside = (event: PointerEvent) => {
      const target = event.target as Node;
      if (!panel.contains(target) && !anchor.contains(target)) close();
    };
    const key = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        event.stopImmediatePropagation();
        close(true);
      }
    };
    const observer = new ResizeObserver(position);
    observer.observe(anchor);
    observer.observe(panel);
    window.addEventListener('resize', position);
    window.addEventListener('scroll', position, true);
    document.addEventListener('pointerdown', outside, true);
    window.addEventListener('keydown', key, true);
    return () => {
      observer.disconnect();
      window.removeEventListener('resize', position);
      window.removeEventListener('scroll', position, true);
      document.removeEventListener('pointerdown', outside, true);
      window.removeEventListener('keydown', key, true);
      panel.hidePopover();
    };
  });
</script>

<div bind:this={panel} {id} popover="manual" role="dialog" aria-label={title} tabindex="-1"
  class="anchored-popover" class:positioned data-placement={placement}>
  <div class="popover-head">
    <span>{title}</span>
    <button type="button" class="popover-close" on:click={() => close(true)} aria-label={`关闭${title}`}>
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m4 4 8 8m0-8-8 8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" /></svg>
    </button>
  </div>
  <div class="popover-content"><slot /></div>
</div>

<style>
  .anchored-popover {
    position: fixed;
    inset: auto;
    margin: 0;
    width: min(296px, calc(100vw - 16px));
    padding: 0;
    border: 1px solid var(--border-strong);
    border-radius: 6px;
    background: var(--panel);
    color: var(--text);
    box-shadow: 0 6px 20px rgb(0 0 0 / 14%);
    overflow: auto;
    visibility: hidden;
    transform-origin: bottom left;
  }
  .anchored-popover::backdrop { background: transparent; pointer-events: none; }
  .anchored-popover.positioned { visibility: visible; animation: reveal 180ms var(--ease); }
  .anchored-popover[data-placement='below'] { transform-origin: top left; }
  .popover-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 12px 16px; border-bottom: 1px solid var(--border); font-weight: 500; }
  .popover-content { padding: 4px 16px 16px; }
  .popover-close { display: grid; place-items: center; width: 28px; height: 28px; padding: 0; border: 0; border-radius: 4px; background: transparent; color: var(--muted); }
  .popover-close:hover { background: var(--glass-strong); color: var(--text); }
  @keyframes reveal { from { opacity: 0; transform: scale(0.98); } to { opacity: 1; transform: scale(1); } }
  @media (prefers-reduced-motion: reduce) { .anchored-popover.positioned { animation: none; } }
</style>

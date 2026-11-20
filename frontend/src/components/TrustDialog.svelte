<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import Trust from '../pages/Trust.svelte';

  export let isolated = false;
  const dispatch = createEventDispatcher();
  let busy = false;
  let feedback = '';
  let feedbackIsError = false;

  function openDialog(element: HTMLDialogElement) {
    const trigger = document.activeElement as HTMLElement | null;
    element.showModal();
    return { destroy() {
      element.close();
      if (trigger?.isConnected) trigger.focus({ preventScroll: true });
    } };
  }

  function close() {
    if (!busy) dispatch('close');
  }

  function cancel(event: Event) {
    event.preventDefault();
    close();
  }

  function outside(event: MouseEvent) {
    if (event.target !== event.currentTarget) return;
    const bounds = (event.currentTarget as HTMLDialogElement).getBoundingClientRect();
    if (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom) close();
  }

  function showFeedback(event: CustomEvent<{ message: string; error: boolean }>) {
    feedback = event.detail.message;
    feedbackIsError = event.detail.error;
  }
</script>

<!-- Native modal dialog handles focus trapping and makes the settings behind it inert. -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<dialog class="trust-dialog" aria-labelledby="trust-dialog-title" use:openDialog on:cancel={cancel} on:click={outside}>
  <header>
    <div>
      <h2 id="trust-dialog-title">信任区</h2>
    </div>
    <button class="btn ghost" aria-label="关闭信任区" disabled={busy} on:click={close}>关闭</button>
  </header>
  <div class="dialog-content" aria-busy={busy}>
    {#if feedback}<div class="feedback" class:error={feedbackIsError} role="status">{feedback}</div>{/if}
    <Trust {isolated} on:busy={(event) => { busy = event.detail; dispatch('busy', busy); }} on:toast={showFeedback} />
  </div>
</dialog>

<style>
  .trust-dialog { width: min(1120px, calc(100vw - 48px)); max-width: none; max-height: calc(100dvh - 64px); padding: 0; border: 1px solid var(--border-strong); border-radius: 10px; background: var(--bg); color: var(--text); box-shadow: var(--shadow); overflow: hidden; }
  .trust-dialog[open] { display: flex; flex-direction: column; }
  .trust-dialog::backdrop { background: var(--backdrop); }
  header { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 20px 24px; border-bottom: 1px solid var(--border); background: var(--panel); flex: none; }
  h2 { margin: 0; font-size: 20px; font-weight: 500; }
  .dialog-content { min-height: 0; overflow: auto; padding: 20px 24px 24px; overscroll-behavior: contain; }
  .feedback { position: sticky; top: 0; z-index: 1; padding: 10px 12px; margin-bottom: 16px; border: 1px solid var(--ok-border); border-radius: 4px; color: var(--ok); background: var(--panel); font-size: 13px; overflow-wrap: anywhere; }
  .feedback.error { border-color: var(--danger-border); color: var(--danger); }
  @media (max-width: 639px) {
    .trust-dialog { width: calc(100vw - 16px); max-height: calc(100dvh - 24px); }
    header { padding: 16px; }
    .dialog-content { padding: 12px; }
  }
</style>

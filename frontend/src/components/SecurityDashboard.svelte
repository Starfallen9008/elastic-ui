<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import NavigationIcon from './NavigationIcon.svelte';

  type NavItem = { id: string; label: string };

  export let items: NavItem[] = [];
  export let active = 'status';

  const dispatch = createEventDispatcher<{ navigate: string }>();
  const positions = ['scan', 'events', 'threats', 'settings'];

  let navigation: HTMLElement;
  $: orderedItems = positions.flatMap((id) => items.filter((item) => item.id === id));

  function handleKey(event: KeyboardEvent, index: number) {
    const buttons = navigation.querySelectorAll<HTMLButtonElement>('button');
    let next = index;
    if (event.key === 'ArrowRight' || event.key === 'ArrowDown') next = (index + 1) % buttons.length;
    else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') next = (index - 1 + buttons.length) % buttons.length;
    else if (event.key === 'Home') next = 0;
    else if (event.key === 'End') next = buttons.length - 1;
    else return;
    event.preventDefault();
    buttons[next]?.focus({ preventScroll: true });
  }
</script>

<div class="security-dashboard">
  <div class="dashboard-layout">
    <!-- Decorative siblings never contain interactive elements or the core. -->
    <div class="orbital-decoration" aria-hidden="true">
      <div class="orbital-ring primary-ring">
        <svg viewBox="0 0 520 520" fill="none">
          <circle class="ring-line" cx="260" cy="260" r="247" />
          <path class="ring-highlight" d="M260 13a247 247 0 0 1 174.66 72.34" />
          <circle class="orbital-particle" cx="260" cy="13" r="2.5" />
        </svg>
      </div>
      <div class="orbital-ring secondary-ring">
        <svg viewBox="0 0 520 520" fill="none">
          <path class="ring-line" d="M260 13a247 247 0 0 1 247 247M260 507A247 247 0 0 1 13 260" />
          <path class="ring-tick" d="M260 8v10M502 260h10M260 502v10M8 260h10" />
        </svg>
      </div>
    </div>

    <div class="security-core">
      <slot name="core" />
    </div>

    <nav class="fixed-navigation" aria-label="主导航" bind:this={navigation}>
      {#each orderedItems as item, index (item.id)}
        <button
          type="button"
          class="navigation-item position-{item.id}"
          class:active={item.id === active}
          aria-current={item.id === active ? 'page' : undefined}
          aria-label={item.label}
          on:keydown={(event) => handleKey(event, index)}
          on:click={() => dispatch('navigate', item.id)}
        >
          <span class="navigation-icon"><NavigationIcon name={item.id} /></span>
          <span class="navigation-label" aria-hidden="true">{item.label}</span>
        </button>
      {/each}
    </nav>
  </div>
</div>

<style>
  .security-dashboard {
    display: grid;
    place-items: center;
    width: 100%;
    height: 100%;
    container: dashboard / size;
  }

  .dashboard-layout {
    position: relative;
    display: grid;
    grid-template-columns: 104px minmax(280px, 1fr) 104px;
    grid-template-rows: 1fr 1fr;
    gap: 12px;
    align-items: center;
    justify-items: center;
    width: min(100%, 680px);
    height: min(100%, 520px);
    padding: 8px;
  }

  .security-core {
    position: relative;
    grid-area: 1 / 2 / 3 / 3;
    display: grid;
    place-items: center;
    width: 100%;
    height: 100%;
    min-width: 0;
    pointer-events: none;
  }

  .orbital-decoration {
    position: absolute;
    inset: 0;
    display: grid;
    place-items: center;
    pointer-events: none;
  }

  .orbital-ring {
    grid-area: 1 / 1;
    width: min(76cqw, 88cqh, 550px);
    aspect-ratio: 1;
    transform-origin: center;
    animation: orbital-rotation 90s linear infinite;
  }

  .orbital-ring svg {
    display: block;
    width: 100%;
    height: 100%;
    stroke-width: 1;
  }

  .ring-line { stroke: var(--security-ring); }
  .ring-highlight { stroke: var(--ok); opacity: 0.35; }
  .orbital-particle { fill: var(--ok); opacity: 0.65; }
  .ring-tick { stroke: var(--security-ring); }

  .secondary-ring {
    width: min(66cqw, 76cqh, 474px);
    animation-duration: 120s;
    animation-direction: reverse;
    opacity: 0.6;
  }

  .fixed-navigation { display: contents; }

  .navigation-item {
    position: relative;
    display: grid;
    place-items: center;
    width: 104px;
    height: 76px;
    padding: 0;
    border: 0;
    border-radius: 8px;
    background: transparent;
    color: var(--text-soft);
    font-size: 13px;
    white-space: nowrap;
    transition: color 200ms var(--ease);
  }

  .navigation-icon { grid-area: 1 / 1; width: 40px; height: 40px; transition: transform 240ms var(--ease); }
  .navigation-label { grid-area: 1 / 1; align-self: end; padding-bottom: 2px; opacity: 0; transform: translateY(5px); transition: opacity 220ms ease, transform 240ms var(--ease); }
  .position-scan { grid-area: 1 / 1; }
  .position-events { grid-area: 1 / 3; }
  .position-threats { grid-area: 2 / 1; }
  .position-settings { grid-area: 2 / 3; }

  .navigation-item:is(:hover, :focus-visible) {
    color: var(--ok);
  }

  .navigation-item:is(:hover, :focus-visible) .navigation-icon { transform: translateY(-10px) scale(0.88); }
  .navigation-item:is(:hover, :focus-visible) .navigation-label { opacity: 1; transform: translateY(0); }
  .navigation-item:active .navigation-icon { transform: translateY(-8px) scale(0.84); }
  .navigation-item:focus-visible { outline: 2px solid var(--ok); outline-offset: 4px; }

  @keyframes orbital-rotation { to { transform: rotate(360deg); } }

  :global([data-theme='light']) .navigation-item { font-size: 14px; }

  /* A separate grid replaces the ring layout before the core loses its safe area. */
  @container dashboard (max-width: 579px) or (max-height: 419px) {
    .dashboard-layout {
      grid-template-columns: minmax(0, 1fr);
      grid-template-rows: minmax(236px, 1fr) auto;
      gap: 20px;
      padding: 8px 4px;
      height: 100%;
    }

    .security-core { grid-area: 1 / 1; }
    .orbital-decoration { bottom: 96px; }
    .orbital-ring { width: min(88cqw, 264px); }
    .secondary-ring { width: min(76cqw, 228px); }
    .fixed-navigation {
      display: grid;
      grid-area: 2 / 1;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 10px;
      width: min(100%, 448px);
    }
    .navigation-item { grid-area: auto; width: 100%; }
    .navigation-icon { width: 34px; height: 34px; }
  }

  @container dashboard (max-width: 339px) {
    .fixed-navigation { gap: 8px; }
    .navigation-item { font-size: 12px; }
  }

  @media (prefers-reduced-motion: reduce) {
    .orbital-ring { animation: none; }
    .navigation-icon, .navigation-label { transition: none; }
  }

  @media (hover: none) {
    .navigation-icon { transform: translateY(-10px) scale(0.88); }
    .navigation-label { opacity: 1; transform: none; }
  }
</style>

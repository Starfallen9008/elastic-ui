<script lang="ts">
  import AgentMark from './AgentMark.svelte';

  export let label: string;
  export let description: string;
</script>

<div class="core-status" role="status" aria-live="polite" aria-atomic="true">
  <div class="shield-icon" aria-hidden="true">
    <div class="shield-glow"></div>
    <AgentMark />
  </div>
  <div class="security-status">{label}</div>
  <div class="security-description">{description}</div>
</div>

<style>
  .core-status {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 14px;
    width: min(100%, 280px);
    text-align: center;
  }

  .shield-icon {
    position: relative;
    width: clamp(104px, 25cqh, 148px);
    aspect-ratio: 1;
    color: rgb(var(--level-rgb));
  }

  .shield-glow {
    position: absolute;
    inset: -24px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(var(--level-rgb), 0.13), transparent 68%);
    animation: core-glow 9s ease-in-out infinite;
  }

  .security-status {
    color: rgb(var(--level-rgb));
    font-size: clamp(44px, 10cqh, 64px);
    font-weight: 500;
    letter-spacing: 0.06em;
    text-indent: 0.06em;
    line-height: 1.15;
    white-space: nowrap;
    text-shadow: 0 0 26px rgba(var(--level-rgb), 0.12);
  }

  .security-description {
    color: var(--muted);
    font-size: 12px;
    line-height: 1.6;
    max-width: 100%;
  }

  @keyframes core-glow {
    0%, 100% { opacity: 0.45; }
    50% { opacity: 0.85; }
  }

  :global([data-theme='light']) .shield-glow { display: none; }
  :global([data-theme='light']) .security-status { font-size: clamp(36px, 8cqh, 48px); letter-spacing: 0.02em; text-indent: 0.02em; text-shadow: none; }
  :global([data-theme='light']) .security-description { font-size: 13px; }

  @media (prefers-reduced-motion: reduce) {
    .shield-glow { animation: none; }
  }
</style>

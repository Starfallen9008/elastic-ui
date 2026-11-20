<script lang="ts">
  export let label: string;
  export let mode: string | undefined;
  export let unreadable = false;

  $: enabled = !unreadable && ['阻止', '仅检测', '已启用', '全部阻止', '已开启'].includes(mode || '');
  $: state = unreadable ? '无法读取' : enabled ? '已开启' : '未开启';
</script>

<span class="protection-state" aria-label={`${label}：${state}${!unreadable && mode ? `，${mode}` : ''}`} title={unreadable ? state : mode}>
  <span class="state-label">{state}</span>
  <span class="state-track" class:enabled aria-hidden="true"><span class="state-thumb"></span></span>
</span>

<style>
  .protection-state { display: inline-flex; align-items: center; gap: 8px; color: var(--muted); font-size: 12px; cursor: default; }
  .state-label { white-space: nowrap; }
  .state-track { display: inline-flex; align-items: center; width: 36px; height: 20px; padding: 3px; border-radius: 10px; background: var(--border-strong); }
  .state-thumb { width: 14px; height: 14px; border-radius: 50%; background: var(--muted); }
  .enabled { background: var(--ok); }
  .enabled .state-thumb { background: #fff; transform: translateX(16px); }
</style>

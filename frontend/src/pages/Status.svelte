<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import type { service } from '../../wailsjs/go/models';
  import SecurityCore from '../components/SecurityCore.svelte';
  import SecurityDashboard from '../components/SecurityDashboard.svelte';
  import AnchoredPopover from '../components/AnchoredPopover.svelte';
  import ProtectionState from '../components/ProtectionState.svelte';

  export let status: service.StatusView | null;
  export let statusError = '';
  export let items: { id: string; label: string }[] = [];

  const dispatch = createEventDispatcher();

  let showProtection = false;
  let protectionTrigger: HTMLButtonElement;
  let dismissedAlert = sessionStorage.getItem('ed-alert-dismissed') || '';
  let entered = false;
  let leaving = false;
  let navigationTimer: ReturnType<typeof setTimeout>;
  let reduced = false;

  onMount(() => {
    reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const timer = window.setTimeout(() => (entered = true), 40);
    return () => window.clearTimeout(timer);
  });

  onDestroy(() => clearTimeout(navigationTimer));

  function handleNavigate(event: CustomEvent<string>) {
    leaveTo(event.detail);
  }

  function leaveTo(id: string) {
    showProtection = false;
    if (!id || id === 'status' || leaving) return;
    leaving = true;
    navigationTimer = window.setTimeout(() => dispatch('navigate', id), reduced ? 0 : 260);
  }

  $: runtimeState = status?.agentRuntime?.state || 'UNAVAILABLE';
  $: connectionError = statusError || status?.connectionError || (!status?.connected ? '管理连接不可用，无法读取当前状态' : '');
  $: protectionError = connectionError || status?.protectionError || '';
  $: tamperUnreadable = !!connectionError || typeof status?.tamperProtection !== 'boolean';
  $: protectionRows = [
    { label: '恶意软件', mode: status?.protection?.malware },
    { label: '勒索软件', mode: status?.protection?.ransomware },
    { label: '内存威胁', mode: status?.protection?.memory },
    { label: '行为防护', mode: status?.protection?.behavior },
    { label: '设备控制', mode: status?.protection?.deviceControl }
  ].map(row => ({ ...row, unreadable: !!protectionError || !row.mode || row.mode === '未知' }));
  $: modes = status
    ? [status.protection?.malware, status.protection?.ransomware, status.protection?.memory, status.protection?.behavior]
    : [];
  $: blocking = modes.length === 4 && modes.every((mode) => mode === '阻止');
  $: allDisabled = modes.length === 4 && modes.every((mode) => mode === '关闭');
  $: reachable =
    !!status && !connectionError && status.connected && status.agentStatus === 'online' && status.endpointStatus === 'enrolled';
  $: runtimeDown = runtimeState === 'FAILED' || runtimeState === 'STOPPED' || runtimeState === 'UNAVAILABLE';
  $: level = !reachable || runtimeDown || allDisabled || !!protectionError ? 'red' : blocking && runtimeState === 'HEALTHY' ? 'green' : 'yellow';
  $: levelLabel = connectionError || protectionError ? '状态未知' : level === 'green' ? '保护中' : level === 'yellow' ? '降级' : '不可用';
  $: description = level === 'green'
    ? '四项防护均为阻止模式'
    : connectionError ? '管理连接失败，防护状态无法确认' : protectionError ? '无法读取防护策略' : level === 'yellow' ? '部分防护或运行状态需要检查' : '请检查 Agent 与防护连接';
  $: alertKey = status
    ? `${!!connectionError}|${!!protectionError}|${runtimeState}|${status.agentStatus}|${status.endpointStatus}|${status.protection?.malware}|${status.protection?.ransomware}|${status.protection?.memory}|${status.protection?.behavior}`
    : '';
  $: showAlert = !!status && level !== 'green' && dismissedAlert !== alertKey;

  function dismissAlert() {
    dismissedAlert = alertKey;
    sessionStorage.setItem('ed-alert-dismissed', alertKey);
  }

  function openAlert() {
    dismissedAlert = '';
    sessionStorage.removeItem('ed-alert-dismissed');
  }

  function copyRaw() {
    if (!status) return;
    const text = status.agentRuntime?.raw || status.agentRuntime?.error || '';
    navigator.clipboard?.writeText(text).then(
      () => dispatch('toast', { message: '已复制 Agent 状态输出', error: false }),
      () => dispatch('toast', { message: '复制失败', error: true })
    );
  }

  function handleKey(event: KeyboardEvent) {
    if (event.key !== 'Escape') return;
    if (showProtection) {
      showProtection = false;
    } else if (showAlert) {
      dismissAlert();
    }
  }
</script>

<svelte:window on:keydown={handleKey} />

{#if status}
  <div class="home" class:entered class:leaving style={`--level-rgb:var(--${level}-rgb)`}>
    <div class="orbit-area">
      <SecurityDashboard {items} active="status" on:navigate={handleNavigate}>
        <SecurityCore slot="core" label={levelLabel} {description} />
      </SecurityDashboard>
    </div>

    <div class="hero-meta">
      <span>{status.hostName}</span>
      <span class="rule"></span>
      <span>Elastic Agent {status.agentRuntime?.version || status.endpointVersion || '—'}</span>
      <span class="rule"></span>
      <span>{runtimeState}</span>
    </div>

    {#if level !== 'green'}
      <button class="alert-link" on:click={openAlert}>查看告警详情</button>
    {/if}

    <div class="home-bottom">
      <button bind:this={protectionTrigger} class="protection-trigger" on:click={() => (showProtection = !showProtection)} title="防护模块状态" aria-label="防护模块状态" aria-haspopup="dialog" aria-expanded={showProtection} aria-controls={showProtection ? 'protection-popover' : undefined}>
        <span class="trigger-dot" class:ok={level === 'green'} class:warn={level === 'yellow'} class:bad={level === 'red'}></span>
        防护模块
      </button>

      <div class="alert-stat" class:has-alerts={status.alerts24h > 0}>
           <span class="alert-count">{connectionError ? '—' : status.alerts24h}</span>
           <span class="alert-label">过去 24 小时告警{connectionError ? ' · 无法读取' : ''}</span>
          <button class="history-link" on:click={() => leaveTo('threats')}>查看威胁历史</button>
      </div>
    </div>
  </div>
{:else}
  <div class="empty">正在读取本机状态...</div>
{/if}

{#if showAlert && status}
  <div class="modal-backdrop" role="presentation" on:click={dismissAlert}>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div class="modal" role="dialog" aria-modal="true" aria-label="防护告警" tabindex="-1" on:click|stopPropagation>
      <div class="modal-head">
        <div>
          <div class="modal-kicker">需要处理</div>
          <div class="modal-title">{connectionError ? '管理连接失败' : protectionError ? '防护状态无法读取' : level === 'red' ? '防护不可用' : '防护降级'}</div>
        </div>
        <button class="btn ghost sm" on:click={dismissAlert}>关闭</button>
      </div>
      <div class="modal-body">
        <dl class="alert-grid">
          <dt>Agent</dt><dd>{connectionError ? '管理状态无法读取' : status.agentStatus || '无法读取'} · {status.agentRuntime?.message || '—'}</dd>
          <dt>运行状态</dt><dd>{runtimeState}</dd>
          <dt>Defend</dt><dd>{connectionError ? '无法读取' : status.endpointStatus || '无法读取'}</dd>
          <dt>防护模式</dt><dd>{protectionRows.slice(0, 4).map(row => row.unreadable ? '无法读取' : row.mode).join(' / ')}</dd>
        </dl>
        {#if protectionError}<div class="alert-error" role="alert">{protectionError}</div>{/if}
        {#if connectionError}<p class="connection-note">管理连接失败不代表本机防护已关闭，请结合本机 Agent 状态检查。</p>{/if}
        {#if status.agentRuntime?.error}
          <div class="alert-error">{status.agentRuntime.error}</div>
        {/if}
        <div class="raw-head">
          <span>elastic-agent status 输出</span>
          <button class="btn ghost sm" on:click={copyRaw}>复制</button>
        </div>
        <pre class="raw">{status.agentRuntime?.raw || status.agentRuntime?.error || '没有读取到详细输出'}</pre>
      </div>
    </div>
  </div>
{/if}

{#if showProtection && status}
  <AnchoredPopover anchor={protectionTrigger} id="protection-popover" title="防护模块" on:close={() => (showProtection = false)}>
        {#if protectionError || protectionRows.some(row => row.unreadable)}
          <div class="protection-error" role="alert"><strong>防护状态无法读取</strong><span>{protectionError || '部分模块未返回有效状态，请检查策略。'}</span></div>
        {/if}
        <div class="prot-grid">
          {#each protectionRows as row}
            <div class="prot-row"><span>{row.label}</span><ProtectionState label={row.label} mode={row.mode} unreadable={row.unreadable} /></div>
          {/each}
          <div class="prot-row"><span>防篡改</span><ProtectionState label="防篡改" mode={status.tamperProtection ? '已开启' : '关闭'} unreadable={tamperUnreadable} /></div>
        </div>
        <div class="policy-line">{connectionError ? '策略与版本无法读取' : `策略 ${status.policyName || '未读取到'} · 版本 ${status.policyVersion || '未读取到'}`}</div>
  </AnchoredPopover>
{/if}

<style>
  .home {
    height: 100%;
    min-height: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding-bottom: 2px;
  }

  .orbit-area {
    width: 100%;
    max-width: 1080px;
    flex: 1;
    min-height: 440px;
    position: relative;
  }

  .orbit-area,
  .hero-meta,
  .alert-link,
  .home-bottom {
    transition:
      opacity 0.42s var(--ease),
      transform 0.42s var(--ease),
      filter 0.3s ease;
  }

  .home:not(.entered) .orbit-area {
    opacity: 0;
    transform: translateY(-20px) scale(0.985);
  }

  .home:not(.entered) .hero-meta,
  .home:not(.entered) .alert-link,
  .home:not(.entered) .home-bottom {
    opacity: 0;
    transform: translateY(-10px);
  }

  .home.leaving .orbit-area {
    opacity: 0;
    transform: translateY(-30px) scale(0.98);
    filter: blur(2px);
    transition-duration: 0.26s;
  }

  .home.leaving .hero-meta,
  .home.leaving .alert-link,
  .home.leaving .home-bottom {
    opacity: 0;
    transform: translateY(-12px);
    transition-duration: 0.2s;
  }

  .hero-meta {
    flex: none;
    max-width: 100%;
    margin-top: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-wrap: wrap;
    gap: 10px;
    color: var(--muted);
    font-size: 13px;
  }

  .rule {
    width: 14px;
    height: 1px;
    background: var(--border-strong);
  }

  .alert-link {
    margin-top: 14px;
    border: 0;
    background: transparent;
    color: rgb(var(--level-rgb));
    font-size: 11px;
    text-decoration: underline;
    text-underline-offset: 3px;
    opacity: 0.85;
  }

  .alert-link:hover {
    opacity: 1;
  }

  .home-bottom {
    width: 100%;
    flex: none;
    margin-top: 16px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 18px;
    padding: 0 0 2px;
  }

  .protection-trigger {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-width: 164px;
    height: 40px;
    padding: 0 15px;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--glass);
    color: var(--text);
    font-size: 14px;
    transition: background-color 0.18s var(--ease), border-color 0.18s var(--ease), transform 0.09s ease;
  }

  .protection-trigger:hover {
    background: var(--glass-strong);
    border-color: var(--border-strong);
  }

  .protection-trigger:active {
    transform: scale(0.97);
  }

  .trigger-dot {
    width: 7px;
    height: 7px;
    flex: none;
    border-radius: 50%;
    background: var(--muted-2);
  }

  .trigger-dot.ok {
    background: var(--ok);
  }

  .trigger-dot.warn {
    background: var(--warn);
  }

  .trigger-dot.bad {
    background: var(--danger);
  }

  .alert-stat {
    display: flex;
    align-items: center;
    gap: 16px;
    min-height: 56px;
    padding: 8px 0 8px 24px;
    border-left: 1px solid var(--border-strong);
    color: var(--text);
    text-align: left;
  }

  .alert-count {
    font-size: 32px;
    font-weight: 500;
    line-height: 1;
    letter-spacing: -0.02em;
    font-variant-numeric: tabular-nums;
  }

  .alert-label {
    font-size: 14px;
    color: var(--muted);
  }

  .has-alerts .alert-count { color: var(--danger); }
  .history-link { min-height: 36px; border: 0; background: transparent; color: var(--accent); padding: 4px; font-size: 14px; text-decoration: underline; text-underline-offset: 4px; }

  .modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: 70;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    background: var(--backdrop);
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
  }

  .modal {
    width: min(760px, 92vw);
    max-height: 82vh;
    display: flex;
    flex-direction: column;
    border-radius: 16px;
    background: var(--glass-strong);
    border: 1px solid var(--border-strong);
    box-shadow: var(--shadow);
    backdrop-filter: blur(30px) saturate(130%);
    -webkit-backdrop-filter: blur(30px) saturate(130%);
    overflow: hidden;
  }

  .modal-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding: 16px 18px;
    border-bottom: 1px solid var(--border);
  }

  .modal-kicker {
    font-size: 10px;
    color: var(--muted-2);
    letter-spacing: 0.12em;
  }

  .modal-title {
    margin-top: 6px;
    font-size: 17px;
    font-weight: 600;
  }

  .modal-body {
    padding: 18px;
    overflow: auto;
  }

  .alert-grid {
    margin: 0;
    display: grid;
    grid-template-columns: 84px 1fr;
    gap: 8px 12px;
    font-size: 12px;
  }

  .alert-grid dt {
    color: var(--muted-2);
  }

  .alert-grid dd {
    margin: 0;
    color: var(--text-soft);
    word-break: break-all;
  }

  .alert-error {
    margin-top: 14px;
    padding: 9px 11px;
    border-radius: 7px;
    border: 1px solid var(--danger-border);
    background: var(--danger-bg);
    color: var(--danger);
    font-size: 11px;
    word-break: break-all;
  }

  .raw-head {
    margin-top: 18px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 10px;
    letter-spacing: 0.12em;
    color: var(--muted-2);
  }

  .raw {
    margin: 8px 0 0;
    padding: 12px;
    max-height: 300px;
    overflow: auto;
    border-radius: 8px;
    background: var(--cmd-bg);
    border: 1px solid var(--border);
    color: var(--text-soft);
    font-family: var(--font-mono);
    font-size: 11px;
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-all;
  }

  .prot-grid {
    display: flex;
    flex-direction: column;
  }

  .protection-error { display: grid; gap: 6px; padding: 10px; border: 1px solid var(--danger-border); border-radius: 4px; color: var(--danger); background: var(--danger-bg); font-size: 13px; line-height: 1.5; overflow-wrap: anywhere; }
  .connection-note { color: var(--text-soft); font-size: 13px; line-height: 1.6; }

  .prot-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 0;
    border-bottom: 1px solid var(--border);
    font-size: 13px;
  }

  .prot-row:last-child {
    border-bottom: 0;
  }

  .policy-line {
    margin-top: 12px;
    font-size: 13px;
    color: var(--muted);
    overflow-wrap: anywhere;
  }

  @media (prefers-reduced-motion: reduce) {
    .orbit-area,
    .hero-meta,
    .alert-link,
    .home-bottom {
      transition: none;
    }
  }

  @media (max-width: 639px) {
    .home-bottom { flex-wrap: wrap; }
    .alert-stat { width: 100%; flex-wrap: wrap; gap: 12px; padding-left: 0; border-left: 0; border-top: 1px solid var(--border); }
    .orbit-area { min-height: 440px; }
  }

  @media (max-width: 1100px) {
    .home-bottom {
      gap: 12px;
    }

    .alert-count {
      font-size: 34px;
    }
  }

  :global([data-theme='light']) .modal-backdrop { backdrop-filter: none; -webkit-backdrop-filter: none; }
  :global([data-theme='light']) .home:not(.entered) :is(.orbit-area, .hero-meta, .alert-link, .home-bottom) { opacity: 1; transform: none; }
  :global([data-theme='light']) .modal { background: var(--panel); backdrop-filter: none; -webkit-backdrop-filter: none; border-radius: 8px; }
  :global([data-theme='light']) :is(.modal-kicker, .raw-head, .alert-error, .alert-grid, .raw) { font-size: 13px; letter-spacing: 0; }
</style>

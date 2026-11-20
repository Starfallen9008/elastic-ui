<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';

  import {
    CloseWindow,
    Disconnect,
    GetSettings,
    GetStatus,
    Isolate,
    RebindLocal,
    SetTrayTheme
  } from '../wailsjs/go/main/App.js';
  import { EventsOn, WindowMinimise, WindowToggleMaximise } from '../wailsjs/runtime/runtime.js';
  import type { config, service } from '../wailsjs/go/models';

  import Setup from './pages/Setup.svelte';
  import IndependentPolicy from './components/IndependentPolicy.svelte';
  import Status from './pages/Status.svelte';
  import Threats from './pages/Threats.svelte';
  import Events from './pages/Events.svelte';
  import Scan from './pages/Scan.svelte';
  import TrustDialog from './components/TrustDialog.svelte';
  import ScanResults from './components/ScanResults.svelte';
  let quarantineOpen = false;
  import Settings from './pages/Settings.svelte';
  import { theme, toggleTheme } from './lib/theme';

  let settings: config.Settings | null = null;
  let status: service.StatusView | null = null;
  let page = 'status';
  let loading = true;
  let error = '';
  let toast = '';
  let toastIsError = false;
  let toastTimer: ReturnType<typeof setTimeout> | null = null;
  let refreshTimer: ReturnType<typeof setInterval> | null = null;
  let direction = 1;
  let reduced = false;
  let backLeaving = false;
  let trustBusy = false;
  let trustOpen = false;
  let refreshing = false;
  let notificationThreatID = '';

  const nav = [
    { id: 'status', label: '首页' },
    { id: 'threats', label: '威胁历史' },
    { id: 'events', label: '事件' },
    { id: 'scan', label: '扫描' },
    { id: 'settings', label: '设置' }
  ];

  $: isolated = !!status?.isolation;
  $: agentOnline = status?.agentStatus === 'online';
  $: needsSetup = !settings?.profile.kibanaUrl || !settings?.profile.agentId;

  function navigate(next: string) {
    if (trustBusy) { showToast('信任规则正在提交，请等待完成'); return; }
    if (next === 'trust') {
      navigate('settings');
      trustOpen = true;
      return;
    }
    if (next === page) return;
    trustOpen = false;
    const from = nav.findIndex((item) => item.id === page);
    const to = nav.findIndex((item) => item.id === next);
    direction = to >= from ? 1 : -1;
    page = next;
  }

  function goBack() {
    if (page === 'status' || page === 'setup' || backLeaving) return;
    backLeaving = true;
    window.setTimeout(() => {
      backLeaving = false;
      navigate('status');
    }, 180);
  }

  async function refresh() {
    if (refreshing || !settings?.profile.agentId) return;
    refreshing = true;
    try {
      status = await GetStatus();
      error = '';
    } catch (e) {
      error = String(e);
    } finally {
      refreshing = false;
    }
  }

  function showToast(message: string, isError = false) {
    toast = message;
    toastIsError = isError;
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(() => (toast = ''), 4200);
  }

  async function handleConfigured() {
    settings = await GetSettings();
    await refresh();
    navigate(settings?.policySetupComplete ? 'status' : 'policy-setup');
    showToast('连接成功');
  }

  async function handlePolicyChanged() { settings = await GetSettings(); await refresh(); }
  async function handlePolicyDone() { await handlePolicyChanged(); navigate('status'); }

  async function handleIsolate(host: string) {
    try {
      status = await Isolate(host);
      showToast('隔离指令已下发');
    } catch (e) {
      showToast(String(e), true);
    }
  }

  async function handleRebind() {
    try {
      status = await RebindLocal();
      settings = await GetSettings();
      showToast('本机绑定已刷新');
    } catch (e) {
      showToast(String(e), true);
    }
  }

  async function handleDisconnect() {
    await Disconnect();
    settings = await GetSettings();
    status = null;
    page = 'setup';
  }

  onMount(async () => {
    reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

    settings = await GetSettings();
    try {
      SetTrayTheme($theme === 'dark');
    } catch {
      // 运行时未就绪时由主题切换再次同步
    }

    EventsOn('navigate', (target: string) => { if (target) navigate(target); });
    EventsOn('scan-error', (message: string) => showToast(message, true));
    EventsOn('notification-details', (id: string) => { notificationThreatID = id; navigate('threats'); });
    EventsOn('configured', handleConfigured);
    refreshTimer = setInterval(refresh, 20000);
    if (!settings.profile.kibanaUrl || !settings.profile.agentId) {
      page = 'setup';
      loading = false;
      return;
    }

    await refresh();
    loading = false;
    if (!settings.policySetupComplete) page = 'policy-setup';
  });

  onDestroy(() => {
    if (refreshTimer) clearInterval(refreshTimer);
    if (toastTimer) clearTimeout(toastTimer);
  });
</script>

<div class="shell">
  <header class="titlebar" style="--wails-draggable: drag">
    {#if !needsSetup && page !== 'setup' && page !== 'status'}
      <button class="back-btn no-drag" disabled={trustBusy} on:click={goBack} aria-label="返回首页">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <path d="M10 3.5 5.5 8 10 12.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        返回
      </button>
    {/if}

    <div class="title-actions no-drag">
      {#if error}
        <span class="conn-error" title={error}>连接异常</span>
      {/if}
      {#if status}
        <div class="host-chip">
          <span class="dot" class:ok={agentOnline} class:bad={!agentOnline}></span>
          {status.hostName}
        </div>
      {/if}
      <button class="icon-btn" on:click={toggleTheme} title={$theme === 'dark' ? '切换到浅色' : '切换到深色'} aria-label="切换主题">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <circle cx="8" cy="8" r="5.2" stroke="currentColor" stroke-width="1.3" />
          <path d="M8 2.8a5.2 5.2 0 0 1 0 10.4Z" fill="currentColor" opacity="0.85" />
        </svg>
      </button>
      <button class="icon-btn" on:click={() => WindowMinimise()} title="最小化" aria-label="最小化">
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
          <path d="M2.5 6h7" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
        </svg>
      </button>
      <button class="icon-btn" on:click={() => WindowToggleMaximise()} title="最大化" aria-label="最大化">
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
          <rect x="2.6" y="2.6" width="6.8" height="6.8" rx="1" stroke="currentColor" stroke-width="1.4" />
        </svg>
      </button>
      <button class="icon-btn close" on:click={() => CloseWindow()} title="关闭" aria-label="关闭">
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
          <path d="m3 3 6 6M9 3l-6 6" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
        </svg>
      </button>
    </div>
  </header>

  {#if isolated && !needsSetup && page !== 'setup'}
    <div class="lock-banner">
      <strong>本机已隔离</strong>
      <span>网络已切断。解除隔离只能由管理员在 Kibana 操作。</span>
    </div>
  {/if}

  <main class="content" class:home-view={!needsSetup && page === 'status'} class:setup={needsSetup || page === 'setup'}>
    {#if loading}
      <div class="boot">正在读取本机绑定...</div>
    {:else if needsSetup || page === 'setup'}
      <Setup initial={settings?.profile} on:done={handleConfigured} />
    {:else}
      {#key page}
        <div
          class="page"
          class:home-page={page === 'status'}
          class:back-leaving={backLeaving}
          in:fly={{ x: direction * 10, duration: reduced || (page === 'status' && $theme === 'light') ? 0 : 180, easing: cubicOut }}
          out:fade={{ duration: reduced ? 0 : 100 }}
        >
           {#if page === 'policy-setup'}
             <IndependentPolicy host={settings?.profile.hostName || ''} onboarding {isolated} on:changed={handlePolicyChanged} on:done={handlePolicyDone} />
           {:else if page === 'status'}
            <Status {status} statusError={error} items={nav} on:navigate={(e) => navigate(e.detail)} on:toast={(e) => showToast(e.detail.message, e.detail.error)} />
          {:else if page === 'threats'}
             <Threats {isolated} initialAlertID={notificationThreatID} on:opened={() => notificationThreatID = ''} on:toast={(e) => showToast(e.detail.message, e.detail.error)} />
          {:else if page === 'events'}
            <Events on:toast={(e) => showToast(e.detail.message, e.detail.error)} />
          {:else if page === 'scan'}
            <Scan {isolated} on:toast={(e) => showToast(e.detail.message, e.detail.error)} />
          {:else if page === 'settings'}
            <Settings
              {settings}
              {status}
              {isolated}
              on:isolate={(e) => handleIsolate(e.detail)}
              on:toast={(e) => showToast(e.detail.message, e.detail.error)}
              on:rebind={handleRebind}
              on:disconnect={handleDisconnect}
                on:exclusions={() => trustOpen = true}
                on:quarantine={() => quarantineOpen = true}
               on:policychanged={handlePolicyChanged}
            />
          {/if}
        </div>
      {/key}
    {/if}
  </main>

  {#if trustOpen}
    <TrustDialog {isolated} on:close={() => { trustOpen = false; trustBusy = false; }} on:busy={(e) => trustBusy = e.detail} />
  {/if}
  {#if quarantineOpen}<ScanResults {isolated} on:close={() => quarantineOpen = false} />{/if}

  {#if toast}
    <div class="toast" class:error={toastIsError}>{toast}</div>
  {/if}
</div>

<style>
  .shell {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: var(--bg);
    position: relative;
  }

  .titlebar {
    height: var(--header-h);
    flex: none;
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 0 12px 0 18px;
    background: transparent;
    border-bottom: 0;
    position: relative;
    z-index: 20;
  }

  .back-btn {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    height: 32px;
    padding: 0 13px 0 9px;
    border-radius: 9px;
    border: 1px solid var(--border);
    background: var(--glass);
    color: var(--text-soft);
    font-size: 12px;
    transition:
      background-color 0.18s var(--ease),
      border-color 0.18s var(--ease),
      color 0.18s var(--ease),
      transform 0.09s ease;
  }

  .back-btn:hover {
    background: var(--glass-strong);
    border-color: var(--border-strong);
    color: var(--text);
  }

  .back-btn:active {
    transform: scale(0.97);
  }

  .title-actions {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 2px;
    position: relative;
    z-index: 3;
  }

  .host-chip {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-right: 8px;
    font-size: 11px;
    color: var(--muted);
  }

  .conn-error {
    margin-right: 6px;
    padding: 2px 7px;
    border-radius: 999px;
    border: 1px solid var(--danger-border);
    background: var(--danger-bg);
    color: var(--danger);
    font-size: 10px;
  }

  .icon-btn {
    width: 30px;
    height: 28px;
    border: 0;
    background: transparent;
    color: var(--muted);
    border-radius: 7px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background-color 0.16s ease, color 0.16s ease, transform 0.09s ease;
  }

  .icon-btn:hover {
    background: var(--glass);
    color: var(--text);
  }

  .icon-btn:active {
    transform: scale(0.94);
  }

  .icon-btn.close:hover {
    background: var(--danger-bg);
    color: var(--danger);
  }

  .content {
    flex: 1;
    min-height: 0;
    min-width: 0;
    overflow-x: hidden;
    overflow-y: auto;
    padding: 26px 30px 32px;
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    align-content: start;
  }

  .content.home-view {
    padding: 8px 30px 24px;
    overflow-x: hidden;
    overflow-y: auto;
    grid-template-rows: minmax(0, 1fr);
    align-content: stretch;
  }

  .content.setup {
    padding: 0;
    overflow: hidden;
  }

  .page {
    grid-area: 1 / 1;
    min-width: 0;
    min-height: 100%;
    transition: opacity 0.16s ease, transform 0.18s var(--ease);
  }

  .page.home-page {
    height: 100%;
    min-height: 0;
  }

  @media (max-width: 639px) {
    .content.home-view { padding: 8px 16px 20px; }
  }

  .page.back-leaving {
    opacity: 0;
    transform: translateY(14px);
    pointer-events: none;
  }

  .boot {
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted);
    font-size: 13px;
  }

  .lock-banner {
    flex: none;
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 10px 16px;
    border-bottom: 1px solid var(--danger-border);
    background: var(--danger-bg);
    color: var(--danger);
    font-size: 12px;
  }

  .lock-banner strong {
    white-space: nowrap;
  }

  .toast {
    position: fixed;
    right: 24px;
    top: 64px;
    z-index: 80;
    max-width: min(520px, calc(100vw - 48px));
    padding: 10px 14px;
    border-radius: 10px;
    background: var(--glass-strong);
    border: 1px solid var(--border-strong);
    color: var(--text);
    font-size: 12px;
    box-shadow: var(--shadow);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
  }

  .toast.error {
    background: var(--danger-bg);
    border-color: var(--danger-border);
    color: var(--danger);
  }

  :global([data-theme='light']) .back-btn { border-radius: 4px; font-size: 13px; }
  :global([data-theme='light']) .host-chip { font-size: 13px; }
  :global([data-theme='light']) .conn-error { border-radius: 3px; font-size: 13px; }
  :global([data-theme='light']) .toast { border-radius: 6px; font-size: 14px; background: var(--panel); backdrop-filter: none; }

  @media (max-width: 1120px) {
    .host-chip {
      display: none;
    }
  }
</style>

<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import { AppInfo, CheckUpdate, InstallUpdate, ClearCache, ExportDiagnostics, GetNotificationState, OpenKibana, SetCloseToTray, SetNotificationMode, SetShellIntegration, ShellIntegrationEnabled, StartupEnabled, SetStartupEnabled } from '../../wailsjs/go/main/App.js';
  import type { config, service } from '../../wailsjs/go/models';
  import IndependentPolicy from '../components/IndependentPolicy.svelte';
  import { formatTime } from '../lib/format';

  export let settings: config.Settings | null;
  export let status: service.StatusView | null;
  export let isolated = false;

  const dispatch = createEventDispatcher();

  let shellEnabled = false;
  let notificationMode = 'native';
  let notificationBusy = false;
  let notificationMessage = '';
  let notificationReady = true;
  let notificationError = '';
  let notificationTimer: ReturnType<typeof setInterval>;
  let checkingNotifications = false;

  async function refreshNotifications() {
    if (checkingNotifications || notificationBusy) return;
    checkingNotifications = true;
    try {
      const state = await GetNotificationState();
      notificationMode = state.mode;
      notificationMessage = state.message;
      notificationReady = state.ready;
    } catch (e) {
      notificationError = String(e);
    } finally { checkingNotifications = false; }
  }
  let closeToTray = true;
  let info: Record<string, string> = {};
  let busy = false;
  let startup = false;
  let startupBusy = false;
  let updateBusy = false;
  let updateInstalling = false;
  let updateError = '';
  let checkedUpdate = false;
  let release: { version: string; available: boolean; notes: string } | null = null;
  let confirmHost = '';
  let isolating = false;

  $: host = status?.hostName || settings?.profile.hostName || '';
  $: canIsolate = !!host && confirmHost.toLowerCase() === host.toLowerCase() && !isolating;

  function isolate() {
    if (!canIsolate) return;
    isolating = true;
    dispatch('isolate', confirmHost);
    window.setTimeout(() => (isolating = false), 1400);
  }

  async function toggleShell() {
    busy = true;
    try {
      await SetShellIntegration(!shellEnabled);
      shellEnabled = await ShellIntegrationEnabled();
      dispatch('toast', { message: shellEnabled ? '右键菜单已注册' : '右键菜单已移除', error: false });
    } catch (e) {
      dispatch('toast', { message: String(e), error: true });
    } finally {
      busy = false;
    }
  }

  async function exportDiagnostics() {
    try {
      const path = await ExportDiagnostics();
      if (path) {
        dispatch('toast', { message: `诊断包已保存：${path}`, error: false });
      }
    } catch (e) {
      dispatch('toast', { message: String(e), error: true });
    }
  }

  async function clearCache() {
    if (!confirm('清理扫描历史和已读状态？审计日志不会删除。')) return;
    try {
      await ClearCache();
      dispatch('toast', { message: '本地缓存已清理', error: false });
    } catch (e) {
      dispatch('toast', { message: String(e), error: true });
    }
  }

  async function changeNotifications(mode: string) {
    if (notificationBusy) return;
    notificationBusy = true;
    notificationError = '';
    try {
      await SetNotificationMode(mode);
      notificationMode = mode;
      if (settings) settings.notificationMode = mode;
      const state = await GetNotificationState();
      notificationMessage = state.message;
      notificationReady = state.ready;
    } catch (e) {
      notificationError = String(e);
    } finally {
      notificationBusy = false;
    }
  }

  async function toggleCloseToTray() {
    try {
      await SetCloseToTray(!closeToTray);
      closeToTray = !closeToTray;
      if (settings) settings.closeToTray = closeToTray;
      dispatch('toast', { message: closeToTray ? '关闭窗口将隐藏到托盘' : '关闭窗口将退出程序', error: false });
    } catch (e) {
      dispatch('toast', { message: String(e), error: true });
    }
  }

  async function toggleStartup() {
    startupBusy = true;
    try {
      await SetStartupEnabled(!startup);
      startup = await StartupEnabled();
      dispatch('toast', { message: startup ? '开机自启已启用' : '开机自启已禁用', error: false });
    } catch (e) { dispatch('toast', { message: String(e), error: true }); }
    finally { startupBusy = false; }
  }

  async function checkUpdate() {
    updateBusy = true; updateError = '';
    try { release = await CheckUpdate(); checkedUpdate = true; }
    catch (e) { updateError = String(e); }
    finally { updateBusy = false; }
  }

  async function installUpdate() {
    if (!confirm(`安装 v${release?.version} 并重新启动程序？`)) return;
    updateInstalling = true; updateError = '';
    try { await InstallUpdate(); }
    catch (e) { updateError = String(e); updateInstalling = false; }
  }

  onMount(async () => {
    try { startup = await StartupEnabled(); }
    catch (e) { dispatch('toast', { message: String(e), error: true }); }
    shellEnabled = await ShellIntegrationEnabled();
    notificationMode = settings?.notificationMode || 'native';
    refreshNotifications();
    notificationTimer = setInterval(refreshNotifications, 15000);
    closeToTray = settings?.closeToTray ?? true;
    info = await AppInfo();
    if (info.updateError) updateError = `上次更新失败：${info.updateError}`;
  });
  onDestroy(() => clearInterval(notificationTimer));
</script>

<div class="cards">
  <section class="panel isolate-panel" aria-labelledby="isolation-title">
    <div class="panel-head">
      <h2 class="panel-title" id="isolation-title">应急隔离</h2>
      {#if isolated}<span class="badge danger">已隔离</span>{/if}
    </div>
    <div class="panel-body">
      {#if isolated}
        <p class="iso-text">本机已切断网络。解除隔离只能由管理员在 Kibana 进行。</p>
      {:else}
        <p class="iso-text">怀疑本机失陷时使用。隔离后除 Elasticsearch 与 Kibana 外，网络连接会被切断，本程序无法解除。</p>
        <div class="iso-confirm">
          <input aria-label="输入主机名确认隔离" bind:value={confirmHost} placeholder="输入主机名 {host} 确认" />
          <button class="btn danger" disabled={!canIsolate} on:click={isolate}>
            {isolating ? '下发中...' : '隔离此主机'}
          </button>
        </div>
      {/if}
    </div>
  </section>

  <section class="panel">
    <div class="panel-head">
      <div class="panel-title">连接</div>
      <button class="btn ghost sm" on:click={() => OpenKibana('/app/security/alerts')}>在 Kibana 中打开</button>
    </div>
    <div class="panel-body">
      <dl>
        <dt>Kibana</dt><dd class="mono">{settings?.profile.kibanaUrl || '—'}</dd>
        <dt>Elasticsearch</dt><dd class="mono">{settings?.profile.esUrl || '—'}</dd>
        <dt>认证方式</dt><dd>{settings?.profile.authMethod === 'apikey' ? 'API Key ' + (settings?.profile.apiKeyHint || '') : '用户名密码 ' + (settings?.profile.username || '')}</dd>
        <dt>Kibana 版本</dt><dd>{status?.kibanaVersion || '—'}</dd>
        <dt>授权</dt><dd>{status?.license || '—'}</dd>
      </dl>
    </div>
  </section>

  <section class="panel">
    <div class="panel-head">
      <div class="panel-title">本机绑定</div>
      <button class="btn ghost sm" on:click={() => dispatch('rebind')}>重新绑定</button>
    </div>
    <div class="panel-body">
      <dl>
        <dt>主机名</dt><dd>{status?.hostName || settings?.profile.hostName || '—'}</dd>
        <dt>Agent ID</dt><dd class="mono break">{status?.agentId || settings?.profile.agentId || '—'}</dd>
        <dt>Endpoint 版本</dt><dd>{status?.endpointVersion || '—'}</dd>
        <dt>Agent 策略</dt><dd class="mono break">{settings?.profile.agentPolicyId || '—'}</dd>
        <dt>Defend 策略</dt><dd>{status?.policyName || settings?.profile.policyName || '—'}</dd>
        <dt>策略包 ID</dt><dd class="mono break">{settings?.profile.packagePolicyId || '—'}</dd>
        <dt>绑定时间</dt><dd class="binding-time">{formatTime(settings?.profile.boundAt || '')}</dd>
      </dl>
      <IndependentPolicy {host} {isolated} on:changed={() => dispatch('policychanged')} />
    </div>
  </section>

  <section class="panel">
    <div class="panel-head">
      <div class="panel-title">右键菜单开关</div>
    </div>
    <div class="panel-body">
      <div class="toggle-row">
        <div>
          <div>右键菜单：使用 Elastic Defend 进行扫描</div>
        </div>
        <button class="btn" class:primary={shellEnabled} aria-pressed={shellEnabled} disabled={busy} on:click={toggleShell}>
          {shellEnabled ? '已启用' : '已禁用'}
        </button>
      </div>
    </div>
  </section>

  <section class="panel">
    <div class="panel-head">
      <div class="panel-title">通知与托盘</div>
    </div>
    <div class="panel-body">
      <div class="maint-row">
        <div>
          <div>拦截通知</div>
        </div>
        <div class="notification-options" role="group" aria-label="拦截通知来源" aria-busy={notificationBusy}>
          <button class="btn" class:primary={notificationMode === 'native'} aria-pressed={notificationMode === 'native'} disabled={notificationBusy || isolated} on:click={() => changeNotifications('native')}>Elastic 原生</button>
          <button class="btn" class:primary={notificationMode === 'ui'} aria-pressed={notificationMode === 'ui'} disabled={notificationBusy || isolated} on:click={() => changeNotifications('ui')}>Elastic UI</button>
        </div>
      </div>
      {#if notificationBusy || !notificationReady}<p class="notification-status" role="status">{notificationBusy ? '正在同步通知策略…' : notificationMessage}</p>{/if}
      {#if notificationError}<p class="notification-error" role="alert">{notificationError}</p>{/if}
      <div class="maint-row">
        <div>
          <div>关闭窗口时隐藏到托盘</div>
        </div>
        <button class="btn" class:primary={closeToTray} aria-pressed={closeToTray} on:click={toggleCloseToTray}>
          {closeToTray ? '已启用' : '已禁用'}
        </button>
      </div>
      <div class="maint-row">
        <div>开机自启</div>
        <button class="btn" class:primary={startup} aria-label="开机自启" aria-pressed={startup} disabled={startupBusy} on:click={toggleStartup}>{startupBusy ? '正在设置…' : startup ? '已启用' : '已禁用'}</button>
      </div>
    </div>
  </section>

  <section class="panel">
    <div class="panel-head">
      <div class="panel-title">维护</div>
    </div>
    <div class="panel-body">
      <div class="maint-row">
        <div>
          <div>导出诊断包</div>
        </div>
        <button class="btn" on:click={exportDiagnostics}>导出</button>
      </div>
      <div class="maint-row">
        <div>
          <div>清理本地缓存</div>
        </div>
        <button class="btn danger" on:click={clearCache}>清理</button>
      </div>
    </div>
  </section>

  <section class="panel">
    <div class="panel-head">
      <div class="panel-title">信任区</div>
    </div>
    <div class="panel-body">
      <div class="maint-row">
        <div>
          <div>本机信任规则</div>
        </div>
        <button class="btn" aria-haspopup="dialog" on:click={() => dispatch('exclusions')}>信任区</button>
      </div>
      <div class="maint-row">
        <div>隔离文件</div>
        <button class="btn" aria-haspopup="dialog" on:click={() => dispatch('quarantine')}>隔离区</button>
      </div>
    </div>
  </section>

  <section class="panel">
    <div class="panel-head">
      <div class="panel-title">关于</div>
    </div>
    <div class="panel-body">
      <dl>
        <dt>版本</dt><dd>{info.version || '1.0.0'}</dd>
        <dt>运行时</dt><dd>{info.buildNote || '—'}</dd>
        <dt>配置目录</dt><dd class="mono break">{info.configDir || '—'}</dd>
      </dl>
      <div class="maint-row update-row">
        <div>{updateInstalling ? '正在下载并校验更新…' : release?.available ? `发现新版本 v${release.version}` : checkedUpdate ? '已是最新版本' : '程序更新'}</div>
        <div class="update-actions"><button class="btn" disabled={updateBusy || updateInstalling} on:click={checkUpdate}>{updateBusy ? '正在检查…' : '检查更新'}</button>{#if release?.available}<button class="btn primary" disabled={updateInstalling} on:click={installUpdate}>更新并重启</button>{/if}</div>
      </div>
      {#if updateError}<div class="notification-error" role="alert">{updateError}</div>{/if}
      <div class="danger-zone">
        <div>
          <div>断开连接</div>
        </div>
        <button class="btn danger" on:click={() => confirm('断开连接并清除本地凭证？') && dispatch('disconnect')}>断开连接</button>
      </div>
    </div>
  </section>
</div>

<style>
  .cards {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 16px;
    width: 100%;
    max-width: 1080px;
    margin: 0 auto;
  }

  .panel-title { margin: 0; }
  .update-actions { display: flex; flex-wrap: wrap; gap: 8px; justify-content: flex-end; }
  .update-row { margin-top: 16px; }
  .notification-options { display: flex; gap: 8px; flex-wrap: wrap; flex: none; }
  .notification-error { color: var(--danger); font-size: 12px; overflow-wrap: anywhere; }

  dl {
    margin: 0;
    display: grid;
    grid-template-columns: 112px minmax(0, 1fr);
    gap: 12px 16px;
    font-size: 13px;
    line-height: 1.6;
  }

  dt {
    color: var(--muted-2);
  }

  dd {
    margin: 0;
    color: var(--text-soft);
    overflow-wrap: anywhere;
  }

  .break {
    word-break: break-all;
  }

  .binding-time { font-variant-numeric: tabular-nums; }
  .notification-status { margin: 12px 0; font-size: 13px; color: var(--muted); overflow-wrap: anywhere; }

  .toggle-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
    font-size: 13px;
  }

  .maint-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
    font-size: 13px;
    padding: 12px 0;
  }

  .maint-row + .maint-row {
    border-top: 1px solid var(--border-soft);
  }

  .maint-row:first-child { padding-top: 0; }
  .maint-row:last-child { padding-bottom: 0; }
  .maint-row > div, .toggle-row > div, .danger-zone > div { min-width: 0; }
  .maint-row .btn, .toggle-row .btn, .danger-zone .btn { flex: none; }

  .isolate-panel {
    border-color: var(--danger-border);
  }

  .isolate-panel .panel-head {
    border-bottom-color: var(--danger-border);
  }

  .iso-text {
    margin: 0 0 14px;
    color: var(--muted);
    font-size: 12px;
    line-height: 1.7;
  }

  .iso-confirm {
    display: flex;
    gap: 10px;
  }

  .iso-confirm input {
    flex: 1;
    min-width: 0;
  }

  .danger-zone {
    margin-top: 20px;
    padding-top: 16px;
    border-top: 1px solid var(--border-soft);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
    font-size: 13px;
  }

  @media (max-width: 639px) {
    dl { grid-template-columns: 88px minmax(0, 1fr); }
    .iso-confirm { flex-direction: column; }
    .maint-row, .toggle-row, .danger-zone { flex-wrap: wrap; gap: 12px; }
  }
</style>

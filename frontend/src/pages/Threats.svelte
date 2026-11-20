<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import { CreateTrust, GetThreatDetail, GetThreats, MarkRead } from '../../wailsjs/go/main/App.js';
  import type { service } from '../../wailsjs/go/models';
  import { actionLabel, formatTime, relative, severityClass, severityLabel } from '../lib/format';
  import ProcessChain from '../components/ProcessChain.svelte';
  import ScanResults from '../components/ScanResults.svelte';

  const dispatch = createEventDispatcher();
  export let initialAlertID = '';
  export let isolated = false;
  $: if (initialAlertID) { const id = initialAlertID; initialAlertID = ''; dispatch('opened'); open(id); }

  let items: service.Threat[] = [];
  let total = 0;
  let offset = 0;
  const limit = 50;
  let query = '';
  let rangeKey = '30d';
  let loading = false;
  let error = '';
  let selected: service.ThreatDetail | null = null;
  let detailLoading = false;
  let showChain = false;
  let showRecovery = false;
  let timer: ReturnType<typeof setInterval> | null = null;

  async function load() {
    loading = true;
    error = '';
    try {
      const page = await GetThreats(offset, limit, query, rangeKey);
      items = page.items ?? [];
      total = page.total;
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  }

  async function open(id: string) {
    detailLoading = true;
    showChain = false;
    try {
      selected = await GetThreatDetail(id);
      await MarkRead([id]);
      const item = items.find((i) => i.id === id);
      if (item) item.read = true;
    } catch (e) {
      dispatch('toast', { message: String(e), error: true });
    } finally {
      detailLoading = false;
    }
  }

  async function trust() {
    if (!selected) return;
    let kind = '';
    let value = '';
    if (selected.sha256) {
      kind = 'hash';
      value = selected.sha256;
    } else if (selected.processPath) {
      kind = 'path';
      value = selected.processPath;
    } else if (selected.filePath) {
      kind = 'path';
      value = selected.filePath;
    }
    if (!value) {
      dispatch('toast', { message: '这条告警没有可用的哈希或路径', error: true });
      return;
    }
    if (!confirm(`把 ${value} 加入本机策略的信任列表？`)) return;
    try {
      await CreateTrust({ kind, value, description: `从告警 ${selected.id} 创建` });
      dispatch('toast', { message: '信任项已创建，等待 agent 拉取', error: false });
    } catch (e) {
      dispatch('toast', { message: String(e), error: true });
    }
  }

  function copy(text: string) {
    navigator.clipboard?.writeText(text).then(
      () => dispatch('toast', { message: '已复制', error: false }),
      () => dispatch('toast', { message: '复制失败', error: true })
    );
  }

  function search() {
    offset = 0;
    load();
  }

  function prev() {
    if (offset >= limit) {
      offset -= limit;
      load();
    }
  }

  function next() {
    if (offset + limit < total) {
      offset += limit;
      load();
    }
  }

  onMount(() => {
    load();
    timer = setInterval(load, 30000);
  });

  onDestroy(() => {
    if (timer) clearInterval(timer);
  });
</script>

<div class="toolbar">
  <select class="range" bind:value={rangeKey} on:change={search}>
    <option value="24h">24 小时</option>
    <option value="7d">7 天</option>
    <option value="30d">30 天</option>
    <option value="all">全部</option>
  </select>
  <input class="search" placeholder="搜索规则、进程、文件路径" bind:value={query} on:keydown={(e) => e.key === 'Enter' && search()} />
  <button class="btn" on:click={search}>搜索</button>
  <div class="spacer"></div>
  <span class="muted count">共 {total} 条</span>
  <button class="btn ghost" on:click={load}>刷新</button>
</div>

{#if error}
  <div class="error">{error}</div>
{/if}

<div class="layout" class:detail={!!selected}>
  <section class="panel list">
    {#if loading && items.length === 0}
      <div class="empty">正在读取告警...</div>
    {:else if items.length === 0}
      <div class="empty">没有告警记录</div>
    {:else}
      <table class="grid">
        <thead>
          <tr>
            <th style="width: 22px"></th>
            <th style="width: 132px">时间</th>
            <th style="width: 64px">级别</th>
            <th>事件</th>
            <th style="width: 190px">进程 / 文件</th>
            <th style="width: 96px">结果</th>
          </tr>
        </thead>
        <tbody>
          {#each items as item}
            <tr class="clickable" class:active={selected?.id === item.id} on:click={() => open(item.id)}>
              <td><span class="unread" class:on={!item.read}></span></td>
              <td class="mono time">{formatTime(item.time)}</td>
              <td><span class="badge {severityClass(item.riskScore || item.severity)}">{severityLabel(item.riskScore || item.severity)}</span></td>
              <td>
                <div class="title">{item.ruleName || item.message}</div>
                {#if item.ruleName && item.message !== item.ruleName}
                  <div class="sub">{item.message}</div>
                {/if}
              </td>
              <td>
                <div class="mono ellipsis">{item.processName || '—'}</div>
                {#if item.filePath}
                  <div class="sub ellipsis">{item.filePath}</div>
                {:else if item.processPath}
                  <div class="sub ellipsis">{item.processPath}</div>
                {/if}
              </td>
              <td>
                {#if item.blocked}
                  <span class="badge danger">{actionLabel(item.action)}</span>
                {:else}
                  <span class="badge warn">仅检测</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>

      <div class="pager">
        <button class="btn ghost sm" disabled={offset === 0} on:click={prev}>上一页</button>
        <span class="muted">{offset + 1} - {Math.min(offset + limit, total)}</span>
        <button class="btn ghost sm" disabled={offset + limit >= total} on:click={next}>下一页</button>
      </div>
    {/if}
  </section>

  {#if selected}
    <aside class="panel detail">
      <div class="panel-head">
        <div class="panel-title">告警详情</div>
        <button class="btn ghost sm" on:click={() => { selected = null; showChain = false; }}>关闭</button>
      </div>
      {#if detailLoading}
        <div class="empty">正在读取详情...</div>
      {:else}
        <div class="detail-body">
          <div class="detail-title">{selected.ruleName || selected.message}</div>
          <div class="detail-meta">
            <span class="badge {severityClass(selected.riskScore || selected.severity)}">{severityLabel(selected.riskScore || selected.severity)}</span>
            <span class="muted mono">{selected.id}</span>
          </div>
          <div class="detail-time">{formatTime(selected.time)} · {relative(selected.time)}</div>

          {#if selected.blocked}
            <div class="verdict blocked">已处置：{actionLabel(selected.action)}</div>
          {:else}
            <div class="verdict detected">仅检测到，未执行处置动作</div>
          {/if}

          <div class="section">
            <div class="section-title">进程</div>
            <dl>
              <dt>名称</dt><dd class="mono">{selected.processName || '—'}</dd>
              <dt>路径</dt><dd class="mono break">{selected.processPath || '—'}</dd>
              <dt>命令行</dt><dd class="mono break">{selected.commandLine || '—'}</dd>
              <dt>工作目录</dt><dd class="mono break">{selected.workingDir || '—'}</dd>
              <dt>父进程</dt><dd class="mono break">{selected.parentName || '—'} {selected.parentPath ? '· ' + selected.parentPath : ''}</dd>
              <dt>用户</dt><dd>{selected.user || '—'}</dd>
            </dl>
          </div>

          {#if selected.sha256}
            <div class="section">
              <div class="section-title">SHA256</div>
              <button type="button" class="hash" on:click={() => selected && copy(selected.sha256)} title="点击复制">{selected.sha256}</button>
            </div>
          {/if}

          {#if selected.responses?.length}
            <div class="section">
              <div class="section-title">响应动作</div>
              {#each selected.responses as r}
                <div class="response">
                  <span class="mono">{r.action}</span>
                  <span class="muted">{r.message || '—'}</span>
                </div>
              {/each}
            </div>
          {/if}

          {#if selected.tactics?.length || selected.techniques?.length}
            <div class="section">
              <div class="section-title">MITRE ATT&CK</div>
              <div class="chips">
                {#each selected.tactics as t}<span class="badge info">{t}</span>{/each}
                {#each selected.techniques as t}<span class="badge">{t}</span>{/each}
              </div>
            </div>
          {/if}

          <div class="detail-actions">
            {#if selected.quarantineFile?.recoverable}<button class="btn" on:click={() => showRecovery = true}>从隔离区找回</button>{/if}
            <button class="btn primary" on:click={() => (showChain = true)}>EDR 进程链</button>
            <button class="btn" on:click={trust}>加入信任</button>
            <button class="btn ghost" on:click={() => dispatch('toast', { message: '请在 Kibana 中打开该告警查看完整上下文', error: false })}>Kibana 中查看</button>
          </div>
        </div>
      {/if}
    </aside>
  {/if}

  {#if showChain && selected}
    <ProcessChain alertId={selected.id} on:close={() => (showChain = false)} />
  {/if}
</div>

{#if showRecovery && selected?.quarantineFile}<ScanResults file={selected.quarantineFile} {isolated} on:close={() => showRecovery = false} />{/if}

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 14px;
  }

  .search {
    width: 320px;
  }

  .range {
    width: 104px;
  }

  .count {
    font-size: 12px;
  }

  .layout {
    display: grid;
    grid-template-columns: 1fr;
    gap: 16px;
    align-items: start;
  }

  .layout.detail {
    grid-template-columns: minmax(0, 1fr) 440px;
  }

  .list {
    overflow: hidden;
  }

  table.grid td {
    font-size: 12px;
  }

  tr.active td {
    background: var(--row-active);
  }

  .unread {
    display: block;
    width: 6px;
    height: 6px;
    border-radius: 50%;
  }

  .unread.on {
    background: var(--accent);
  }

  .time {
    color: var(--muted);
    white-space: nowrap;
  }

  .title {
    font-size: 13px;
    color: var(--text);
  }

  .sub {
    margin-top: 2px;
    font-size: 11px;
    color: var(--muted-2);
  }

  .ellipsis {
    max-width: 220px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .pager {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 12px;
    padding: 12px 16px;
    border-top: 1px solid var(--border-soft);
    font-size: 12px;
  }

  .detail {
    position: sticky;
    top: 0;
    max-height: calc(100vh - 150px);
    overflow: auto;
  }

  .detail-body {
    padding: 18px;
  }

  .detail-title {
    font-size: 15px;
    font-weight: 600;
    line-height: 1.5;
  }

  .detail-meta {
    margin-top: 8px;
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 11px;
  }

  .detail-time {
    margin-top: 6px;
    font-size: 12px;
    color: var(--muted-2);
  }

  .verdict {
    margin-top: 16px;
    padding: 10px 12px;
    border-radius: 6px;
    font-size: 12px;
  }

  .verdict.blocked {
    background: var(--danger-bg);
    border: 1px solid var(--danger-border);
    color: var(--danger);
  }

  .verdict.detected {
    background: var(--warn-bg);
    border: 1px solid var(--warn-border);
    color: var(--warn);
  }

  .section {
    margin-top: 20px;
  }

  .section-title {
    font-size: 11px;
    color: var(--muted-2);
    text-transform: uppercase;
    letter-spacing: 0.6px;
    margin-bottom: 8px;
  }

  dl {
    margin: 0;
    display: grid;
    grid-template-columns: 74px 1fr;
    gap: 6px 10px;
    font-size: 12px;
  }

  dt {
    color: var(--muted-2);
  }

  dd {
    margin: 0;
    color: var(--text-soft);
  }

  .break {
    word-break: break-all;
  }

  .hash {
    display: block;
    width: 100%;
    padding: 0;
    border: 0;
    background: transparent;
    text-align: left;
    font-family: var(--mono);
    font-size: 11px;
    color: var(--accent);
    word-break: break-all;
    cursor: pointer;
    line-height: 1.6;
  }

  .response {
    display: flex;
    justify-content: space-between;
    gap: 10px;
    font-size: 12px;
    padding: 6px 0;
    border-bottom: 1px solid var(--border-soft);
  }

  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .detail-actions {
    margin-top: 22px;
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
  }

  .error {
    margin-bottom: 14px;
    padding: 10px 12px;
    border-radius: 6px;
    background: var(--danger-bg);
    border: 1px solid var(--danger-border);
    color: var(--danger);
    font-size: 12px;
  }

  @media (max-width: 1180px) {
    .layout.detail {
      grid-template-columns: 1fr;
    }

    .detail {
      position: static;
      max-height: none;
    }
  }
</style>

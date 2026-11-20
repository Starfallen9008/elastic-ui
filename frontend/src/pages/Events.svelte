<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import { GetEvents } from '../../wailsjs/go/main/App.js';
  import type { service } from '../../wailsjs/go/models';
  import { formatTime, relative } from '../lib/format';

  const dispatch = createEventDispatcher();

  let items: service.EventItem[] = [];
  let total = 0;
  let offset = 0;
  const limit = 100;
  let rangeKey = '24h';
  let category = 'all';
  let query = '';
  let loading = false;
  let error = '';
  let selected: service.EventItem | null = null;
  let timer: ReturnType<typeof setInterval> | null = null;

  async function load() {
    loading = true;
    error = '';
    try {
      const page = await GetEvents(offset, limit, rangeKey, category, query);
      items = page.items ?? [];
      total = page.total;
      if (selected && !items.some((item) => item.id === selected?.id)) {
        selected = null;
      }
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  }

  function search() {
    offset = 0;
    selected = null;
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

  function categoryClass(value: string): string {
    if (value.includes('security')) return 'danger';
    if (value.includes('file')) return 'warn';
    if (value.includes('network')) return 'info';
    if (value.includes('process')) return 'ok';
    return '';
  }

  function pretty(raw: string): string {
    try {
      return JSON.stringify(JSON.parse(raw), null, 2);
    } catch {
      return raw;
    }
  }

  function copyRaw() {
    if (!selected) return;
    navigator.clipboard?.writeText(selected.raw).then(
      () => dispatch('toast', { message: '已复制原始事件', error: false }),
      () => dispatch('toast', { message: '复制失败', error: true })
    );
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
    <option value="1h">1 小时</option>
    <option value="24h">24 小时</option>
    <option value="7d">7 天</option>
    <option value="30d">30 天</option>
  </select>
  <select class="category" bind:value={category} on:change={search}>
    <option value="all">全部类别</option>
    <option value="process">进程</option>
    <option value="file">文件</option>
    <option value="network">网络</option>
    <option value="registry">注册表</option>
    <option value="security">安全</option>
  </select>
  <input class="search" placeholder="搜索进程、路径、目标 IP、用户" bind:value={query} on:keydown={(e) => e.key === 'Enter' && search()} />
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
      <div class="empty">正在读取事件...</div>
    {:else if items.length === 0}
      <div class="empty">没有匹配的事件</div>
    {:else}
      <table class="grid">
        <thead>
          <tr>
            <th style="width: 132px">时间</th>
            <th style="width: 82px">类别</th>
            <th style="width: 90px">动作</th>
            <th style="width: 170px">进程</th>
            <th>路径 / 目标</th>
            <th style="width: 96px">用户</th>
          </tr>
        </thead>
        <tbody>
          {#each items as item}
            <tr class="clickable" class:active={selected?.id === item.id} on:click={() => (selected = item)}>
              <td class="mono time">{formatTime(item.time)}</td>
              <td><span class="badge {categoryClass(item.category)}">{item.category || '—'}</span></td>
              <td class="muted">{item.action || '—'}</td>
              <td>
                <div class="mono ellipsis">{item.processName || '—'}</div>
                {#if item.processPid}<div class="sub">PID {item.processPid}</div>{/if}
              </td>
              <td>
                {#if item.filePath}
                  <div class="mono ellipsis" title={item.filePath}>{item.filePath}</div>
                {:else if item.destIp}
                  <div class="mono">{item.destIp}{item.destPort ? ':' + item.destPort : ''} <span class="sub">{item.protocol}</span></div>
                {:else if item.processPath}
                  <div class="mono ellipsis" title={item.processPath}>{item.processPath}</div>
                {:else}
                  <span class="muted">—</span>
                {/if}
              </td>
              <td class="muted ellipsis">{item.user || '—'}</td>
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
        <div class="panel-title">事件详情</div>
        <button class="btn ghost sm" on:click={() => (selected = null)}>关闭</button>
      </div>
      <div class="detail-body">
        <div class="detail-title">{selected.processName || selected.message || '事件'}</div>
        <div class="detail-meta">
          <span class="badge {categoryClass(selected.category)}">{selected.category || '—'}</span>
          <span class="muted mono">{selected.id}</span>
        </div>
        <div class="detail-time">{formatTime(selected.time)} · {relative(selected.time)}</div>

        <dl>
          <dt>动作</dt><dd>{selected.action || '—'}</dd>
          <dt>进程</dt><dd class="mono break">{selected.processName || '—'} {selected.processPid ? '(' + selected.processPid + ')' : ''}</dd>
          <dt>进程路径</dt><dd class="mono break">{selected.processPath || '—'}</dd>
          <dt>文件</dt><dd class="mono break">{selected.filePath || '—'}</dd>
          <dt>目标</dt><dd class="mono break">{selected.destIp || '—'}{selected.destPort ? ':' + selected.destPort : ''}</dd>
          <dt>协议</dt><dd>{selected.protocol || '—'}</dd>
          <dt>用户</dt><dd>{selected.user || '—'}</dd>
        </dl>

        <div class="raw-head">
          <span>原始事件</span>
          <button class="btn ghost sm" on:click={copyRaw}>复制</button>
        </div>
        <pre class="raw">{pretty(selected.raw)}</pre>
      </div>
    </aside>
  {/if}
</div>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 14px;
  }

  .range {
    width: 104px;
  }

  .category {
    width: 118px;
  }

  .search {
    width: 320px;
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
    grid-template-columns: minmax(0, 1fr) 460px;
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

  .time {
    color: var(--muted);
    white-space: nowrap;
  }

  .sub {
    margin-top: 2px;
    font-size: 11px;
    color: var(--muted-2);
  }

  .ellipsis {
    max-width: 360px;
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

  dl {
    margin: 18px 0 0;
    display: grid;
    grid-template-columns: 74px 1fr;
    gap: 7px 10px;
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

  .raw-head {
    margin-top: 20px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 11px;
    color: var(--muted-2);
    text-transform: uppercase;
    letter-spacing: 0.6px;
  }

  .raw {
    margin: 8px 0 0;
    padding: 12px;
    max-height: 320px;
    overflow: auto;
    background: var(--cmd-bg);
    border: 1px solid var(--border-soft);
    border-radius: 6px;
    font-family: var(--mono);
    font-size: 11px;
    line-height: 1.6;
    color: var(--text-soft);
    white-space: pre-wrap;
    word-break: break-all;
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

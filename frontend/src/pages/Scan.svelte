<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import { GetScans, StartScan } from '../../wailsjs/go/main/App.js';
  import type { service } from '../../wailsjs/go/models';
  import { formatTime } from '../lib/format';
  import ScanResults from '../components/ScanResults.svelte';

  export let isolated: boolean;

  const dispatch = createEventDispatcher();

  let path = 'C:\\Users';
  let scans: service.ScanTask[] = [];
  let busy = false;
  let selected: service.ScanTask | null = null;
  let resultsOpen = false;
  let timer: ReturnType<typeof setInterval> | null = null;

  async function load() {
    try {
      scans = (await GetScans()) ?? [];
    } catch {
      // 状态轮询失败不打断界面
    }
  }

  async function start() {
    busy = true;
    try {
      await StartScan(path);
      dispatch('toast', { message: '扫描任务已提交', error: false });
      await load();
    } catch (e) {
      dispatch('toast', { message: String(e), error: true });
    } finally {
      busy = false;
    }
  }

  function statusLabel(status: string): string {
    switch (status) {
      case 'pending': return '执行中';
      case 'successful': return '已完成';
      case 'failed': return '失败';
      case 'expired': return '已过期';
      default: return status || '未知';
    }
  }

  function statusClass(status: string): string {
    if (status === 'successful') return 'ok';
    if (status === 'failed' || status === 'expired') return 'danger';
    return 'warn';
  }

  function elapsed(from: string, to: string): string {
    const a = new Date(from).getTime();
    const b = new Date(to).getTime();
    if (Number.isNaN(a) || Number.isNaN(b) || b < a) return '—';
    const sec = Math.round((b - a) / 1000);
    if (sec < 60) return `${sec} 秒`;
    const min = Math.floor(sec / 60);
    return `${min} 分 ${sec % 60} 秒`;
  }

  onMount(() => {
    load();
    timer = setInterval(load, 3000);
  });

  onDestroy(() => {
    if (timer) clearInterval(timer);
  });
</script>

<section class="panel">
  <div class="panel-head">
    <div class="panel-title">发起扫描</div>
  </div>
  <div class="panel-body">
    <div class="start-row">
      <input bind:value={path} placeholder="绝对路径，例如 C:\\Users" spellcheck="false" />
      <button class="btn primary" disabled={busy || isolated} on:click={start}>{busy ? '提交中...' : '开始扫描'}</button>
    </div>
    <div class="quick">
      <button class="btn ghost sm" disabled={isolated} on:click={() => (path = 'C:\\Users')}>用户目录</button>
      <button class="btn ghost sm" disabled={isolated} on:click={() => (path = 'C:\\Windows\\Temp')}>临时目录</button>
      <button class="btn ghost sm" disabled={isolated} on:click={() => (path = 'C:\\')}>系统盘</button>
    </div>
    {#if isolated}
      <div class="locked">本机已隔离，扫描功能不可用。</div>
    {/if}
  </div>
</section>

<section class="panel tasks">
  <div class="panel-head">
    <div class="panel-title">扫描记录</div>
    <span class="muted count">{scans.length} 条</span>
    <button class="btn ghost sm" on:click={() => { selected = null; resultsOpen = true; }}>隔离区</button>
  </div>
  {#if scans.length === 0}
    <div class="empty">还没有扫描任务</div>
  {:else}
    <div class="table-scroll"><table class="grid">
      <thead>
        <tr>
          <th style="width: 150px">开始时间</th>
          <th>路径</th>
          <th style="width: 90px">状态</th>
          <th style="width: 64px">发现</th>
          <th style="width: 100px">耗时</th>
          <th style="width: 100px">结果</th>
        </tr>
      </thead>
      <tbody>
        {#each scans as task}
          <tr>
            <td>
              <div class="mono">{formatTime(task.startedAt)}</div>
            </td>
            <td class="mono break">{task.path}</td>
            <td><span class="badge {statusClass(task.status)}">{statusLabel(task.status)}</span></td>
            <td class="muted">{task.found > 0 ? task.found : '—'}</td>
            <td class="muted">{task.completedAt ? elapsed(task.startedAt, task.completedAt) : '—'}</td>
            <td><button class="btn ghost sm" on:click={() => { selected = task; resultsOpen = true; }}>查看结果</button></td>
          </tr>
        {/each}
      </tbody>
    </table></div>
  {/if}
</section>

{#if resultsOpen}<ScanResults task={selected} {isolated} on:close={() => { resultsOpen = false; load(); }} />{/if}

<style>
  .start-row {
    display: flex;
    gap: 10px;
  }

  .start-row input {
    flex: 1;
    font-family: var(--mono);
    font-size: 12px;
  }

  .quick {
    margin-top: 12px;
    display: flex;
    gap: 8px;
  }

  .locked {
    margin-top: 14px;
    padding: 10px 12px;
    border-radius: 6px;
    background: var(--danger-bg);
    border: 1px solid var(--danger-border);
    color: var(--danger);
    font-size: 12px;
  }

  .tasks {
    margin-top: 16px;
    overflow: hidden;
  }

  .count {
    font-size: 12px;
    margin-left: auto;
  }
  .table-scroll { overflow-x: auto; }
  table { min-width: 720px; }

  .break {
    word-break: break-all;
  }
</style>

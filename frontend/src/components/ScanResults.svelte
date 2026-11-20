<script lang="ts">
  import { createEventDispatcher, onMount, tick } from 'svelte';
  import { GetScanResults, GetQuarantine, GetQuarantineFile, RecoverQuarantinedFile, RecoverQuarantinedFiles, TrustQuarantinedFile } from '../../wailsjs/go/main/App.js';
  import type { service } from '../../wailsjs/go/models';
  import { formatTime } from '../lib/format';
  export let task: service.ScanTask | null = null;
  export let file: service.ScanFile | null = null;
  export let isolated = false;
  const dispatch = createEventDispatcher();
  let result: { items: service.ScanFile[]; total: number } | null = null;
  let offset = 0;
  let loading = false;
  let error = '';
  let confirmation: { action: 'recover' | 'trust'; files: service.ScanFile[] } | null = null;
  let selected: service.ScanFile[] = [];
  let outcomes: Record<string, { message: string; failed: boolean }> = {};
  let recovering = '';
  let savedPath = '';
  let dialog: HTMLElement;
  let modal: HTMLDialogElement;
  let request = 0;
  async function load() {
    const id = ++request;
    loading = true; error = '';
    try {
      const next = file ? { items: [await GetQuarantineFile(file.id)], total: 1 } : task ? await GetScanResults(task.id, offset) : await GetQuarantine(offset);
      if (id === request) {
        result = next;
        selected = selected.map(item => next.items.find(f => f.id === item.id) || item);
      }
    } catch (e) { if (id === request) error = String(e); }
    finally { if (id === request) loading = false; }
  }
  function choose(item: service.ScanFile, checked: boolean) {
    selected = checked ? [...selected.filter(f => f.id !== item.id), item] : selected.filter(f => f.id !== item.id);
  }
  function selectPage(checked: boolean) {
    for (const item of result?.items || []) if (canSelect(item)) choose(item, checked);
  }
  function canSelect(item: service.ScanFile) { return !!item.quarantinePath && item.outcome === '已隔离'; }
  const availabilityLabels: Record<string,string> = {available:'文件在隔离区',missing:'隔离文件已不存在',unavailable:'无法读取隔离文件',unknown:'文件状态未确认'};
  function prepare(action: 'recover' | 'trust', files: service.ScanFile[]) {
    confirmation = { action, files }; savedPath = ''; error = '';
    tick().then(() => dialog.querySelector('.recovery-confirm')?.scrollIntoView({ block: 'nearest' }));
  }
  async function execute() {
    if (!confirmation || recovering) return;
    const operation = confirmation;
    recovering = operation.action; error = ''; savedPath = '';
    try {
      if (operation.action === 'trust') {
        for (const item of operation.files) {
          try { await TrustQuarantinedFile(item.id); outcomes = {...outcomes, [item.id]: { message: '已加入信任区', failed: false }}; }
          catch (e) { outcomes = {...outcomes, [item.id]: { message: String(e), failed: true }}; }
        }
      } else if (operation.files.length === 1) {
        savedPath = await RecoverQuarantinedFile(operation.files[0].id);
        if (savedPath) outcomes = {...outcomes, [operation.files[0].id]: { message: `已恢复至 ${savedPath}`, failed: false }};
      } else {
        const results = await RecoverQuarantinedFiles(operation.files.map(item => item.id));
        for (const item of results || []) outcomes = {...outcomes, [item.id]: { message: item.error || `已恢复至 ${item.path}（ZIP 密码：elastic）`, failed: !!item.error }};
      }
      confirmation = null;
      await load();
    } catch (e) { error = String(e); }
    finally { recovering = ''; }
  }
  function close() { if (!recovering) dispatch('close'); }
  function keydown(event: KeyboardEvent) {
    if (event.key === 'Escape') { event.stopPropagation(); if (recovering) return; if (confirmation) confirmation = null; else close(); }
    if (event.key === 'Tab') {
      const buttons = [...dialog.querySelectorAll<HTMLElement>('button:not(:disabled), input:not(:disabled), [tabindex="0"]')];
      const first = buttons[0], last = buttons[buttons.length - 1];
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
    }
  }
  async function page(step: number) { offset += step * 50; await load(); }
  onMount(() => {
    const previous = document.activeElement as HTMLElement;
    modal.showModal();
    load(); tick().then(() => dialog.focus());
    return () => { request++; previous?.focus(); };
  });
</script>

<svelte:window on:keydown={keydown} />
<dialog class="result-overlay" bind:this={modal} aria-labelledby="scan-result-title" on:cancel|preventDefault={close} on:click={(e) => { if (e.target === e.currentTarget) close(); }}>
  <div class="result-dialog" tabindex="-1" bind:this={dialog}>
    <header>
      <div><h2 id="scan-result-title">{task ? '扫描结果' : '隔离区'}</h2>{#if task}<div class="scan-path mono">{task.path}</div><div class="scan-meta">{formatTime(task.startedAt)}{task.completedAt ? ` 至 ${formatTime(task.completedAt)}` : '，扫描进行中'}</div>{/if}</div>
      <button class="btn ghost sm" disabled={!!recovering} aria-label="关闭扫描结果" on:click={close}>×</button>
    </header>
    <div class="result-body" aria-busy={loading || !!recovering}>
      <div class="result-toolbar"><span>{result ? `${result.total} 条文件记录` : '文件记录'}</span><button class="btn ghost sm" disabled={loading || !!recovering} on:click={load}>刷新</button></div>
      {#if result?.items.some(canSelect)}
        <div class="selection-toolbar">
          <label><input type="checkbox" aria-label="全选本页" checked={result.items.filter(canSelect).every(f => selected.some(s => s.id === f.id))} disabled={!!recovering} on:change={(e) => selectPage(e.currentTarget.checked)} />全选本页</label>
          <span>已选 {selected.length} 个</span>
          <div class="batch-actions"><button class="btn ghost sm" disabled={!selected.some(f => f.recoverable) || !!recovering} on:click={() => prepare('recover', selected.filter(f => f.recoverable))}>批量恢复</button><button class="btn ghost sm" disabled={!selected.some(f => f.sha256 && f.trustStatus !== 'configured') || !!recovering || isolated} on:click={() => prepare('trust', selected.filter(f => f.sha256 && f.trustStatus !== 'configured'))}>批量加白</button></div>
        </div>
      {/if}
      {#if task?.detail}<div class="error" role="status">{task.detail}</div>{/if}
      {#if error}<div class="error" role="alert">{error}</div>{/if}
      {#if savedPath}<div class="saved" role="status">已保存到 <span class="mono">{savedPath}</span>。ZIP 密码：elastic。</div>{/if}
      {#if confirmation}
        <div class="recovery-confirm" role="alert">
          <strong>{confirmation.action === 'recover' ? `恢复 ${confirmation.files.length} 个文件？` : `将 ${confirmation.files.length} 个文件加入信任区？`}</strong>
          {#if confirmation.files.length === 1}<div class="mono">{confirmation.files[0].path}</div>{/if}
          {#if confirmation.action === 'recover'}<p>保存为 ZIP，密码：elastic</p>{/if}
          <div class="actions"><button class="btn ghost sm" disabled={!!recovering} on:click={() => confirmation = null}>取消</button><button class="btn primary sm" disabled={!!recovering} on:click={execute}>{recovering ? '正在处理…' : confirmation.action === 'trust' ? '确认加白' : '选择保存位置'}</button></div>
        </div>
      {/if}
      {#if loading && !result}<div class="empty" role="status">正在读取文件记录…</div>
      {:else if result}
        {#if !result.items.length}<div class="empty">{task ? '暂未查询到关联的威胁文件' : '没有隔离文件记录'}</div>{/if}
        <div class="file-list">
          {#each result.items as file (file.id)}
            <article class="file-row">
               <div class="file-head"><label>{#if canSelect(file)}<input type="checkbox" aria-label={`选择 ${file.path}`} checked={selected.some(item => item.id === file.id)} disabled={!!recovering} on:change={(e) => choose(file, e.currentTarget.checked)} />{/if}<span class="outcome" class:failed={file.outcome === '隔离失败'}>{file.outcome}</span></label><time>{formatTime(file.time)}</time></div>
              <div class="file-path mono">{file.path || '未记录文件路径'}</div>
              <div class="file-name">{file.malwareName || '恶意文件检测（未提供家族名称）'}</div>
              {#if file.detail && file.detail !== 'Success'}<div class="file-detail">{file.detail}</div>{/if}
              <dl><dt>SHA256</dt><dd class="mono">{file.sha256 || '—'}</dd><dt>隔离路径</dt><dd class="mono">{file.quarantinePath || '—'}</dd></dl>
              {#if file.availability}<div class="file-status"><span>{availabilityLabels[file.availability] || '文件状态未确认'}</span><span>{file.trustStatus === 'configured' ? '已配置信任' : file.trustStatus === 'unknown' ? '信任状态读取失败' : '未加白'}</span></div>{/if}
              {#if file.recoveredAt}<dl><dt>恢复状态</dt><dd>ZIP 已保存</dd><dt>恢复时间</dt><dd>{formatTime(file.recoveredAt)}</dd><dt>保存位置</dt><dd class="mono">{file.savedPath}</dd></dl>{/if}
              {#if file.recoveryError}<div class="error">{file.recoveryError}</div>{/if}
              {#if outcomes[file.id]}<div class:error={outcomes[file.id].failed} class:saved={!outcomes[file.id].failed} role="status">{outcomes[file.id].message}</div>{/if}
              {#if canSelect(file)}<div class="actions"><button class="btn ghost sm" disabled={!!recovering || !file.recoverable} on:click={() => prepare('recover', [file])}>{file.recoveredAt ? '再次恢复' : '恢复'}</button><button class="btn ghost sm" disabled={!!recovering || !file.sha256 || isolated || file.trustStatus === 'configured'} on:click={() => prepare('trust', [file])}>{file.trustStatus === 'configured' ? '已加白' : '加白'}</button></div>{/if}
            </article>
          {/each}
        </div>
        {#if result.total > 50}<div class="pagination"><button class="btn ghost sm" disabled={offset === 0 || loading || !!recovering} on:click={() => page(-1)}>上一页</button><span>{offset + 1}–{Math.min(offset + 50, result.total)} / {result.total}</span><button class="btn ghost sm" disabled={offset + 50 >= result.total || offset >= 9900 || loading || !!recovering} on:click={() => page(1)}>下一页</button></div>{/if}
      {/if}
    </div>
  </div>
</dialog>

<style>
  .result-overlay { position: fixed; inset: 0; width: 100vw; max-width: none; height: 100dvh; max-height: none; margin: 0; padding: 24px; border: 0; background: transparent; color: var(--text); --wails-draggable: no-drag; }
  .result-overlay[open] { display: grid; place-items: center; }
  .result-overlay::backdrop { background: var(--backdrop); }
  .result-dialog { width: min(850px, 100%); max-height: 88vh; display: flex; flex-direction: column; background: var(--dialog-panel); border: 1px solid var(--border-strong); border-radius: 8px; box-shadow: var(--shadow); overflow: hidden; outline: none; }
  header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; padding: 20px 24px; border-bottom: 1px solid var(--border); }
  h2 { margin: 0; font-size: 17px; font-weight: 600; }
  header .btn { font-size: 22px; line-height: 1; flex: none; }
  .scan-path { margin-top: 10px; overflow-wrap: anywhere; font-size: 12px; }
  .scan-meta, time { color: var(--muted); font-size: 12px; line-height: 1.7; }
  .scan-meta { margin-top: 6px; }
  .result-body { overflow: auto; padding: 16px 24px 24px; }
  .result-toolbar, .file-head, .pagination { display: flex; justify-content: space-between; align-items: center; gap: 12px; font-size: 13px; }
  .file-list { margin-top: 12px; }
  .file-row { padding: 18px 0; border-top: 1px solid var(--border); }
  .outcome { color: var(--ok); font-size: 12px; font-weight: 600; }
  .outcome.failed { color: var(--danger); }
  .file-path { overflow-wrap: anywhere; margin-top: 8px; font-size: 13px; line-height: 1.7; }
  .file-name, .file-detail { margin-top: 8px; font-size: 13px; color: var(--muted); overflow-wrap: anywhere; }
  dl { display: grid; grid-template-columns: 72px minmax(0, 1fr); gap: 10px; padding: 10px 0; font-size: 12px; color: var(--muted); }
  dd { margin: 0; overflow-wrap: anywhere; }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 12px; }
  label { display: flex; align-items: center; gap: 8px; }
  input[type="checkbox"] { width: 15px; height: 15px; margin: 0; accent-color: var(--accent); }
  .selection-toolbar { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; padding: 14px 0 4px; font-size: 12px; }
  .batch-actions { display: flex; gap: 8px; margin-left: auto; }
  .file-status { display: flex; flex-wrap: wrap; gap: 16px; font-size: 12px; color: var(--text-soft); }
  .recovery-confirm { margin-top: 16px; padding: 16px; border: 1px solid var(--warn-border); background: var(--warn-bg); border-radius: 6px; font-size: 13px; line-height: 1.7; }
  .recovery-confirm .mono { overflow-wrap: anywhere; margin-top: 8px; }
  .error, .saved { margin-top: 12px; font-size: 13px; line-height: 1.7; overflow-wrap: anywhere; }
  .error { color: var(--danger); } .saved { color: var(--ok); }
  .pagination { margin-top: 12px; }
  @media (max-width: 600px) { .result-overlay { padding: 8px; } .result-dialog { max-height: 96vh; } header { padding: 16px; } .result-body { padding: 12px 16px; } }
</style>

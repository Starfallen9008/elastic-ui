<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { CreateTrust, DeleteTrust, GetTrustItems } from '../../wailsjs/go/main/App.js';
  import type { service } from '../../wailsjs/go/models';
  import { formatTime } from '../lib/format';
  import { BATCH_LIMIT, kindLabels, parseRules, ruleKey, runSequential, validateRule } from '../lib/trust';
  import type { OperationResult, TrustKind } from '../lib/trust';

  export let isolated: boolean;
  const dispatch = createEventDispatcher();
  let items: service.TrustItem[] = [];
  let loading = false;
  let busy = false;
  let kind: TrustKind = 'path';
  let value = '';
  let description = '';
  let error = '';
  let mode: 'single' | 'batch' = 'single';
  let batchText = '';
  let selected = new Set<string>();
  let result: OperationResult | null = null;
  let confirmation: { action: 'add' | 'delete'; values: string[]; skipped: number } | null = null;
  let completed = 0;
  let total = 0;
  $: dispatch('busy', busy);

  // matchKey is supplied only for equivalent single-condition Windows rules;
  // compound rules must not be mistaken for an unconditional path/hash trust.
  $: existing = new Set(items.map(item => item.matchKey).filter(Boolean));
  $: parsed = parseRules(batchText, kind, existing);
  $: valid = parsed.filter(row => row.state === 'valid');
  $: invalid = parsed.filter(row => row.state === 'invalid');
  $: duplicates = parsed.filter(row => row.state === 'duplicate');
  $: editable = items.filter(item => !item.readOnly);
  $: allSelected = editable.length > 0 && editable.every(item => selected.has(item.id));
  $: selectedCount = editable.filter(item => selected.has(item.id)).length;

  async function load(): Promise<boolean> {
    loading = true;
    try {
      items = (await GetTrustItems()) ?? [];
      selected = new Set([...selected].filter(id => items.some(item => item.id === id && !item.readOnly)));
      error = '';
      return true;
    } catch (e) {
      error = `读取信任区失败：${String(e)}`;
      return false;
    } finally { loading = false; }
  }

  async function create() {
    if (busy || loading || isolated) return;
    const input = value.trim();
    error = validateRule(kind, input);
    if (error) return;
    busy = true;
    total = 0;
    result = null;
    try {
      if (!await load()) return;
      if (items.some(item => item.matchKey === ruleKey(kind, input))) { error = '规则已存在（含全局项），未重复创建'; return; }
      await CreateTrust({ kind, value: input, description: description.trim() });
      value = '';
      description = '';
      dispatch('toast', { message: '信任项已创建，等待 Agent 拉取', error: false });
      await load();
    } catch (e) { error = String(e); }
    finally { busy = false; }
  }

  async function prepareAdd() {
    if (busy || loading || isolated) return;
    busy = true;
    total = 0;
    result = null;
    try {
      if (!await load()) return;
      const rows = parseRules(batchText, kind, new Set(items.map(item => item.matchKey).filter(Boolean)));
      if (rows.length > BATCH_LIMIT) { error = `每批最多 ${BATCH_LIMIT} 条，请分批提交`; return; }
      if (rows.some(row => row.state === 'invalid')) { error = '请修正无效规则后再提交'; return; }
      const values = rows.filter(row => row.state === 'valid').map(row => row.value);
      const skipped = rows.filter(row => row.state === 'duplicate').length;
      if (!values.length) { result = { action: 'add', success: 0, skipped, failures: [] }; return; }
      confirmation = { action: 'add', values, skipped };
    } finally { busy = false; }
  }

  function prepareDelete(ids: string[]) {
    if (busy || loading || isolated) return;
    const values = ids.filter(id => items.some(item => item.id === id && !item.readOnly));
    if (values.length) confirmation = { action: 'delete', values, skipped: 0 };
  }

  async function execute() {
    if (!confirmation || busy || isolated) return;
    const operation = confirmation;
    confirmation = null;
    const inputKind = kind;
    const inputDescription = description.trim();
    busy = true;
    result = null;
    completed = 0;
    total = operation.values.length;
    const names = new Map(items.map(item => [item.id, item.value || item.name]));
    try {
      const outcome = await runSequential(operation.values, async entry => {
        if (isolated) throw new Error('本机已隔离，停止写入');
        if (operation.action === 'add') await CreateTrust({ kind: inputKind, value: entry, description: inputDescription });
        else {
          await DeleteTrust(entry);
          // Reflect confirmed deletions even if the final refresh fails.
          items = items.filter(item => item.id !== entry);
          selected = new Set([...selected].filter(id => id !== entry));
        }
      }, entry => operation.action === 'add' ? entry : names.get(entry) || entry,
      operation.action, done => completed = done);
      outcome.skipped += operation.skipped;
      result = outcome;
      await load();
    } finally { busy = false; }
  }

  function toggle(id: string, checked: boolean) {
    const next = new Set(selected);
    if (checked) next.add(id); else next.delete(id);
    selected = next;
  }

  async function copyFailures() {
    try {
      await navigator.clipboard.writeText(result!.failures.map(item => `${item.value}\t${item.reason}`).join('\r\n'));
      dispatch('toast', { message: '已复制失败列表', error: false });
    } catch { dispatch('toast', { message: '复制失败', error: true }); }
  }

  function focusConfirmation(element: HTMLElement) {
    element.focus({ preventScroll: true });
    element.scrollIntoView({ block: 'nearest' });
  }

  onMount(load);
</script>

<section class="panel trust-editor" aria-label="新增本机信任规则">
  <div class="panel-head">
    <div class="panel-title">新增本机信任规则</div>
    <div class="editor-modes" role="group" aria-label="添加方式">
      <button class="btn sm" class:current={mode === 'single'} aria-pressed={mode === 'single'} disabled={busy || !!confirmation} on:click={() => mode = 'single'}>单条添加</button>
      <button class="btn sm" class:current={mode === 'batch'} aria-pressed={mode === 'batch'} disabled={busy || !!confirmation} on:click={() => mode = 'batch'}>批量添加</button>
    </div>
  </div>
  <fieldset disabled={busy || !!confirmation}>
    <div class="panel-body form">
      <div class="field">
        <label for="trust-kind">条件类型</label>
        <select id="trust-kind" bind:value={kind}>
          <option value="path">可执行文件路径</option><option value="wildcard">路径通配</option><option value="hash">SHA256</option>
        </select>
      </div>
      {#if mode === 'single'}
        <div class="field grow">
          <label for="trust-value">条件值</label>
          <input id="trust-value" bind:value={value} placeholder={kind === 'hash' ? '64 位 SHA256' : kind === 'wildcard' ? 'D:\\Tools\\*' : 'C:\\Program Files\\App\\app.exe'} spellcheck="false" />
        </div>
      {/if}
      <div class="field grow"><label for="trust-desc">{mode === 'batch' ? '统一说明（可选）' : '说明（可选）'}</label><input id="trust-desc" bind:value={description} /></div>
      {#if mode === 'single'}<button class="btn primary" disabled={busy || loading || isolated} on:click={create}>添加</button>{/if}
    </div>
    {#if mode === 'batch'}
      <div class="batch-input">
        <label for="batch-values">每行一条{kindLabels[kind]}，每批最多 {BATCH_LIMIT} 条</label>
        <textarea id="batch-values" rows="5" bind:value={batchText} spellcheck="false" placeholder={kind === 'hash' ? '粘贴 SHA256，每行一条' : '粘贴 Windows 路径，每行一条'}></textarea>
        <div class="batch-summary" aria-live="polite"><span>有效 {valid.length} 条</span><span>重复 {duplicates.length} 条</span><span class:invalid={invalid.length > 0}>无效 {invalid.length} 条</span></div>
        {#if parsed.length}
          <div class="preview" aria-label="规则预览">
            {#each parsed as row}<div class="preview-row"><span class="line">{row.line}</span><span class="mono">{row.value}</span><span class:invalid={row.state === 'invalid'}>{row.reason || '待添加'}</span></div>{/each}
          </div>
        {/if}
        <button class="btn primary" disabled={busy || loading || isolated || !parsed.length || !!invalid.length || parsed.length > BATCH_LIMIT} on:click={prepareAdd}>检查并确认添加</button>
      </div>
    {/if}
  </fieldset>
</section>

{#if isolated}<div class="notice">本机已隔离，信任规则只读</div>{/if}
{#if error}<div class="error" role="alert">{error}</div>{/if}
{#if busy && total > 0}<div class="notice" role="status">正在处理 {completed} / {total}，请勿关闭或离开页面</div>{/if}
{#if result}
  <section class="operation-result" role="status" aria-label="操作结果">
    <div class="result-summary">
      <strong>{result.action === 'add' ? '成功添加' : '成功删除'} {result.success} 条</strong>
      {#if result.action === 'add'}<span>重复跳过 {result.skipped} 条</span>{/if}
      <span class:invalid={result.failures.length > 0}>失败 {result.failures.length} 条</span>
      {#if result.failures.length}<button class="btn sm" on:click={copyFailures}>复制失败列表</button>{/if}
    </div>
    {#if result.failures.length}<ul>{#each result.failures as failure}<li><span class="mono">{failure.value}</span><span>{failure.reason}</span></li>{/each}</ul>{/if}
  </section>
{/if}

<section class="panel list" aria-label="对本机生效的信任规则">
  <div class="panel-head"><div class="panel-title">对本机生效的信任规则 <span class="muted">· {items.length} 条</span></div><button class="btn sm" disabled={loading || busy || !!confirmation} on:click={load}>{loading ? '读取中…' : '刷新'}</button></div>
  <div class="selection-toolbar">
    <span>{selectedCount ? `已选 ${selectedCount} 条` : '选择当前列表中的本机策略项'}</span>
    <button class="btn danger sm" disabled={!selectedCount || busy || loading || isolated || !!confirmation} on:click={() => prepareDelete([...selected])}>批量删除</button>
    {#if selectedCount}<button class="btn ghost sm" disabled={busy} on:click={() => selected = new Set()}>取消选择</button>{/if}
  </div>
  {#if loading && !items.length}<div class="empty">正在读取…</div>
  {:else if !items.length}<div class="empty">{error ? '读取失败，请刷新重试' : '还没有信任项'}</div>
  {:else}
    <div class="table-scroll"><table class="grid">
      <thead><tr>
        <th class="check-cell"><input type="checkbox" aria-label="全选当前列表可编辑项" checked={allSelected} indeterminate={selectedCount > 0 && !allSelected} disabled={busy || isolated || !editable.length || !!confirmation} on:change={(e) => selected = e.currentTarget.checked ? new Set(editable.map(item => item.id)) : new Set()} /></th>
        <th>条件</th><th>类型</th><th>作用域</th><th>系统</th><th>更新时间</th><th>操作</th>
      </tr></thead>
      <tbody>{#each items as item (item.id)}<tr class:selected={selected.has(item.id)}>
        <td class="check-cell"><input type="checkbox" aria-label={`选择 ${item.value || item.name}`} checked={selected.has(item.id)} disabled={item.readOnly || busy || isolated || !!confirmation} on:change={(e) => toggle(item.id, e.currentTarget.checked)} /></td>
        <td><div class="mono break">{item.value || item.name}</div>{#if item.description}<div class="sub">{item.description}</div>{/if}</td>
        <td class="nowrap">{item.kind}</td><td class="nowrap">{item.scope}</td><td>{item.os || '—'}</td><td class="nowrap muted">{formatTime(item.updatedAt)}</td>
        <td>{#if item.readOnly}<span class="readonly">全局只读</span>{:else}<button class="btn ghost sm" disabled={isolated || busy || loading || !!confirmation} on:click={() => prepareDelete([item.id])}>删除</button>{/if}</td>
      </tr>{/each}</tbody>
    </table></div>
  {/if}
</section>

{#if confirmation}
  <div class="confirm-panel" role="alertdialog" aria-modal="false" aria-labelledby="trust-confirm-title" tabindex="-1" use:focusConfirmation on:keydown={(event) => { if (event.key === 'Escape') confirmation = null; }}>
    <strong id="trust-confirm-title">{confirmation.action === 'add' ? `确认添加 ${confirmation.values.length} 条${kindLabels[kind]}？` : `确认删除 ${confirmation.values.length} 条本机策略规则？`}</strong>
    <p>{confirmation.action === 'add' ? `重复跳过 ${confirmation.skipped} 条。以下规则将作为信任应用写入本机策略。` : '删除后将不再信任这些规则，等待 Agent 拉取生效。全局项不会被删除。'}</p>
    <ul>{#each confirmation.values as entry}<li class="mono">{confirmation.action === 'add' ? entry : items.find(item => item.id === entry)?.value || entry}</li>{/each}</ul>
    <div class="row"><button class="btn" on:click={() => confirmation = null}>取消</button><button class="btn" class:danger={confirmation.action === 'delete'} class:primary={confirmation.action === 'add'} disabled={isolated || busy} on:click={execute}>确认{confirmation.action === 'add' ? '添加' : '删除'}</button></div>
  </div>
{/if}

<style>
  fieldset { margin: 0; padding: 0; border: 0; min-width: 0; }
  .form { display: flex; gap: 16px; align-items: flex-end; flex-wrap: wrap; }
  .form select { min-width: 164px; }
  .grow { flex: 1; min-width: 200px; }
  .editor-modes { display: flex; gap: 4px; }
  .current { border-color: var(--accent); color: var(--accent); background: var(--accent-dim); }
  .batch-input { padding: 0 18px 16px; display: flex; flex-direction: column; gap: 12px; align-items: flex-start; }
  .batch-input textarea { width: 100%; resize: vertical; min-height: 112px; font-family: var(--font-mono); font-size: 13px; }
  .batch-input label, .sub, .readonly { font-size: 13px; color: var(--muted); }
  .batch-summary, .result-summary, .selection-toolbar { display: flex; gap: 16px; align-items: center; flex-wrap: wrap; font-size: 13px; }
  .preview { width: 100%; max-height: 200px; overflow: auto; border: 1px solid var(--border); }
  .preview-row { display: grid; grid-template-columns: 32px minmax(0, 1fr) minmax(140px, 28%); gap: 12px; padding: 8px 12px; border-bottom: 1px solid var(--border); font-size: 13px; overflow-wrap: anywhere; }
  .line { color: var(--muted); font-variant-numeric: tabular-nums; }
  .invalid, .error { color: var(--danger); }
  .error, .notice, .operation-result { margin-top: 16px; padding: 12px 16px; border: 1px solid var(--border); border-radius: 6px; background: var(--panel); }
  .error { border-color: var(--danger-border); background: var(--danger-bg); }
  .operation-result strong { font-weight: 500; }
  .operation-result ul { max-height: 220px; overflow: auto; padding-left: 20px; }
  .operation-result li { padding: 4px; overflow-wrap: anywhere; }
  .operation-result li span { display: block; }
  .list { margin-top: 16px; }
  .selection-toolbar { padding: 12px 18px; border-bottom: 1px solid var(--border); gap: 12px; }
  .table-scroll { overflow-x: auto; }
  table { min-width: 800px; }
  .check-cell { width: 44px; text-align: center; }
  input[type='checkbox'] { width: 16px; height: 16px; margin: 2px 0; accent-color: var(--ok); cursor: pointer; }
  input:disabled { cursor: not-allowed; }
  tr.selected { background: var(--row-active); }
  .break { overflow-wrap: anywhere; min-width: 160px; }
  .sub { margin-top: 4px; }
  .nowrap, .readonly { white-space: nowrap; }
  .confirm-panel { position: sticky; bottom: 0; border: 1px solid var(--border-strong); border-radius: 6px; padding: 16px; background: var(--panel); margin-top: 16px; box-shadow: 0 -4px 16px rgb(0 0 0 / 8%); }
  .confirm-panel strong { font-weight: 500; }
  .confirm-panel p { color: var(--muted); font-size: 13px; }
  .confirm-panel ul { margin: 12px 0; padding-left: 20px; max-height: 128px; overflow: auto; overflow-wrap: anywhere; }
  .confirm-panel .row { justify-content: flex-end; }
</style>

<script lang="ts">
  import { createEventDispatcher, onMount, tick } from 'svelte';
  import { GetProcessChain, OpenKibana } from '../../wailsjs/go/main/App.js';
  import type { service } from '../../wailsjs/go/models';
  import { formatTime } from '../lib/format';
  import { processGraph } from '../lib/processGraph.mjs';
  export let alertId: string;
  const dispatch = createEventDispatcher();
  let chain: service.ProcessChain | null = null;
  let selected: service.ProcessNode | null = null;
  let loading = true;
  let error = '';
  let viewport: HTMLElement;
  let dialog: HTMLElement;
  let modal: HTMLDialogElement;
  let zoom = 1, panX = 0, panY = 0;
  let dragging = false;
  let lastX = 0, lastY = 0;
  let copied = false;
  let mounted = true;
  $: graph = processGraph(chain);
  async function load() {
    loading = true; error = '';
    try {
      const result = await GetProcessChain(alertId);
      if (!mounted) return;
      chain = result;
      selected = chain.nodes.find(n => n.isAlert) || chain.nodes[0] || null;
    } catch (e) { if (mounted) error = String(e); }
    finally { if (mounted) { loading = false; await tick(); fit(); } }
  }
  function fit() {
    if (!viewport) return;
    zoom = Math.max(.15, Math.min(1, (viewport.clientWidth - 32) / graph.width, (viewport.clientHeight - 32) / graph.height));
    panX = (viewport.clientWidth - graph.width * zoom) / 2;
    panY = (viewport.clientHeight - graph.height * zoom) / 2;
  }
  function scale(next: number, x = viewport?.clientWidth / 2, y = viewport?.clientHeight / 2) {
    const value = Math.max(.15, Math.min(2, next));
    panX = x - (x - panX) * value / zoom; panY = y - (y - panY) * value / zoom; zoom = value;
  }
  function wheel(e: WheelEvent) {
    e.preventDefault(); const box = viewport.getBoundingClientRect();
    scale(zoom * (e.deltaY > 0 ? .9 : 1.1), e.clientX - box.left, e.clientY - box.top);
  }
  function pointerdown(e: PointerEvent) {
    if (e.button !== 0 || (e.target as Element).closest('button')) return;
    dragging = true; lastX = e.clientX; lastY = e.clientY; viewport.setPointerCapture(e.pointerId);
  }
  function pointermove(e: PointerEvent) {
    if (!dragging) return;
    panX += e.clientX - lastX; panY += e.clientY - lastY; lastX = e.clientX; lastY = e.clientY;
  }
  function canvasKey(e: KeyboardEvent) {
    const moves: Record<string, number[]> = { ArrowLeft: [40, 0], ArrowRight: [-40, 0], ArrowUp: [0, 40], ArrowDown: [0, -40] };
    if (e.target !== viewport) return;
    if (moves[e.key]) { e.preventDefault(); panX += moves[e.key][0]; panY += moves[e.key][1]; }
  }
  function keydown(e: KeyboardEvent) {
    if (e.key === 'Escape') { e.stopPropagation(); dispatch('close'); }
    if (e.key === 'Tab' && dialog) {
      const controls = [...dialog.querySelectorAll<HTMLElement>('button:not(:disabled), [tabindex="0"]')];
      const first = controls[0], last = controls.at(-1);
      if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last?.focus(); }
      else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first?.focus(); }
    }
  }
  async function copyCommand() {
    try { await navigator.clipboard.writeText(selected?.commandLine || selected?.args?.join(' ') || ''); copied = true; }
    catch { error = '复制失败，请选择命令行文字手动复制'; }
  }
  onMount(() => {
    const previous = document.activeElement as HTMLElement;
    modal.showModal();
    load(); tick().then(() => dialog.focus());
    const observer = new ResizeObserver(() => fit());
    // The viewport appears after the async response; observe the persistent dialog.
    observer.observe(dialog);
    return () => { mounted = false; observer.disconnect(); previous?.focus(); };
  });
</script>

<svelte:window on:keydown={keydown} />
<dialog class="chain-overlay" bind:this={modal} aria-labelledby="chain-title" on:cancel|preventDefault={() => dispatch('close')} on:click={(e) => { if (e.target === e.currentTarget) dispatch('close'); }}>
  <div class="chain-dialog" tabindex="-1" bind:this={dialog}>
    <header><div><h2 id="chain-title">进程分析</h2><div class="chain-context">{chain?.hostName || ''}{chain ? ` / ${chain.alertName}` : ''}</div></div><button class="btn ghost sm close" aria-label="关闭进程分析" on:click={() => dispatch('close')}>×</button></header>
    {#if loading}<div class="empty" role="status">正在还原进程关系…</div>
    {:else if !chain}<div class="empty error" role="alert">{error}<div><button class="btn ghost sm" on:click={load}>重试</button></div></div>
    {:else}
      <div class="analysis-body">
        <div class="graph-area">
          <div class="graph-toolbar"><div class="legend"><span class="alert-dot"></span>告警进程<span class="legend-count">{graph.nodes.length} 个节点</span></div><div class="zoom-actions"><button class="btn ghost sm" aria-label="缩小进程图" on:click={() => scale(zoom / 1.2)}>−</button><span class="zoom-value">{Math.round(zoom * 100)}%</span><button class="btn ghost sm" aria-label="放大进程图" on:click={() => scale(zoom * 1.2)}>+</button><button class="btn ghost sm" on:click={fit}>适应画布</button></div></div>
          <!-- svelte-ignore a11y_no_noninteractive_element_interactions a11y_no_noninteractive_tabindex -->
          <div class="graph-viewport" class:dragging bind:this={viewport} role="region" aria-label="进程关系图，可用方向键移动画布" tabindex="0" on:keydown={canvasKey} on:wheel|nonpassive={wheel} on:pointerdown={pointerdown} on:pointermove={pointermove} on:pointerup={() => dragging = false} on:pointercancel={() => dragging = false} on:lostpointercapture={() => dragging = false}>
            <div class="graph-world" style={`width:${graph.width}px;height:${graph.height}px;transform:translate(${panX}px, ${panY}px) scale(${zoom})`}>
              <svg class="edges" width={graph.width} height={graph.height} aria-hidden="true">{#each graph.edges as edge (edge.id)}<path d={edge.path} class:alert-edge={edge.alert} />{/each}</svg>
              {#each graph.nodes as node (node.entityId)}
                <button class="process-node" class:alert-node={node.isAlert} class:selected={selected?.entityId === node.entityId} class:unresolved={!node.resolved} style={`left:${node.x}px;top:${node.y}px`} aria-pressed={selected?.entityId === node.entityId} aria-label={`${node.name || '记录已过期'}，PID ${node.pid || '未知'}${node.isAlert ? '，告警进程' : ''}`} on:click={() => { selected = node; copied = false; }}>
                  <span class="node-name">{node.name || '记录已过期'}</span><span class="node-meta">PID {node.pid || '—'}<span>{node.isAlert ? '告警' : !node.resolved ? '记录缺失' : node.end ? '已结束' : '进程'}</span></span>
                </button>
              {/each}
            </div>
          </div>
          <div class="graph-bottom"><button class="btn ghost sm" on:click={() => OpenKibana('/app/security/alerts')}>在 Kibana 中查看</button></div>
        </div>
        <aside class="node-detail" aria-label="选中进程详情">
          {#if selected}
            <div class="detail-heading"><span class="detail-kind" class:danger={selected.isAlert}>{selected.isAlert ? '告警进程' : '进程详情'}</span><h3>{selected.name || '记录已过期'}</h3><span class="muted">PID {selected.pid || '—'}</span></div>
            {#if !selected.resolved}<div class="missing">该进程的事件记录已过期或未采集。</div>{/if}
            <div class="command-heading"><h4>完整命令行</h4><button class="btn ghost sm" disabled={!selected.commandLine && !selected.args?.length} on:click={copyCommand}>{copied ? '已复制' : '复制'}</button></div>
            <pre class="command">{selected.commandLine || selected.args?.join(' ') || '未记录命令行'}</pre>
            <dl class="process-info"><dt>可执行文件</dt><dd class="mono">{selected.executable || '—'}</dd><dt>工作目录</dt><dd class="mono">{selected.workingDir || '—'}</dd><dt>用户</dt><dd>{selected.user || '—'}</dd><dt>启动时间</dt><dd>{formatTime(selected.start || selected.timestamp)}</dd><dt>结束时间</dt><dd>{selected.end ? formatTime(selected.end) : '未记录'}</dd></dl>
            {#if error}<div class="error" role="alert">{error}</div>{/if}
          {/if}
        </aside>
      </div>
    {/if}
  </div>
</dialog>

<style>
  .chain-overlay { position: fixed; inset: 0; width: 100vw; max-width: none; height: 100dvh; max-height: none; margin: 0; border: 0; padding: 24px; background: transparent; color: var(--text); --wails-draggable: no-drag; }
  .chain-overlay[open] { display: grid; place-items: center; }
  .chain-overlay::backdrop { background: var(--backdrop); }
  .chain-dialog { width: min(1320px, 100%); height: min(850px, 90vh); display: flex; flex-direction: column; overflow: hidden; border: 1px solid var(--border-strong); border-radius: 8px; background: var(--dialog-panel); box-shadow: var(--shadow); outline: none; }
  header { padding: 18px 24px; display: flex; justify-content: space-between; gap: 20px; border-bottom: 1px solid var(--border); }
  h2 { margin: 0; font-size: 17px; font-weight: 600; } .close { font-size: 24px; align-self: flex-start; }
  .chain-context { color: var(--muted); margin-top: 7px; font-size: 12px; overflow-wrap: anywhere; }
  .analysis-body { flex: 1; min-height: 0; display: grid; grid-template-columns: minmax(0, 1fr) 340px; }
  .graph-area { min-width: 0; display: flex; flex-direction: column; }
  .graph-toolbar { display: flex; align-items: center; flex-wrap: wrap; justify-content: space-between; gap: 10px; padding: 12px 16px; border-bottom: 1px solid var(--border-soft); }
  .legend, .zoom-actions { display: flex; align-items: center; gap: 8px; color: var(--muted); font-size: 12px; }
  .alert-dot { width: 7px; height: 7px; background: var(--danger); border-radius: 50%; }
  .legend-count { padding-left: 12px; color: var(--muted-2); } .zoom-value { min-width: 38px; text-align: center; font-variant-numeric: tabular-nums; }
  .graph-viewport { position: relative; overflow: hidden; flex: 1; min-height: 240px; background-color: var(--panel-2); background-image: radial-gradient(var(--border-strong) .7px, transparent .7px); background-size: 20px 20px; cursor: grab; touch-action: none; }
  .graph-viewport.dragging { cursor: grabbing; } .graph-world { position: absolute; transform-origin: 0 0; }
  .edges { position: absolute; inset: 0; pointer-events: none; overflow: visible; } .edges path { fill: none; stroke: var(--muted-2); stroke-width: 1.5; } .edges .alert-edge { stroke: var(--danger); }
  .process-node { position: absolute; width: 208px; height: 72px; padding: 12px 14px; text-align: left; display: flex; flex-direction: column; justify-content: center; gap: 8px; background: var(--panel); border: 1px solid var(--border-strong); border-radius: 5px; color: var(--text); cursor: pointer; }
  .process-node:hover { border-color: var(--accent); } .process-node.selected { outline: 2px solid var(--accent); outline-offset: 3px; } .process-node.alert-node { border-color: var(--danger); box-shadow: inset 3px 0 var(--danger); } .process-node.unresolved { border-style: dashed; color: var(--muted); }
  .node-name { font-size: 13px; font-weight: 600; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; } .node-meta { font-size: 11px; color: var(--muted); display: flex; justify-content: space-between; } .alert-node .node-meta span { color: var(--danger); }
  .graph-bottom { display: flex; align-items: center; justify-content: flex-end; gap: 8px; border-top: 1px solid var(--border-soft); padding: 10px 16px; }
  .node-detail { overflow-y: auto; padding: 22px; border-left: 1px solid var(--border); }
  .detail-heading { margin-bottom: 20px; }
  .detail-kind { font-size: 12px; color: var(--muted); } .detail-kind.danger { color: var(--danger); } h3 { margin: 10px 0 6px; font-size: 18px; font-weight: 600; overflow-wrap: anywhere; } .detail-heading > .muted { font-size: 12px; }
  dl { margin: 20px 0; display: grid; grid-template-columns: 72px minmax(0, 1fr); gap: 12px; font-size: 12px; line-height: 1.7; } dt { color: var(--muted); } dd { color: var(--text-soft); margin: 0; overflow-wrap: anywhere; }
  h4 { font-size: 12px; font-weight: 600; margin: 16px 0 10px; } .command-heading { display: flex; justify-content: space-between; align-items: center; } .command-heading h4 { margin: 0; }
  .command { padding: 12px; background: var(--panel-2); border: 1px solid var(--border-soft); border-radius: 4px; font: 12px/1.8 var(--mono); white-space: pre-wrap; overflow-wrap: anywhere; user-select: text; }
  .missing { margin-top: 16px; color: var(--muted); font-size: 12px; line-height: 1.7; } .error { color: var(--danger); overflow-wrap: anywhere; } .empty { padding: 60px 24px; } .empty button { margin-top: 20px; }
  @media (max-width: 900px) { .analysis-body { grid-template-columns: minmax(0, 1fr) 280px; } .node-detail { padding: 16px; } .legend-count { display: none; } }
  @media (max-width: 640px) { .chain-overlay { padding: 8px; } .chain-dialog { height: 96vh; } header { padding: 14px 16px; } .analysis-body { display: flex; flex-direction: column; overflow-y: auto; } .graph-area { flex: none; height: 350px; } .node-detail { flex: none; border-left: 0; border-top: 1px solid var(--border); overflow-y: visible; } .graph-toolbar { padding: 10px; } .graph-bottom { padding: 6px 10px; } }
</style>

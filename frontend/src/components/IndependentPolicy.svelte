<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { GetIndependentPolicyState, CreateIndependentPolicy, RenameIndependentPolicy, CompletePolicySetup, OpenKibana } from '../../wailsjs/go/main/App.js';
  import type { service } from '../../wailsjs/go/models';
  export let host = '';
  export let onboarding = false;
  export let isolated = false;
  const dispatch = createEventDispatcher();
  let state: service.IndependentPolicyState | null = null;
  let name = `Elastic UI - ${host || '本机'}`;
  let busy = false;
  let error = '';
  async function refresh() {
    if (busy) return;
    busy = true; error = '';
    try { state = await GetIndependentPolicyState(); if (state.independent) name = state.name; }
    catch (e) { error = String(e); } finally { busy = false; }
  }
  async function save() {
    if (busy || isolated || !state || !name.trim()) return;
    busy = true; error = '';
    try {
      state = state?.independent ? await RenameIndependentPolicy(name) : await CreateIndependentPolicy(name);
      dispatch('changed');
    } catch (e) { error = String(e); } finally { busy = false; }
  }
  async function finish() {
    if (busy) return;
    busy = true;error = '';
    try { await CompletePolicySetup(); dispatch('done'); }
    catch(e) { error=String(e); } finally {busy=false;}
  }
  onMount(refresh);
</script>

<section class="policy-panel" class:panel={onboarding} class:embedded={!onboarding} aria-busy={busy}>
  <div class="panel-head"><h2 class="panel-title">{onboarding ? '为本机创建独立策略' : '本机独立策略'}</h2><button class="btn ghost sm" disabled={busy} on:click={refresh}>刷新状态</button></div>
  <div class="panel-body">
    {#if state}
      <dl><dt>当前策略</dt><dd>{state.name}</dd><dt>绑定主机</dt><dd>{state.hostCount} 台</dd><dt>状态</dt><dd role="status" class:applied={state.applied}>{state.message}</dd></dl>
    {/if}
    <form on:submit|preventDefault={save}>
      <label for="independent-policy-name">独立策略名称</label>
      <div class="policy-actions"><input id="independent-policy-name" bind:value={name} maxlength="120" disabled={busy || isolated} /><button class="btn primary" disabled={!state || busy || isolated || !name.trim()}>{busy ? '处理中…' : state?.independent ? '保存名称' : '创建并绑定本机'}</button></div>
    </form>
    {#if state?.independent}<div class="policy-footer"><button class="btn ghost" disabled={busy} on:click={() => OpenKibana('/app/security/administration/policy/' + state?.packageId + '/settings')}>自定义防护策略</button></div>{/if}
    {#if error}<p class="policy-error" role="alert">{error}</p>{/if}
    {#if onboarding}<div class="policy-actions"><button class="btn" disabled={busy} on:click={finish}>{state?.independent ? '进入控制台' : '稍后在设置中创建'}</button></div>{/if}
  </div>
</section>

<style>
  .policy-panel { min-width: 0; }
  .embedded { margin-top: 24px; padding-top: 20px; border-top: 1px solid var(--border); }
  .embedded .panel-head { padding: 0 0 16px; border: 0; background: transparent; }
  .embedded .panel-body { padding: 0; }
  h2 { margin: 0; }
  dl { display: grid; grid-template-columns: 112px minmax(0, 1fr); gap: 12px 16px; margin: 0 0 24px; font-size: 13px; line-height: 1.6; }
  dt { color: var(--muted); }
  dd { margin: 0; overflow-wrap: anywhere; color: var(--text-soft); }
  dd.applied { color: var(--ok); }
  form { display: grid; grid-template-columns: 112px minmax(0, 1fr); gap: 16px; align-items: start; }
  label { padding-top: 8px; font-size: 13px; color: var(--muted); }
  .policy-actions { display: flex; gap: 8px; align-items: center; }
  input { flex: 1; min-width: 0; width: 100%; height: 36px; font-size: 13px; }
  .policy-actions .btn { flex: none; min-height: 36px; }
  .policy-footer { display: flex; justify-content: flex-end; margin-top: 12px; }
  .policy-error { color: var(--danger); margin: 16px 0 0; overflow-wrap: anywhere; font-size: 13px; }
  .panel-body > .policy-actions { margin-top: 20px; justify-content: flex-end; }
  @media (max-width: 639px) {
    dl { grid-template-columns: 88px minmax(0, 1fr); }
    form { grid-template-columns: minmax(0, 1fr); gap: 8px; }
    label { padding: 0; }
    .policy-actions { flex-wrap: wrap; }
    .policy-actions input { flex-basis: 100%; }
    .policy-footer { justify-content: flex-start; }
  }
</style>

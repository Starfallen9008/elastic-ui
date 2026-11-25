<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { Configure } from '../../wailsjs/go/main/App.js';
  import type { config, service } from '../../wailsjs/go/models';

  export let initial: config.Profile | undefined;

  const dispatch = createEventDispatcher();

  let kibanaUrl = initial?.kibanaUrl || '';
  let esUrl = initial?.esUrl || '';
  let authMethod = initial?.authMethod || 'basic';
  let username = initial?.username || '';
  let password = '';
  let apiKey = '';
  let skipTlsVerify = initial?.skipTlsVerify || false;
  let busy = false;
  let error = '';
  let result: service.ConnectionResult | null = null;

  async function submit() {
    busy = true;
    error = '';
    result = null;
    try {
      result = await Configure({
        kibanaUrl,
        esUrl,
        authMethod,
        username,
        password,
        apiKey,
        skipTlsVerify
      });
      if (result.ok) {
        dispatch('done');
      } else {
        error = '连接检查未通过，请查看下面的项目';
      }
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }
</script>

<div class="setup">
  <div class="setup-left">
    <div class="mark">ED</div>
    <h1>Elastic Defend 本机控制台</h1>
  </div>

  <div class="setup-right panel">
    <div class="panel-head">
      <div class="panel-title">连接配置</div>
    </div>
    <form class="panel-body" on:submit|preventDefault={submit}>
      <div class="field">
        <label for="kibana">Kibana 地址</label>
        <input id="kibana" bind:value={kibanaUrl} placeholder="https://kibana.example.com" spellcheck="false" />
      </div>

      <div class="field">
        <label for="es">Elasticsearch 地址</label>
        <input id="es" bind:value={esUrl} placeholder="https://es.example.com" spellcheck="false" />
      </div>

      <div class="field">
        <div class="field-label">认证方式</div>
        <div class="segmented" role="group" aria-label="认证方式">
          <button type="button" class:on={authMethod === 'basic'} on:click={() => (authMethod = 'basic')}>用户名密码</button>
          <button type="button" class:on={authMethod === 'apikey'} on:click={() => (authMethod = 'apikey')}>API Key</button>
        </div>
      </div>

      {#if authMethod === 'basic'}
        <div class="two">
          <div class="field">
            <label for="user">用户名</label>
            <input id="user" bind:value={username} autocomplete="username" />
          </div>
          <div class="field">
            <label for="pass">密码</label>
            <input id="pass" type="password" bind:value={password} autocomplete="current-password" />
          </div>
        </div>
      {:else}
        <div class="field">
          <label for="apikey">API Key</label>
          <input id="apikey" type="password" bind:value={apiKey} placeholder="粘贴 Base64 编码的 API Key" />
        </div>
      {/if}

      <label class="check">
        <input type="checkbox" bind:checked={skipTlsVerify} />
        跳过 TLS 证书校验（仅用于自签证书环境）
      </label>

      <div class="actions">
        <button class="btn primary" type="submit" disabled={busy}>
          {busy ? '正在连接...' : '测试并连接'}
        </button>
      </div>

      {#if error}
        <div class="error">{error}</div>
      {/if}

      {#if result}
        <div class="checks">
          {#each result.checks as check}
            <div class="check-row">
              <span class="dot" class:ok={check.ok} class:bad={!check.ok}></span>
              <span class="check-name">{check.name}</span>
              <span class="check-detail">{check.detail}</span>
            </div>
          {/each}
          {#if result.agent}
            <div class="bound">
              已绑定本机：{result.agent.hostName}，策略 {result.agent.policyName}
            </div>
          {/if}
        </div>
      {/if}
    </form>
  </div>
</div>

<style>
  .setup {
    height: 100%;
    display: grid;
    grid-template-columns: minmax(320px, 1fr) minmax(420px, 520px);
    gap: 48px;
    align-items: center;
    padding: 56px 64px;
  }

  .setup-left {
    max-width: 460px;
  }

  .mark {
    width: 44px;
    height: 44px;
    border-radius: 10px;
    background: var(--glass-strong);
    border: 1px solid var(--border-strong);
    color: var(--accent);
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    letter-spacing: 0.5px;
    margin-bottom: 22px;
  }

  h1 {
    font-size: 26px;
    margin: 0 0 12px;
    font-weight: 600;
  }

  .setup-right {
    width: 100%;
  }

  .field-label {
    font-size: 12px;
    color: var(--muted);
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .two {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .segmented {
    display: flex;
    border: 1px solid var(--border);
    border-radius: 6px;
    overflow: hidden;
    width: fit-content;
  }

  .segmented button {
    height: 30px;
    padding: 0 14px;
    border: 0;
    background: transparent;
    color: var(--muted);
  }

  .segmented button.on {
    background: var(--panel-3);
    color: var(--text);
  }

  .check {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: var(--muted);
  }

  .check input {
    margin: 0;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
  }

  .error {
    padding: 10px 12px;
    border-radius: 6px;
    background: var(--danger-bg);
    border: 1px solid var(--danger-border);
    color: var(--danger);
    font-size: 12px;
    word-break: break-word;
  }

  .checks {
    border-top: 1px solid var(--border-soft);
    padding-top: 14px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .check-row {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
  }

  .check-name {
    width: 130px;
    color: var(--muted);
  }

  .check-detail {
    color: var(--muted-2);
    word-break: break-word;
  }

  .bound {
    margin-top: 4px;
    font-size: 12px;
    color: var(--ok);
  }
</style>

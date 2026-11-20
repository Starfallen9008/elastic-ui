// Run against the local Wails fixture. All policy and scan mutations stay mocked.
(async () => {
  const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
  const assert = (ok, message) => { if (!ok) throw new Error(message); };
  const until = async fn => { for (let i = 0; i < 160; i++) { if (fn()) return; await pause(25); } throw new Error('UI timed out'); };
  const bridge = window.go.main.App;
  await until(() => document.querySelector('.protection-trigger'));
  const status = await bridge.GetStatus();
  bridge.GetStatus = async () => status;
  document.querySelector('.protection-trigger').click();
  await until(() => document.querySelector('#protection-popover:popover-open'));
  const row = label => [...document.querySelectorAll('.prot-row')].find(el => el.firstElementChild.textContent === label);
  assert(document.querySelectorAll('.protection-state').length === 6, 'Missing state pills');
  assert(row('设备控制').querySelector('.enabled'), 'Enabled device control is not green');
  assert(!row('防篡改').querySelector('.enabled'), 'Disabled tamper protection is green');
  assert(!document.querySelector('.protection-state button, .protection-state input, [role="switch"]'), 'Status pills are interactive');
  const close = document.querySelector('[aria-label="关闭防护模块"]');
  assert(close.querySelector('svg') && !close.textContent.trim(), 'Close is not an X icon');
  close.click();
  status.protection.deviceControl = '关闭';
  window.fixtureEvents['status:updated']?.(status);
  // App refreshes status through its configured handler as well as its timer.
  await window.fixtureEvents.configured();
  await until(() => document.querySelector('.protection-trigger'));
  document.querySelector('.protection-trigger').click();
  await until(() => document.querySelector('.prot-row'));
  assert(!row('设备控制').querySelector('.enabled'), 'Disabled device control is green');
  document.querySelector('[aria-label="关闭防护模块"]').click();
  bridge.GetScans = async () => [{ id: 'scan', actionId: 'action', path: 'C:\\Users', startedAt: '2026-11-19T12:05:08+08:00', completedAt: '2026-11-19T12:05:27+08:00', status: 'successful', found: 0 }];
  document.querySelector('.position-scan').click();
  await until(() => document.querySelector('.tasks tbody tr'));
  assert(!/小时前|天前|分钟前/.test(document.querySelector('.tasks').textContent), 'Relative scan time remains');
  assert(!document.querySelector('.hint'), 'Scan explanatory copy remains');
  document.querySelector('.back-btn').click();
  await until(() => document.querySelector('.position-settings') && !document.querySelector('.back-btn'));
  bridge.GetIndependentPolicyState = async () => ({ independent: true, applied: true, name: 'Elastic UI - Dreamending-PC', hostCount: 1, packageId: 'defend', message: '本机独立策略已生效' });
  document.querySelector('.position-settings').click();
  await until(() => document.querySelector('#independent-policy-name'));
  assert(document.querySelector('.policy-panel.embedded:not(.panel)'), 'Policy has nested panel chrome');
  assert(!document.querySelector('.cards .note'), 'Static settings explanatory copy remains');
  assert(document.querySelector('.policy-footer button'), 'Advanced policy action missing');
  assert(document.querySelector('.content').scrollWidth <= document.querySelector('.content').clientWidth, 'Settings overflow horizontally');
  return { passed: true, viewport: [innerWidth, innerHeight], statePills: 6, deviceControlStates: ['enabled', 'disabled'], scanTime: 'absolute' };
})()

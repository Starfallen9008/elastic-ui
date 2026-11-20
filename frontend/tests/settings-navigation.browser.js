// Evaluate after agent-browser press Tab to enable keyboard focus indicators.
// Uses the local mocked Wails preview; never invokes real security actions.
(async () => {
  const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
  const assert = (value, message) => { if (!value) throw new Error(message); };
  const until = async predicate => {
    for (let i = 0; i < 120; i++) { if (predicate()) return; await pause(25); }
    throw new Error('Timed out waiting for UI');
  };
  const button = text => [...document.querySelectorAll('button')].find(element => element.textContent.trim() === text);
  const bridge = window.go.main.App;
  const calls = [];
  for (const name of ['Isolate', 'CreateTrust', 'DeleteTrust', 'Disconnect', 'ClearCache']) bridge[name] = async () => { calls.push(name); throw new Error('Unexpected mutation'); };
  await until(() => document.querySelector('.position-settings'));
  assert(document.querySelectorAll('.navigation-item').length === 4, 'Wrong home entry count');
  assert(!document.querySelector('.position-status, .position-trust'), 'Removed entries remain');
  for (const entry of document.querySelectorAll('.navigation-item')) {
    assert(getComputedStyle(entry).borderWidth === '0px', 'Entry has border');
    assert(entry.getAttribute('aria-label'), 'Missing accessible name');
    entry.focus(); await pause(260);
    assert(getComputedStyle(entry.querySelector('.navigation-label')).opacity === '1', 'Focus did not reveal label');
  }
  document.activeElement.blur();
  document.querySelector('.position-settings').click();
   await until(() => button('信任区'));
  const titles = [...document.querySelectorAll('.cards > section > .panel-head > .panel-title')].map(element => element.textContent.trim());
   assert(JSON.stringify(titles) === JSON.stringify(['应急隔离', '连接', '本机绑定', '右键菜单开关', '通知与托盘', '维护', '信任区', '关于']), 'Settings order incorrect: ' + titles);
  assert(button('隔离此主机').disabled, 'Isolation enabled without hostname confirmation');
   const trigger = button('信任区');
  trigger.focus(); trigger.click();
  await until(() => document.querySelector('.trust-dialog:modal'));
  const dialog = document.querySelector('.trust-dialog');
  assert(dialog.contains(document.activeElement), 'Focus did not enter dialog');
  const bounds = dialog.getBoundingClientRect();
  assert(bounds.left >= 0 && bounds.right <= innerWidth && bounds.top >= 0 && bounds.bottom <= innerHeight, 'Dialog outside viewport');
  assert(getComputedStyle(dialog.querySelector('.dialog-content')).overflowY === 'auto', 'Missing independent scroll');
  dialog.dispatchEvent(new Event('cancel', { cancelable: true }));
  await until(() => !document.querySelector('.trust-dialog'));
  assert(document.activeElement === trigger, 'Dialog did not restore focus');
  trigger.click(); await until(() => document.querySelector('.trust-dialog:modal'));
  // Hold a fixture request to check every dismissal path while a write is pending.
  let resolveCreate;
  bridge.CreateTrust = () => new Promise(resolve => { resolveCreate = resolve; });
  const input = document.querySelector('#trust-value');
  input.value = 'C:\\Fixture.exe'; input.dispatchEvent(new Event('input', { bubbles: true }));
  await pause(25); button('添加').click();
  await until(() => resolveCreate);
  const busyDialog = document.querySelector('.trust-dialog');
   assert(busyDialog.querySelector('[aria-label="关闭信任区"]').disabled, 'Busy close enabled');
  busyDialog.dispatchEvent(new Event('cancel', { cancelable: true }));
  busyDialog.dispatchEvent(new MouseEvent('click', { clientX: -1, clientY: -1 }));
  await pause(25);
  assert(document.querySelector('.trust-dialog:modal'), 'Busy dialog was dismissed');
   resolveCreate(); await until(() => !busyDialog.querySelector('[aria-label="关闭信任区"]').disabled);
   busyDialog.querySelector('[aria-label="关闭信任区"]').click();
  await until(() => !document.querySelector('.trust-dialog'));
  assert(!calls.length, 'Unexpected mutation: ' + calls);
  return { passed: true, viewport: [innerWidth, innerHeight], titles, dialog: { width: bounds.width, height: bounds.height }, realMutations: 0 };
})()

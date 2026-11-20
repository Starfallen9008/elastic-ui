// Run only against the mocked preview. All policy writes are replaced here.
(async () => {
  const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
  const assert = (condition, message) => { if (!condition) throw new Error(message); };
  const until = async predicate => {
    for (let i = 0; i < 150; i++) { if (predicate()) return; await pause(20); }
    throw new Error('Timed out waiting for notification UI');
  };
  const bridge = window.go.main.App;
  const button = label => [...document.querySelectorAll('button')].find(el => el.textContent.trim() === label);
  await until(() => document.querySelector('.position-settings'));
  const layout = document.querySelector('.dashboard-layout').getBoundingClientRect();
  const center = { x: layout.x + layout.width / 2, y: layout.y + layout.height / 2 };
  const positions = {};
  for (const id of ['scan', 'events', 'threats', 'settings']) {
    const element = document.querySelector(`.position-${id}`);
    const rect = element.getBoundingClientRect();
    positions[id] = { x: rect.x + rect.width / 2, y: rect.y + rect.height / 2 };
    assert(element.querySelector('svg').getAttribute('stroke-width') === '1.5', 'Inconsistent icon stroke');
    assert(element.querySelector('svg').getAttribute('viewBox') === '0 0 24 24', 'Inconsistent icon canvas');
  }
  if (innerWidth >= 800 && innerHeight >= 680) {
    assert(Math.abs(positions.scan.y - positions.events.y) < 1, 'Top icons misaligned');
    assert(Math.abs(positions.threats.y - positions.settings.y) < 1, 'Bottom icons misaligned');
    assert(Math.abs((positions.scan.x + positions.events.x) / 2 - center.x) < 1, 'Horizontal symmetry lost');
    assert(Math.abs((positions.scan.y + positions.threats.y) / 2 - center.y) < 1, 'Vertical symmetry lost');
  }
  let mode = 'native';
  bridge.GetNotificationState = async () => ({ mode, ready: true, message: mode === 'ui' ? '由 Elastic UI 显示桌面弹窗' : '由 Elastic Agent 显示原生通知' });
  let release;
  const requests = [];
  bridge.SetNotificationMode = value => { requests.push(value); return new Promise(resolve => release = () => { mode = value; resolve(); }); };
  document.querySelector('.position-settings').click();
  await until(() => button('Elastic UI'));
  assert(button('Elastic 原生').getAttribute('aria-pressed') === 'true', 'Default native mode missing');
  button('Elastic UI').click();
  await until(() => release);
  assert(button('Elastic UI').disabled && button('Elastic 原生').disabled, 'Switch allows overlapping writes');
  assert(button('Elastic 原生').getAttribute('aria-pressed') === 'true', 'Mode changed before API success');
  release();
  await until(() => button('Elastic UI').getAttribute('aria-pressed') === 'true');
  assert(requests.length === 1, 'Unexpected extra policy write');
  bridge.SetNotificationMode = async () => { throw new Error('Fixture policy denied'); };
  button('Elastic 原生').click();
  await until(() => document.querySelector('.notification-error'));
  assert(button('Elastic UI').getAttribute('aria-pressed') === 'true', 'Failed switch changed mode');
  assert(document.querySelector('.notification-error').textContent.includes('Fixture policy denied'), 'API failure hidden');
  bridge.SetNotificationMode = async value => { mode = value; };
  button('Elastic 原生').click();
  await until(() => button('Elastic 原生').getAttribute('aria-pressed') === 'true');
  let openedID;
  bridge.GetThreatDetail = async id => { openedID = id; return { id, time: new Date().toISOString(), ruleName: 'Fixture detail', responses: [], tactics: [], techniques: [] }; };
  bridge.MarkRead = async () => {};
  window.fixtureEvents['notification-details']('fixture-alert');
  await until(() => openedID === 'fixture-alert');
  await until(() => document.body.textContent.includes('Fixture detail'));
  return { passed: true, positions, mockedPolicyWrites: requests, detailID: openedID };
})()

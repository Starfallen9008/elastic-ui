// Independent preview fixtures only. __refreshStatus is captured before mount.
(async () => {
  const assert = (value, message) => { if (!value) throw new Error(message); };
  const until = async predicate => {
    for (let i = 0; i < 120; i++) {
      if (predicate()) return;
      await new Promise(resolve => setTimeout(resolve, 25));
    }
    throw new Error('Timed out waiting for failure UI');
  };
  await until(() => document.querySelector('.modal-title'));
  assert(document.querySelector('.modal-title').textContent === '管理连接失败', 'Missing connection failure title');
  assert(document.querySelector('.alert-error').textContent.includes('fixture timeout'), 'Backend error swallowed');
  assert(document.querySelector('.raw').textContent.includes('Running'), 'Local Agent output lost');
  document.querySelector('.modal-head button').click();
  document.querySelector('.protection-trigger').click();
  await until(() => document.querySelector('#protection-popover:popover-open'));
  const badges = () => [...document.querySelectorAll('.prot-row .badge')];
  assert(badges().length === 6 && badges().every(b => b.textContent === '无法读取' && b.classList.contains('danger')), 'Blank or non-red failure badge');
  assert(document.querySelector('.protection-error').textContent.includes('fixture timeout'), 'Missing warning reason');
  assert(!document.querySelector('#protection-popover').textContent.includes('未开启'), 'Unknown tamper reported as disabled');
  assert(document.querySelector('.alert-count').textContent === '—', 'Unknown alert count reported as zero');
  const colors = [getComputedStyle(badges()[0]).color];
  document.querySelector('[aria-label="切换主题"]').click();
  await new Promise(resolve => setTimeout(resolve, 50));
  colors.push(getComputedStyle(badges()[0]).color);
  assert(colors[0] !== colors[1], 'Theme colors did not change');
  document.querySelector('[aria-label="切换主题"]').click();
  window.__nextStatus = window.__healthyStatus;
  await window.__refreshStatus();
  await until(() => badges()[0].textContent === '阻止');
  assert(!document.querySelector('.protection-error'), 'Warning not cleared after recovery');
  assert(!document.querySelector('.modal-backdrop'), 'Recovery retained failure modal');
  window.go.main.App.GetStatus = async () => { throw new Error('fixture bridge rejection'); };
  await window.__refreshStatus();
  await until(() => document.querySelector('.modal-title')?.textContent === '管理连接失败');
  document.querySelector('.modal-head button').click();
  assert(badges().every(b => b.textContent === '无法读取'), 'Bridge failure shows stale healthy statuses');
  assert(document.querySelector('.protection-error').textContent.includes('fixture bridge rejection'), 'Bridge error missing');
  return { passed: true, badges: badges().map(b => b.textContent), colors, recovery: true, bridgeFailure: true };
})()

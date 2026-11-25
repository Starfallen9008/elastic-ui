// Preview fixture: all four protections are configured as detect-only.
(async () => {
  const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
  const assert = (value, message) => { if (!value) throw new Error(message); };
  const until = async predicate => {
    for (let i = 0; i < 160; i++) { if (predicate()) return; await pause(25); }
    throw new Error('Timed out waiting for detect-only UI');
  };
  await until(() => document.querySelector('.security-status'));
  await pause(300);
  assert(!document.querySelector('.modal-backdrop'), 'Detect-only auto-opened the incident dialog');
  assert(document.querySelector('.security-status').textContent === '降级', 'Detect-only level changed');
  assert(document.querySelector('.security-description').textContent.includes('仅检测'), 'Detect-only description missing');
  const link = document.querySelector('.alert-link');
  assert(link && link.textContent === '查看防护详情', 'Link still promises alerts');
  link.click();
  await until(() => document.querySelector('.modal-title'));
  assert(document.querySelector('.modal-title').textContent === '仅检测模式', 'Wrong manual dialog title');
  assert(document.querySelector('.modal-kicker').textContent === '当前配置', 'Detect-only presented as an incident');
  assert(document.querySelector('.modal-body').textContent.includes('不阻止'), 'Detect-only behavior not explained');
  assert(!document.querySelector('.raw'), 'Detect-only dialog shows unrelated Agent output');
  document.querySelector('.modal-head button').click();
  await pause(50);
  assert(!document.querySelector('.modal-backdrop'), 'Dialog did not close');
  return { passed: true, level: '降级', autoDialog: false, manualTitle: '仅检测模式' };
})()

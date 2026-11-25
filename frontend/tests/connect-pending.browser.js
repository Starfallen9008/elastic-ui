// Preview fixture: the first status request never resolves, like an unreachable server.
(async () => {
  const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
  const assert = (value, message) => { if (!value) throw new Error(message); };
  const until = async predicate => {
    for (let i = 0; i < 120; i++) { if (predicate()) return; await pause(25); }
    throw new Error('Timed out waiting for connecting UI');
  };
  await until(() => document.querySelector('.boot-state'));
  assert(document.querySelector('.boot-state').textContent.includes('正在连接 Kibana 与 Elasticsearch'), 'Connecting state not shown');
  assert(document.querySelector('.boot-state').textContent.includes('30 秒'), 'Timeout expectation not shown');
  assert(!document.querySelector('.modal-backdrop'), 'Pending connection opened an alert dialog');
  const before = document.documentElement.dataset.theme;
  document.querySelector('[aria-label="切换主题"]').click();
  await pause(350);
  assert(document.documentElement.dataset.theme !== before, 'Window blocked while connecting');
  return { passed: true, connectingVisible: true, shellResponsive: true };
})()

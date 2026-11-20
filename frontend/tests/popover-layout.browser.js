(async () => {
  const assert = (value, message) => { if (!value) throw new Error(message); };
  const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
  await pause(500);
  const trigger = document.querySelector('.protection-trigger');
  trigger.scrollIntoView({ block: 'nearest' });
  if (trigger.getAttribute('aria-expanded') !== 'true') trigger.click();
  await pause(220);
  const panel = document.querySelector('#protection-popover');
  assert(panel?.matches(':popover-open'), 'Popover not open');
  const a = trigger.getBoundingClientRect(), p = panel.getBoundingClientRect();
  assert(p.x >= 0 && p.right <= innerWidth && p.y >= 0 && p.bottom <= innerHeight, 'Viewport overflow');
  const before = [p.x, p.y];
  window.dispatchEvent(new Event('resize')); await pause(50);
  const next = panel.getBoundingClientRect();
  assert(Math.abs(before[0] - next.x) < 0.1 && Math.abs(before[1] - next.y) < 0.1, 'Resize moves a stationary popover');
  assert(document.querySelector('.content').scrollWidth <= document.querySelector('.content').clientWidth, 'Horizontal overflow');
  for (const element of [document.querySelector('.home'), document.querySelector('.page'), document.querySelector('.orbit-area')]) {
    assert(Number(getComputedStyle(element).opacity) === 1, 'Background dimmed');
    assert(getComputedStyle(element).filter === 'none', 'Background filtered');
  }
  const saved = { x: a.x, y: a.y, width: a.width, height: a.height };
  // Force trigger near the top: the overlay must flip below, not leave viewport.
  const original = trigger.getBoundingClientRect;
  trigger.getBoundingClientRect = () => ({ ...saved, top: 64, bottom: 64 + saved.height, left: saved.x, right: saved.x + saved.width });
  window.dispatchEvent(new Event('resize')); await pause(50);
  assert(panel.dataset.placement === 'below' && panel.getBoundingClientRect().top >= 64 + saved.height, 'Popover did not flip below');
  trigger.getBoundingClientRect = original;
  window.dispatchEvent(new Event('resize')); await pause(50);
  // Internal scrolling must not change the anchor gap or dismiss the panel.
  panel.dispatchEvent(new Event('scroll')); await pause(25);
  assert(document.querySelector('#protection-popover'), 'Scroll dismissed panel');
  const theme = document.documentElement.dataset.theme;
  document.querySelector('[aria-label="切换主题"]').click(); await pause(50);
  assert(document.querySelector('#protection-popover'), 'Theme switching dismissed panel');
  const darkBackground = getComputedStyle(panel).backgroundColor;
  document.querySelector('[aria-label="切换主题"]').click(); await pause(50);
  assert(document.documentElement.dataset.theme === theme, 'Theme restore failed');
  if (document.querySelector('.toast')) document.querySelector('.toast').style.visibility = 'hidden';
  return { passed: true, viewport: [innerWidth, innerHeight], placement: panel.dataset.placement, panel: { x: next.x, y: next.y, width: next.width, height: next.height }, alternateThemeBackground: darkBackground };
})()

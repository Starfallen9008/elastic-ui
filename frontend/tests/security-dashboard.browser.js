// Evaluate in the mocked home preview with agent-browser eval -b <base64>.
// This checks layout and routing, not live Agent/Kibana operations.
(async () => {
  const delay = ms => new Promise(resolve => setTimeout(resolve, ms));
  const assert = (value, message) => { if (!value) throw new Error(message); };
  const rect = element => {
    const { x, y, width, height } = element.getBoundingClientRect();
    return { x, y, width, height };
  };
  const overlaps = (a, b) => a.x < b.x + b.width && a.x + a.width > b.x && a.y < b.y + b.height && a.y + a.height > b.y;
  const buttons = () => [...document.querySelectorAll('.navigation-item')];
  const sample = () => [document.querySelector('.core-status'), ...buttons()].map(rect);
  const validateLayout = () => {
    const bounds = rect(document.querySelector('.security-dashboard'));
    const core = rect(document.querySelector('.core-status'));
    const entries = buttons().map(rect);
    assert(entries.length === 4, 'Expected four navigation entries');
    if (getComputedStyle(document.querySelector('.fixed-navigation')).display === 'contents') {
      assert(Math.abs(entries[0].y - entries[1].y) < 1, 'Top icons not aligned');
      assert(Math.abs(entries[2].y - entries[3].y) < 1, 'Bottom icons not aligned');
      assert(Math.abs(entries[0].x - entries[2].x) < 1, 'Left icons not aligned');
      assert(Math.abs(entries[1].x - entries[3].x) < 1, 'Right icons not aligned');
    }
    entries.forEach((button, i) => {
      assert(!overlaps(core, button), `Core overlaps button ${i}`);
      assert(button.width >= 44 && button.height >= 44, `Small click target ${i}`);
      assert(button.x >= bounds.x && button.x + button.width <= bounds.x + bounds.width + 1, `Button ${i} outside width`);
      assert(button.y >= bounds.y && button.y + button.height <= bounds.y + bounds.height + 1, `Button ${i} outside height`);
      entries.slice(i + 1).forEach(other => assert(!overlaps(button, other), 'Navigation collision'));
    });
    const content = document.querySelector('.content');
    assert(content.scrollWidth <= content.clientWidth + 1, 'Horizontal content overflow');
    if (innerWidth >= 1024 && innerHeight >= 680) assert(content.scrollHeight <= content.clientHeight + 1, 'Desktop vertical overflow');
    return { core, entries, compact: getComputedStyle(document.querySelector('.fixed-navigation')).display !== 'contents' };
  };

  await delay(650);
  const layout = validateLayout();
  const before = sample();
  const ring = document.querySelector('.primary-ring');
  const transform = getComputedStyle(ring).transform;
  await delay(350);
  const after = sample();
  before.forEach((a, i) => Object.keys(a).forEach(key => assert(Math.abs(a[key] - after[i][key]) < 0.05, `Static element moved: ${i}/${key}`)));
  const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches;
  assert(reduced ? getComputedStyle(ring).animationName === 'none' : transform !== getComputedStyle(ring).transform, 'Incorrect orbital motion');
  assert(getComputedStyle(document.querySelector('.orbital-decoration')).pointerEvents === 'none', 'Decoration intercepts input');
  // Advance decor through half a revolution without waiting for the 90s cycle.
  document.getAnimations().filter(animation => animation.effect?.target.matches('.orbital-ring'))
    .forEach(animation => { animation.currentTime += 45000; });
  const rotated = sample();
  after.forEach((a, i) => Object.keys(a).forEach(key => assert(Math.abs(a[key] - rotated[i][key]) < 0.05, 'Decoration changed fixed coordinates')));
  for (const button of buttons()) {
    const r = rect(button);
    const hit = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
    if (r.y >= 0 && r.y < innerHeight) assert(hit?.closest('button') === button, 'Button is covered');
  }
  const animated = document.getAnimations().filter(animation => animation.effect?.target.closest('.security-dashboard'));
  assert(animated.every(animation => animation.effect.target.matches('.orbital-ring, .shield-glow')), 'Functional layer is animated');

  buttons()[0].focus({ preventScroll: true });
  buttons()[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
  assert(document.activeElement === buttons()[1], 'Keyboard navigation failed');
  // One click per functional route. Returning restores the same fixed layout.
  for (const id of ['scan', 'settings', 'threats', 'events']) {
    const target = document.querySelector(`.position-${id}`);
    target.click();
    await delay(650);
    assert(document.querySelector('.back-btn') && !document.querySelector('.security-dashboard'), `Route ${id} did not open`);
    document.querySelector('.back-btn').click();
    // Back includes its own 180ms exit plus the page/home entrance sequence.
    // Measure after those finite transitions, not during entrance scale(0.985).
    await delay(1000);
    validateLayout();
  }
  assert(!document.querySelector('.position-status, .position-trust'), 'Removed home/exclusion entry still present');
  document.querySelector('.protection-trigger').click();
  await delay(50);
  assert(document.querySelector('#protection-popover:popover-open'), 'Protection popover failed');
  window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
  await delay(50);
  assert(!document.querySelector('#protection-popover'), 'Popover dismissal failed');
  return { passed: true, viewport: [innerWidth, innerHeight], reduced, ...layout };
})()

import { writable } from 'svelte/store';
import { SetTrayTheme } from '../../wailsjs/go/main/App.js';

export type Theme = 'dark' | 'light';

function readInitial(): Theme {
  try {
    const saved = localStorage.getItem('ed-theme');
    if (saved === 'dark' || saved === 'light') return saved;
  } catch {
    // localStorage 不可用时使用默认值
  }
  return 'dark';
}

const initial = readInitial();
if (typeof document !== 'undefined') {
  document.documentElement.dataset.theme = initial;
}

export const theme = writable<Theme>(initial);

theme.subscribe((value) => {
  if (typeof document !== 'undefined') {
    document.documentElement.dataset.theme = value;
  }
  try {
    localStorage.setItem('ed-theme', value);
  } catch {
    // 忽略存储失败
  }
  try {
    SetTrayTheme(value === 'dark')?.catch(() => {});
  } catch {
    // Wails 运行时尚未注入时忽略
  }
});

export function toggleTheme() {
  const update = () => theme.update(value => value === 'dark' ? 'light' : 'dark');
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) { update(); return; }
  if (document.startViewTransition && !document.querySelector(':modal')) {
    document.startViewTransition(update);
    return;
  }
  document.documentElement.classList.add('theme-changing');
  update();
  window.setTimeout(() => document.documentElement.classList.remove('theme-changing'), 320);
}

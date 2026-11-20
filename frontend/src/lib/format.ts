export function formatTime(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
}

export function relative(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso).getTime();
  if (Number.isNaN(d)) return iso;
  const diff = Date.now() - d;
  if (diff < 0) return '刚刚';
  const min = Math.floor(diff / 60000);
  if (min < 1) return '刚刚';
  if (min < 60) return `${min} 分钟前`;
  const hour = Math.floor(min / 60);
  if (hour < 24) return `${hour} 小时前`;
  return `${Math.floor(hour / 24)} 天前`;
}

export function severityLabel(sev: number): string {
  if (sev >= 90) return '严重';
  if (sev >= 70) return '高';
  if (sev >= 40) return '中';
  return '低';
}

export function severityClass(sev: number): string {
  if (sev >= 90) return 'critical';
  if (sev >= 70) return 'high';
  if (sev >= 40) return 'medium';
  return 'low';
}

export function shortId(id?: string): string {
  if (!id) return '—';
  return id.length > 12 ? id.slice(0, 8) : id;
}

export function actionLabel(action?: string): string {
  switch (action) {
    case 'kill_process': return '已终止进程';
    case 'suspend_process': return '已挂起进程';
    case 'isolate': return '已隔离';
    case 'unisolate': return '已解除隔离';
    default: return action ? action : '已阻止';
  }
}

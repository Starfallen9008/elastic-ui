export type TrustKind = 'path' | 'wildcard' | 'hash';
export const kindLabels: Record<TrustKind, string> = { path: '可执行文件路径', wildcard: '路径通配', hash: 'SHA256' };
export const BATCH_LIMIT = 200;

export function validateRule(kind: TrustKind, value: string): string {
  if (!value) return '条件不能为空';
  if (kind === 'hash') return /^[a-f\d]{64}$/i.test(value) ? '' : 'SHA256 必须是 64 位十六进制字符';
  if (!/^(?:[a-z]:[\\/]|\\\\[^\\/]+[\\/][^\\/]+[\\/])/i.test(value)) return '请输入 Windows 绝对路径或 UNC 路径';
  if (/[\x00-\x1f"<>|]/.test(value) || value.slice(2).includes(':')) return '路径含不支持的字符';
  if (kind === 'path' && /[*?]/.test(value)) return '含通配符的路径请选择“路径通配”类型';
  if (kind === 'path' && /[\\/]$/.test(value)) return '请指定可执行文件，不支持目录排除';
  if (kind === 'wildcard' && !/[*?]/.test(value)) return '路径通配至少需要一个 * 或 ?';
  return '';
}

export function ruleKey(kind: TrustKind, value: string): string {
  return `${kind}:${value.trim().toLowerCase()}`;
}

export type ParsedRule = { line: number; value: string; state: 'valid' | 'duplicate' | 'invalid'; reason: string };
export function parseRules(text: string, kind: TrustKind, existing: Set<string>): ParsedRule[] {
  const seen = new Set<string>();
  return text.split(/\r?\n/).flatMap<ParsedRule>((raw, index) => {
    const value = raw.trim();
    if (!value) return [];
    const reason = validateRule(kind, value);
    if (reason) return [{ line: index + 1, value, state: 'invalid' as const, reason }];
    const key = ruleKey(kind, value);
    const duplicate = existing.has(key) ? '规则已存在（含全局项）' : seen.has(key) ? '本次输入重复' : '';
    seen.add(key);
    return [{ line: index + 1, value, state: duplicate ? 'duplicate' as const : 'valid' as const, reason: duplicate }];
  });
}

export type OperationResult = { action: 'add' | 'delete'; success: number; skipped: number; failures: { value: string; reason: string }[] };

// A single in-flight request avoids flooding Kibana and preserves per-item results.
// Failed/ambiguous requests are never automatically retried.
export async function runSequential<T>(values: T[], perform: (value: T) => Promise<void>, describe: (value: T) => string,
  action: OperationResult['action'], progress: (done: number) => void): Promise<OperationResult> {
  const result: OperationResult = { action, success: 0, skipped: 0, failures: [] };
  for (const [index, value] of values.entries()) {
    try { await perform(value); result.success++; }
    catch (error) {
      const reason = String(error);
      if (action === 'add' && reason.includes('重复规则：')) result.skipped++;
      else result.failures.push({ value: describe(value), reason });
    }
    progress(index + 1);
  }
  return result;
}

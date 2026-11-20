import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import ts from 'typescript';

const source = readFileSync(new URL('../src/lib/trust.ts', import.meta.url), 'utf8');
const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } }).outputText;
const { validateRule, parseRules, ruleKey, runSequential } = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`);

test('validates Windows path, UNC, wildcard, SHA256 and rejects malformed rules', () => {
  for (const [kind, value] of [['path', 'C:\\App\\app.exe'], ['path', '\\\\server\\share\\app.exe'], ['wildcard', 'D:\\Games\\*'], ['hash', 'A'.repeat(64)]]) assert.equal(validateRule(kind, value), '');
  for (const [kind, value] of [['path', '..\\app.exe'], ['path', 'C:\\App\\*'], ['path', 'C:\\App\\'], ['path', 'C:\\App\\a.exe:stream'], ['wildcard', 'C:\\app.exe'], ['hash', 'G'.repeat(64)], ['hash', 'abc']]) assert.notEqual(validateRule(kind, value), '');
});
test('reports existing and input duplicates, invalid line numbers, and CRLF', () => {
  const rows = parseRules('C:\\App\\a.exe\r\n\r\nc:\\app\\A.EXE\r\nC:\\new.exe\r\nrelative\r\nC:\\new.exe', 'path', new Set([ruleKey('path', 'C:\\App\\a.exe')]));
  assert.deepEqual(rows.map(row => row.state), ['duplicate', 'duplicate', 'valid', 'invalid', 'duplicate']);
  assert.equal(rows[3].line, 5);
});
test('serial requests retain partial success, duplicates and failures without retries', async () => {
  let active = 0, maximum = 0;
  const calls = [], progress = [];
  const result = await runSequential(['ok', 'duplicate', 'bad', 'last'], async value => {
    calls.push(value); active++; maximum = Math.max(maximum, active);
    await Promise.resolve(); active--;
    if (value === 'duplicate') throw new Error('重复规则：已存在');
    if (value === 'bad') throw new Error('403 forbidden');
  }, value => value, 'add', done => progress.push(done));
  assert.equal(maximum, 1);
  assert.deepEqual(calls, ['ok', 'duplicate', 'bad', 'last']);
  assert.equal(result.success, 2); assert.equal(result.skipped, 1);
  assert.deepEqual(result.failures, [{ value: 'bad', reason: 'Error: 403 forbidden' }]);
  assert.deepEqual(progress, [1, 2, 3, 4]);
});
test('delete does not classify duplicate-like errors as skipped', async () => {
  const result = await runSequential(['id'], async () => { throw new Error('重复规则：error'); }, value => value, 'delete', () => {});
  assert.equal(result.failures.length, 1); assert.equal(result.skipped, 0);
});

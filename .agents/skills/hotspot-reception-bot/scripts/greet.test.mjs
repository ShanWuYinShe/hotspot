// 接待机器人单元测试:node greet.test.mjs
// 覆盖:接待时段 / 每日配额滚动 / engage 三层守卫 / 防投毒净化 / 人格配置完整性
import { readFileSync, existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { inWindow, dayKey, dailySent, bumpDaily, sanitizeHandle } from './greet.mjs';
import { engageBlocked } from './engage.mjs';

const HERE = dirname(fileURLToPath(import.meta.url));
const ROOT = dirname(HERE);

// ── 时段门控 ──
const cases = [];
for (let h = 0; h < 24; h++) {
  for (const m of [0, 30, 59]) {
    const d = new Date(2026, 8, 25, h, m, 0);
    cases.push([h, m, inWindow(d)]);
  }
}
const expect = (h) => (h >= 23 || h < 9);
let failed = 0;
for (const [h, m, got] of cases) {
  if (got !== expect(h)) { failed++; console.error(`FAIL ${h}:${String(m).padStart(2, '0')} 期望 ${expect(h)} 实际 ${got}`); }
}
const inWin = cases.filter(([, , g]) => g).length;

// ── 每日配额 ──
let qfail = 0;
const dq = (label, got, want) => { if (got !== want) { qfail++; console.error(`FAIL ${label}: 期望 ${want} 实际 ${got}`); } };
const d1 = new Date(2026, 8, 25, 23, 5);
const d2 = new Date(2026, 8, 26, 0, 5);
dq('dayKey 格式', dayKey(d1), '2026-09-25');
dq('dayKey 跨日', dayKey(d2), '2026-09-26');
const s0 = {};
dq('空状态已发 0', dailySent(s0, d1), 0);
bumpDaily(s0, d1); bumpDaily(s0, d1);
dq('当日累加', dailySent(s0, d1), 2);
dq('跨日清零', dailySent(s0, d2), 0);
bumpDaily(s0, d2);
dq('新日重新计数', dailySent(s0, d2), 1);
dq('旧日期干扰', s0.daily.date, '2026-09-26');

// ── engage 三层守卫 ──
const NOW = Date.parse('2026-09-25T21:00:00Z');
const H = 3600 * 1000, D = 24 * H;
const gf = (at, followed_up) => ({ at, followed_up });
const guardCases = [
  ['无任何记录 -> 可关怀', {}, false],
  ['关怀过 2 天前 -> 跳过', { engaged: { 'u': { at: new Date(NOW - 2 * D).toISOString() } } }, true],
  ['关怀过 4 天前 -> 可关怀', { engaged: { 'u': { at: new Date(NOW - 4 * D).toISOString() } } }, false],
  ['欢迎 12h 前 -> 跳过', { greeted: { 'u': gf(new Date(NOW - 12 * H).toISOString()) } }, true],
  ['欢迎 2 天前未跟进 -> 可跟进', { greeted: { 'u': gf(new Date(NOW - 2 * D).toISOString()) } }, false],
  ['欢迎 2 天前已跟进 -> 收手', { greeted: { 'u': gf(new Date(NOW - 2 * D).toISOString(), true) } }, true],
  ['欢迎 3 天前已跟进 -> 仍收手', { greeted: { 'u': gf(new Date(NOW - 3 * D).toISOString(), true) } }, true],
  ['关怀 4 天前+欢迎 2 天前 -> 可跟进', { engaged: { 'u': { at: new Date(NOW - 4 * D).toISOString() } }, greeted: { 'u': gf(new Date(NOW - 2 * D).toISOString()) } }, false],
];
let gfail = 0;
for (const [label, state, want] of guardCases) {
  const got = engageBlocked(state, 'u', NOW);
  if (got !== want) { gfail++; console.error(`FAIL ${label}: 期望 ${want} 实际 ${got}`); }
}

// ── 防投毒净化 ──
let sfail = 0;
const sd = (label, got, want) => { if (got !== want) { sfail++; console.error('FAIL ' + label + ': 期望 ' + JSON.stringify(want) + ' 实际 ' + JSON.stringify(got)); } };
sd('正常中文保留', sanitizeHandle('林小满'), '林小满');
sd('换行剥离', sanitizeHandle('小满\n第二行'), '小满第二行');
sd('控制字符剥离', sanitizeHandle('a\u0000b\u001fc'), 'abc');
sd('反引号剥离', sanitizeHandle('\u0060rm -rf\u0060'), 'rm -rf');
sd('shell危险字符剥离', sanitizeHandle('a;b|c&d<e>f'), 'abcdef');
sd('美元符剥离', sanitizeHandle('a$b'), 'ab');
sd('长度截断为20', sanitizeHandle('x'.repeat(50)).length, 20);
sd('洗后为空返回空串', sanitizeHandle('\n\n'), '');

// ── 人格配置完整性 ──
let pfail = 0;
const personasDir = join(ROOT, 'personas');
const personaFiles = existsSync(personasDir) ? readFileSync(0, 'utf8') && null : null;
const { readdirSync } = await import('node:fs');
const jsons = readdirSync(personasDir).filter((f) => f.endsWith('.json'));
for (const f of jsons) {
  const p = JSON.parse(readFileSync(join(personasDir, f), 'utf8'));
  const label = '人格 ' + f;
  for (const key of ['name', 'nickname', 'loginId', 'userId', 'deviceId', 'card']) {
    if (!p[key]) { pfail++; console.error('FAIL ' + label + ' 缺字段 ' + key); }
  }
  for (const key of ['welcome', 'engageNewWithMoments', 'engageNew', 'followUp']) {
    if (!Array.isArray(p.lines?.[key]) || p.lines[key].length === 0) { pfail++; console.error('FAIL ' + label + ' 缺话术组 ' + key); }
  }
  if (!existsSync(join(ROOT, p.card))) { pfail++; console.error('FAIL ' + label + ' 人物卡不存在 ' + p.card); }
}

console.log(`时段 ${cases.length}(窗口内 ${inWin}) / 配额 7 / 守卫 ${guardCases.length} / 净化 8 / 人格配置 ${jsons.length} 份,失败 ${failed + qfail + gfail + sfail + pfail}`);
process.exit(failed + qfail + gfail + sfail + pfail ? 1 : 0);

// 接待机器人单元测试:node greet.test.mjs
// 覆盖:接待时段 / 每日配额滚动 / engage 三层守卫 / 防投毒净化 / 人格配置完整性
import { readFileSync, existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { inWindow, dayKey, dailySent, bumpDaily, sanitizeHandle, addressableNick } from './greet.mjs';
import { engageBlocked } from './engage.mjs';
import { pickRoundRobin, resolvePersonaFor, listPersonas, personaPassword, loadPersona } from './persona.mjs';

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
// 可称呼判定:带情绪标点/过长 → 不称呼;正常短名 → 可称呼
sd('正常短名可称呼', addressableNick('举重冠军'), '举重冠军');
sd('带问号不称呼', addressableNick('你也通倭?'), '');
sd('带感叹号不称呼', addressableNick('冲呀!'), '');
sd('超12字截断仍可称呼', addressableNick('这是一个非常非常非常长的昵称啊').length, 12);
sd('空名不称呼', addressableNick(''), '');

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

// ── 自动人格分配:轮转分配 ──
let rfail = 0;
const rd = (label, got, want) => { if (got !== want) { rfail++; console.error('FAIL ' + label + ': 期望 ' + JSON.stringify(want) + ' 实际 ' + JSON.stringify(got)); } };

// 分布:3 人格连续分配 3 次 → 各得 1;共 9 次 → 各得 3;rrIndex 随分配递增
const NAMES = ['jiajing', 'linxiaoman', 'yanshifan'];
{
	const shared = {};
	const got = [0, 0, 0];
	for (let i = 0; i < 3; i++) got[NAMES.indexOf(pickRoundRobin(shared, NAMES))]++;
	rd('轮转 3 次→各1', got.join(','), '1,1,1');
	rd('rrIndex=3', shared.rrIndex, 3);
}
{
	const shared = {};
	const got = [0, 0, 0];
	for (let i = 0; i < 9; i++) got[NAMES.indexOf(pickRoundRobin(shared, NAMES))]++;
	rd('轮转 9 次→各3', got.join(','), '3,3,3');
	rd('rrIndex=9', shared.rrIndex, 9);
}
{
	const shared = { rrIndex: 7 };
	rd('续接 rrIndex=7 → 名单[1]', pickRoundRobin(shared, NAMES), NAMES[7 % 3]);
	rd('续接后 rrIndex=8', shared.rrIndex, 8);
}
{
	let threw = false;
	try { pickRoundRobin({}, []); } catch { threw = true; }
	rd('空名单抛错', threw, true);
}

// ── 自动人格分配:by 归属解析 ──
{
	const shared = { contacted: { u1: { by: 'yanshifan' } } };
	rd('by 合法归原人格', resolvePersonaFor(shared, 'u1', NAMES), 'yanshifan');
	rd('by 合法不推进轮转', shared.rrIndex ?? 0, 0);
	const s2 = { contacted: { u2: { by: 'system' } } };
	rd('by 非法回退默认人格', resolvePersonaFor(s2, 'u2', NAMES), 'linxiaoman');
	rd('by 非法不推进轮转', s2.rrIndex ?? 0, 0);
	const s3 = {};
	const pick = resolvePersonaFor(s3, 'u3', NAMES);
	rd('新用户轮转分配到合法人格', NAMES.includes(pick), true);
	rd('新用户已登记归属', s3.contacted.u3.by, pick);
	rd('登记 kind=auto_assign', s3.contacted.u3.kind, 'auto_assign');
	rd('登记推进 rrIndex=1', s3.rrIndex, 1);
	rd('再次解析归属固定', resolvePersonaFor(s3, 'u3', NAMES), pick);
	rd('归属固定不再推进', s3.rrIndex, 1);
}

// ── 显式 --persona:优先于 by 归属(同一用户可与多个人格各有会话,by 只是最早接触者)──
{
	const shared = { contacted: { u1: { by: 'jiajing' } } };
	rd('显式优先于 by 归属', resolvePersonaFor(shared, 'u1', NAMES, { explicit: 'yanshifan' }), 'yanshifan');
	rd('显式不推进轮转', shared.rrIndex ?? 0, 0);
	rd('显式不覆盖登记簿', shared.contacted.u1.by, 'jiajing');
	const s4 = {};
	rd('显式指定新用户直接采用', resolvePersonaFor(s4, 'u9', NAMES, { explicit: 'linxiaoman' }), 'linxiaoman');
	rd('显式新用户不登记 auto_assign', s4.contacted?.u9 ?? null, null);
	rd('显式新用户不推进轮转', s4.rrIndex ?? 0, 0);
	let threw = false;
	try { resolvePersonaFor({}, 'u', NAMES, { explicit: 'nope' }); } catch { threw = true; }
	rd('显式非法抛错', threw, true);
}

// ── 口令解析:WELCOME_PASSWORD_<大写人格名> 优先,回退 WELCOME_PASSWORD ──
{
	const savedP = process.env.WELCOME_PASSWORD, savedJ = process.env.WELCOME_PASSWORD_JIAJING, savedL = process.env.WELCOME_PASSWORD_LINXIAOMAN;
	delete process.env.WELCOME_PASSWORD; delete process.env.WELCOME_PASSWORD_JIAJING; delete process.env.WELCOME_PASSWORD_LINXIAOMAN;
	rd('无 env 时口令为空', personaPassword('jiajing') ?? '', '');
	process.env.WELCOME_PASSWORD = 'common-secret';
	rd('回退通用口令', personaPassword('jiajing'), 'common-secret');
	process.env.WELCOME_PASSWORD_JIAJING = 'jiajing-secret';
	rd('专属口令优先', personaPassword('jiajing'), 'jiajing-secret');
	rd('loadPersona 带专属口令', loadPersona('jiajing').password, 'jiajing-secret');
	rd('其他人格走通用回退', loadPersona('linxiaoman').password, 'common-secret');
	if (savedP === undefined) delete process.env.WELCOME_PASSWORD; else process.env.WELCOME_PASSWORD = savedP;
	if (savedJ === undefined) delete process.env.WELCOME_PASSWORD_JIAJING; else process.env.WELCOME_PASSWORD_JIAJING = savedJ;
	if (savedL === undefined) delete process.env.WELCOME_PASSWORD_LINXIAOMAN; else process.env.WELCOME_PASSWORD_LINXIAOMAN = savedL;
}

// ── 人格池:按文件名稳定排序 ──
{
	const pool = listPersonas();
	rd('人格池至少 3 份', pool.length >= 3, true);
	rd('文件名稳定排序', JSON.stringify(pool.map((p) => p.name)), JSON.stringify(pool.map((p) => p.name).slice().sort()));
}

// 段落计数(输出供值班核对,计数与实际用例数一致)
const sanitizeCases = 13; // 防投毒净化段实际用例数
const rrCases = 29;      // 轮转与归属解析段实际用例数
const totalFail = failed + qfail + gfail + sfail + pfail + rfail;
console.log(`时段 ${cases.length}(窗口内 ${inWin}) / 配额 7 / 守卫 ${guardCases.length} / 净化 ${sanitizeCases} / 人格配置 ${jsons.length} 份 / 轮转归属 ${rrCases},失败 ${totalFail}`);
process.exit(totalFail ? 1 : 0);

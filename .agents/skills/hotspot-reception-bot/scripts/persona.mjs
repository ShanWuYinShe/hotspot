// 人格加载与多人格状态管理(所有脚本共用)
// 人格 = personas/<name>.json(机器配置)+ personas/<name>.md(人物卡,回复前必读)
// 状态:state/state.<persona>.json(各自 greeted/engaged/daily)
//      state/shared-state.json(跨人格联系登记簿 + 自动分配轮转计数 rrIndex)
// 自动分配:新用户首次联系时按轮转(round-robin)选人格;WELCOME_PERSONA env
//      可强制锁定单人格(手动模式),锁定时轮转跳过;连续性铁律:谁欢迎,
//      跟进/回复/收件箱永远归属谁(shared.contacted[uid].by 为唯一事实源)。
import { readFileSync, writeFileSync, existsSync, readdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const HERE = dirname(fileURLToPath(import.meta.url));
const ROOT = dirname(HERE);
const PERSONA_DIR = join(ROOT, 'personas');
const STATE_DIR = join(ROOT, 'state');

export function activePersonaName() {
  return process.env.WELCOME_PERSONA || 'linxiaoman';
}

// 口令解析:优先 WELCOME_PASSWORD_<大写人格名>(如 WELCOME_PASSWORD_JIAJING,
// 多人格并发注入互不干扰),未设时回退 WELCOME_PASSWORD(单人格手动模式)。
// 口令只经 env 注入,禁止写入任何文件/commit。
export function personaPassword(name) {
  return process.env[`WELCOME_PASSWORD_${name.toUpperCase()}`] || process.env.WELCOME_PASSWORD;
}

// 加载人格配置;环境变量可逐项覆盖(账号迁移/临时接管场景)
export function loadPersona(name = activePersonaName()) {
  const file = join(PERSONA_DIR, `${name}.json`);
  if (!existsSync(file)) throw new Error(`人格配置不存在: ${file}`);
  const p = JSON.parse(readFileSync(file, 'utf8'));
  return {
    ...p,
    loginId: process.env.WELCOME_LOGIN_ID || p.loginId,
    password: personaPassword(name),
    userId: process.env.WELCOME_USER_ID || p.userId,
    deviceId: process.env.WELCOME_DEVICE_ID || p.deviceId,
  };
}

// 人格池:personas/*.json 全部,按文件名稳定排序(轮转顺序的可预期基础)
export function listPersonas() {
  return readdirSync(PERSONA_DIR)
    .filter((f) => f.endsWith('.json'))
    .sort()
    .map((f) => loadPersona(f.replace(/\.json$/, '')));
}

// 轮转分配(纯函数,可单测):pick = 名单[rrIndex % n],分配后计数 +1。
// rrIndex 持久在 shared(重启/跨脚本延续轮转位置)。
export function pickRoundRobin(shared, personaNames) {
  if (!Array.isArray(personaNames) || personaNames.length === 0) throw new Error('人格名单为空');
  shared.rrIndex ||= 0;
  const pick = personaNames[shared.rrIndex % personaNames.length];
  shared.rrIndex += 1;
  return pick;
}

// 回复归属解析(纯函数,可单测):用户已被人格接触过 → 永远归那个人格(连续性
// 铁律);by 不是合法人格名(system/admin 等)→ 回退默认人格;从未联系过(用户
// 主动私聊我们)→ 按轮转分配并当场登记,后续归属就此固定。
export function resolvePersonaFor(shared, uid, personaNames, fallback = 'linxiaoman') {
  const by = shared.contacted?.[uid]?.by;
  if (by) return personaNames.includes(by) ? by : fallback;
  const pick = pickRoundRobin(shared, personaNames);
  markContacted(shared, uid, pick, { kind: 'auto_assign' });
  return pick;
}

// 每人格独立状态文件(各自的欢迎/关怀记录与当日配额计数)
export function personaStatePath(name = activePersonaName()) {
  return join(STATE_DIR, `state.${name}.json`);
}

export function loadState(name = activePersonaName()) {
  const file = personaStatePath(name);
  const s = existsSync(file) ? JSON.parse(readFileSync(file, 'utf8')) : {};
  s.greeted ||= {};
  s.engaged ||= {};
  s.log ??= [];
  return s;
}

export function saveState(state, name = activePersonaName()) {
  writeFileSync(personaStatePath(name), JSON.stringify(state, null, 2));
}

// 跨人格共享登记簿:任何人格联系过的用户都记录在此,换人格不重复开场
const SHARED = join(STATE_DIR, 'shared-state.json');
export function loadShared() {
  const s = existsSync(SHARED) ? JSON.parse(readFileSync(SHARED, 'utf8')) : {};
  s.contacted ||= {};
  return s;
}
export function saveShared(shared) {
  writeFileSync(SHARED, JSON.stringify(shared, null, 2));
}
export function markContacted(shared, userId, by, extra = {}) {
  shared.contacted ||= {}; // 防御:resolvePersonaFor 等纯函数路径可能传入未初始化的 shared
  shared.contacted[userId] = { by, at: new Date().toISOString(), ...extra };
}

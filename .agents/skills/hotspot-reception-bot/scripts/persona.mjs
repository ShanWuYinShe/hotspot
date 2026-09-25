// 人格加载与多人格状态管理(所有脚本共用)
// 人格 = personas/<name>.json(机器配置)+ personas/<name>.md(人物卡,回复前必读)
// 状态:state/state.<persona>.json(各自 greeted/engaged/daily)
//      state/shared-state.json(跨人格联系登记簿,防止换人格后重复欢迎同一用户)
import { readFileSync, writeFileSync, existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const HERE = dirname(fileURLToPath(import.meta.url));
const ROOT = dirname(HERE);
const PERSONA_DIR = join(ROOT, 'personas');
const STATE_DIR = join(ROOT, 'state');

export function activePersonaName() {
  return process.env.WELCOME_PERSONA || 'linxiaoman';
}

// 加载人格配置;环境变量可逐项覆盖(账号迁移/临时接管场景)
export function loadPersona(name = activePersonaName()) {
  const file = join(PERSONA_DIR, `${name}.json`);
  if (!existsSync(file)) throw new Error(`人格配置不存在: ${file}`);
  const p = JSON.parse(readFileSync(file, 'utf8'));
  return {
    ...p,
    loginId: process.env.WELCOME_LOGIN_ID || p.loginId,
    password: process.env.WELCOME_PASSWORD,
    userId: process.env.WELCOME_USER_ID || p.userId,
    deviceId: process.env.WELCOME_DEVICE_ID || p.deviceId,
  };
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
  shared.contacted[userId] = { by, at: new Date().toISOString(), ...extra };
}

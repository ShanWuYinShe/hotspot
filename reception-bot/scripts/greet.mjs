#!/usr/bin/env node
// HotSpot 接待机器人 —— 新用户欢迎(greet)
// 用法: node greet.mjs [--dry-run] [--force]
//       WELCOME_PERSONA=<名> 可锁定单人格(手动模式,轮转跳过);不设则自动轮转分配
// 口令: WELCOME_PASSWORD_<大写人格名>(如 WELCOME_PASSWORD_JIAJING),未设回退 WELCOME_PASSWORD;只经 env 注入
// 数据源: WELCOME_DATA_SOURCE=db(默认,本地 docker 精确判定) | api(线上,走 admin API)
// 时段: 每天 23:00–次日 09:00(本机时区);--force 绕过
// 分配: 人格池 = personas/*.json(文件名稳定排序);新用户按 shared.rrIndex 轮转分配,
//       谁欢迎记入 shared.contacted[uid].by,后续跟进/回复永远归属该人格。
import { execFileSync } from 'node:child_process';
import { pathToFileURL } from 'node:url';
import {
  loadPersona, listPersonas, pickRoundRobin,
  loadState, saveState, loadShared, saveShared, markContacted, activePersonaName,
} from './persona.mjs';

const FORCE = process.argv.includes('--force');
const DRY = process.argv.includes('--dry-run');
const PERSONA = activePersonaName();
const P = loadPersona(PERSONA); // 默认人格(向后兼容:参数化函数的缺省值)
const DATA_SOURCE = process.env.WELCOME_DATA_SOURCE || 'db';
const BASE = process.env.WELCOME_BASE_URL || 'http://localhost:8888';
const DB_CONTAINER = process.env.WELCOME_DB_CONTAINER || 'hotspot-postgres-1';
const ADMIN_LOGIN_ID = process.env.WELCOME_ADMIN_LOGIN_ID || 'ops_admin';
const ADMIN_PASSWORD = process.env.WELCOME_ADMIN_PASSWORD;

// ── 时段 ────────────────────────────────────────────────────────────────
// 接待窗口:每天 23:00 起,次日 09:00 结束(本机时区判定)
export function inWindow(d = new Date()) {
  const h = d.getHours();
  return h >= 23 || h < 9;
}

// ── 配额计数(自然日本地日) ─────────────────────────────────────────────
export function dayKey(d = new Date()) {
  const p = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}
export function dailySent(state, d = new Date()) {
  return state.daily && state.daily.date === dayKey(d) ? state.daily.sent : 0;
}
export function bumpDaily(state, d = new Date()) {
  const date = dayKey(d);
  state.daily = { date, sent: (state.daily && state.daily.date === date ? state.daily.sent : 0) + 1 };
}

// ── 防投毒净化:用户可控内容进入话术前必须过滤 ─────────────────────────
export function sanitizeHandle(name, max = 20) {
  if (!name) return '';
  return String(name)
    .replace(/[\u0000-\u001f\u007f]/g, '')
    .replace(/[`$<>\\|;&]/g, '')
    .trim()
    .slice(0, max);
}

// 可称呼判定:昵称带情绪标点(?!!~…)或过长时,嵌入话术会产生歧义/冒犯
// (实测:昵称「你也通倭?」嵌入欢迎语变成「你好呀,你也通倭?,欢迎加入」,读来像骂人),此时降级为不称呼
export function addressableNick(name, max = 12) {
  const s = sanitizeHandle(name, max);
  if (!s) return '';
  if (/[?!?!~\u2026\u3002,.,\s]/.test(s)) return '';
  return s;
}

// ── HTTP ───────────────────────────────────────────────────────────────
export async function api(path, body, token, idem) {
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = 'Bearer ' + token;
  if (idem) headers['Idempotency-Key'] = idem;
  const res = await fetch(BASE + path, { method: 'POST', headers, body: JSON.stringify(body) });
  return res.json().catch(() => ({}));
}

// 登录态缓存:token 有效期内(余量 ≥120s)复用,砍掉高频轮次的重复登录(平台鉴权限流 60 次/10min/IP)
// 缓存挂在 state(gitignored 运行时文件,TTL ≤15 分钟),不新增任何落盘面。
// persona 参数化:多人格各用各的账号/deviceId 登录、各写各的 state。
export async function signin(state, persona = P, deviceIdOverride = null) {
  const deviceId = deviceIdOverride || persona.deviceId;
  if (state?.auth?.token && state?.auth?.expiresAt) {
    const remainMs = Date.parse(state.auth.expiresAt) - Date.now();
    if (remainMs > 120 * 1000) {
      return { access_token: state.auth.token, user_id: state.auth.userId, cached: true };
    }
  }
  const attempt = async () => api('/public/v1/auth/signin', {
    login_id: persona.loginId, password: persona.password, client_type: 'app', device_id: deviceId,
  });
  let r = await attempt();
  // 高频轮次会触发 IP 限流(10500,窗口 10 分钟),退避 60s 后重试一次
  if (r.code === 10500) {
    console.log('触发限流(10500),等待 60s 重试…');
    await new Promise((res) => setTimeout(res, 60000));
    r = await attempt();
  }
  if (r.code !== 0) throw new Error(`登录失败[${persona.name}]: ${JSON.stringify(r)}`);
  if (state) {
    state.auth = { token: r.data.access_token, expiresAt: r.data.access_token_expires_at, userId: r.data.user_id };
  }
  return r.data;
}

// 鉴权请求自愈:同一人格账号/deviceId 被其他脚本或轮次重新登录时会顶掉旧会话
// (10100),登录缓存只能缓解不能根治。此封装在 10100 时清掉 state.auth 缓存、
// 强制重登录取新 token 并重试一次;state.auth 已同步刷新,调用方后续请求从
// state.auth.token 取值即为新会话。
export async function apiAuthed(state, persona, path, body, token, idem) {
  let r = await api(path, body, token, idem);
  if (r.code === 10100) {
    delete state.auth;
    const me = await signin(state, persona);
    r = await api(path, body, me.access_token, idem);
  }
  return r;
}

// ── 当日已发计数:DB 模式以出站消息为准;API 模式以共享登记簿为准 ────────
export function dbOutboundToday(persona = P) {
  if (DATA_SOURCE !== 'db') return 0;
  const midnight = new Date();
  midnight.setHours(0, 0, 0, 0);
  return Number(execFileSync('docker', [
    'exec', DB_CONTAINER, 'psql', '-U', 'postgres', '-d', 'hotspot', '-t', '-A', '-c',
    `SELECT count(*) FROM private_messages WHERE sender_user_id = '${persona.userId}' AND sent_at >= '${midnight.toISOString()}'`,
  ], { encoding: 'utf8' }).trim() || '0');
}
export function sharedSentToday(shared, personaName = PERSONA) {
  const today = dayKey();
  return Object.values(shared.contacted).filter((c) => c.by === personaName && c.at.startsWith(today)).length;
}
export function effectiveDailySent(state, shared, persona = P) {
  return Math.max(dailySent(state), DATA_SOURCE === 'db' ? dbOutboundToday(persona) : sharedSentToday(shared, persona.name));
}

// ── 候选数据源 ──────────────────────────────────────────────────────────
// DB 模式:直查库(排除全部人格账号/已删/封禁)
export function dbUsers(excludeIds = [P.userId]) {
  const notIn = excludeIds.map((id) => `'${id}'`).join(',');
  return execFileSync('docker', [
    'exec', DB_CONTAINER, 'psql', '-U', 'postgres', '-d', 'hotspot', '-t', '-A', '-F', '|', '-c',
    `SELECT user_id, nickname, user_type, created_at FROM users WHERE is_deleted = false AND status = 'normal' AND user_id NOT IN (${notIn}) ORDER BY created_at ASC;`,
  ], { encoding: 'utf8' }).trim().split('\n').filter(Boolean).map((line) => {
    const [user_id, nickname, user_type, created_at] = line.split('|');
    return { user_id, nickname, user_type, created_at };
  });
}
// API 模式(线上):admin users/search 默认 created_at DESC,翻首页取最近注册
export async function apiRecentUsers(limit = 100) {
  const admin = await api('/public/v1/auth/signin', {
    login_id: ADMIN_LOGIN_ID, password: ADMIN_PASSWORD, client_type: 'app',
    device_id: process.env.WELCOME_ADMIN_DEVICE_ID || '00000000-0000-4000-8000-000000000001',
  });
  if (admin.code !== 0) throw new Error('管理员登录失败: ' + JSON.stringify(admin));
  const r = await api('/admin/api/v1/users/search', {
    user_id: null, nickname: null, login_id: null, status: 'normal', user_type: null,
    pagination: { limit },
  }, admin.data.access_token);
  if (r.code !== 0) throw new Error('admin users/search 失败: ' + JSON.stringify(r));
  return (r.data?.items ?? []).map((it) => ({
    user_id: it.user.user_id, nickname: it.user.nickname,
    user_type: it.user_type, created_at: it.created_at,
  }));
}

// DB 兜底:每个用户已收到的「各人格出站消息」条数(state 丢失时防重复欢迎)。
// 一次查询按发送者聚合,返回 Map<personaName, Map<otherUserId, count>>。
export function dbOutboundCounts(personas = [P]) {
  const out = new Map();
  if (DATA_SOURCE !== 'db') return out;
  const nameOf = new Map(personas.map((p) => [p.userId, p.name]));
  const ids = personas.map((p) => `'${p.userId}'`).join(',');
  const rows = execFileSync('docker', [
    'exec', DB_CONTAINER, 'psql', '-U', 'postgres', '-d', 'hotspot', '-t', '-A', '-F', '|', '-c',
    `SELECT pm.sender_user_id,
            CASE WHEN pc.user_a_id = pm.sender_user_id THEN pc.user_b_id ELSE pc.user_a_id END AS other_id,
            count(*)
      FROM private_chats pc
      JOIN private_messages pm ON pm.chat_key = pc.chat_key
      WHERE pm.sender_user_id IN (${ids})
      GROUP BY 1, 2;`,
  ], { encoding: 'utf8' }).trim().split('\n').filter(Boolean);
  for (const row of rows) {
    const [sender, other, n] = row.split('|');
    const name = nameOf.get(sender);
    if (!name) continue;
    if (!out.has(name)) out.set(name, new Map());
    out.get(name).set(other, Number(n));
  }
  return out;
}

// 话术渲染:personas/<名>.json lines + {nick}/{tag} 占位符(值均经 sanitizeHandle)
export function renderLine(tpl, nick, tag) {
  return tpl.replaceAll('{nick}', nick ? '，' + nick : '').replaceAll('{tag}', tag ?? '').replaceAll('{tagPart}', tag ? `你那个「#${tag}」的标签跟我太对口了,` : '');
}

async function main() {
  const personas = listPersonas();
  const locked = process.env.WELCOME_PERSONA || ''; // 手动模式:锁定单人格,轮转跳过
  const pool = locked ? personas.filter((p) => p.name === locked) : personas;
  if (locked && !pool.length) { console.error(`WELCOME_PERSONA=${locked} 不在人格池中(可用: ${personas.map((p) => p.name).join(', ')})`); process.exit(1); }
  const active = pool.filter((p) => p.password);
  for (const p of pool) {
    if (!p.password) console.error(`[警告] ${p.name} 未设置口令(WELCOME_PASSWORD_${p.name.toUpperCase()}),本轮跳过该人格`);
  }
  if (!active.length) { console.error('没有任何已设置口令的人格,无法执行'); process.exit(1); }
  const names = active.map((p) => p.name);

  if (!FORCE && !inWindow()) {
    console.log(`当前 ${new Date().toLocaleString('zh-CN')} 不在接待时段(23:00–次日09:00),跳过。加 --force 可强制执行。`);
    console.log(JSON.stringify({ sent: 0, skipped: 0, failed: [], out_of_window: true }));
    return;
  }
  const shared = loadShared();
  const states = {};
  for (const p of active) states[p.name] = loadState(p.name);
  const sessions = {};
  for (const p of active) {
    sessions[p.name] = await signin(states[p.name], p);
    console.log(`[${p.name}] 登录成功, user_id = ${sessions[p.name].user_id} (数据源: ${DATA_SOURCE})`);
  }

  let cap = 10;
  const cfg = await api('/public/v1/system/client_config', {});
  if (cfg.code === 0 && cfg.data?.limits?.stranger_message_daily_quota) {
    cap = cfg.data.limits.stranger_message_daily_quota;
  }

  const raw = DATA_SOURCE === 'db' ? dbUsers(personas.map((p) => p.userId)) : await apiRecentUsers();
  const outbound = DATA_SOURCE === 'db' ? dbOutboundCounts(active) : new Map();
  // 多重防重:跨人格共享登记簿 + 各人格 state.greeted(含 skipped)+ DB 出站兜底(仅 db 模式)
  const users = raw.filter((u) =>
    !shared.contacted[u.user_id]
    && !active.some((p) => states[p.name].greeted[u.user_id])
    && !active.some((p) => (outbound.get(p.name)?.get(u.user_id) ?? 0) >= 1)
  );
  console.log(`待接待新用户 ${users.length} 个(人格池: ${names.join(' → ')})`);
  if (!users.length) { console.log(JSON.stringify({ sent: 0, skipped: 0, failed: [] })); return; }

  let sent = 0, skipped = 0; const failed = [];
  const sentBy = {}; // 各人格本轮发送数(欢迎话术在各自模板组内轮换)
  for (const u of users) {
    const name = locked ? names[0] : pickRoundRobin(shared, names);
    const persona = active.find((x) => x.name === name);
    const st = states[name];
    if (effectiveDailySent(st, shared, persona) >= cap) {
      console.log(`[${name}] 今日陌生人配额(${cap} 条)已用完,${u.user_id} 留到下个窗口`);
      continue;
    }
    const nick = u.nickname.startsWith('游客·') ? '' : addressableNick(u.nickname); // 语义怪/带情绪标点昵称降级不称呼(防投毒)
    const templates = persona.lines.welcome;
    const perCount = sentBy[name] || 0;
    const content = renderLine(templates[perCount % templates.length], nick);
    if (DRY) { console.log('[dry-run]', name, '->', u.user_id, u.nickname); sent++; sentBy[name] = perCount + 1; continue; }
    const r = await apiAuthed(st, persona, '/api/v1/private_messages/send', {
      target_user_id: u.user_id, type: 'text', content,
      file_id: null, share_code: null, reply_to_message_id: null,
    }, st.auth?.token, crypto.randomUUID());
    if (r.code === 0) {
      sent++; sentBy[name] = perCount + 1;
      bumpDaily(st);
      st.greeted[u.user_id] = { at: new Date().toISOString(), message_id: r.data?.message_id };
      markContacted(shared, u.user_id, name, { message_id: r.data?.message_id, kind: 'welcome' });
      console.log(`[${name}] 已欢迎`, u.user_id, u.nickname);
    } else if (r.code === 10200) {
      const reason = r.data?.forbidden_reason || 'unknown';
      if (reason === 'stranger_quota_exceeded') {
        console.log(`[${name}] 陌生人配额耗尽,${u.user_id} 留待配额恢复`);
        continue;
      }
      skipped++;
      st.greeted[u.user_id] = { at: new Date().toISOString(), skipped: reason };
      console.log(`[${name}] 跳过`, u.user_id, reason);
    } else {
      failed.push({ user_id: u.user_id, code: r.code, message: r.message, by: name });
      console.error('失败', u.user_id, JSON.stringify(r));
    }
    saveState(st, name);
    saveShared(shared);
  }
  console.log(JSON.stringify({ sent, skipped, failed, by_persona: sentBy }));
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  // 顶层异常包装:崩溃栈对值班无信息量,输出干净一行错误(下轮自动重试)
  main().catch((e) => { console.error('执行异常:', e?.message || e); process.exit(3); });
}

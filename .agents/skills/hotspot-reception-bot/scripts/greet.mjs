#!/usr/bin/env node
// HotSpot 接待机器人 —— 新用户欢迎(greet)
// 用法: WELCOME_PASSWORD=<人格口令> node greet.mjs [--dry-run] [--force] [--persona <名>]
// 数据源: WELCOME_DATA_SOURCE=db(默认,本地 docker 精确判定) | api(线上,走 admin API)
// 时段: 每天 23:00–次日 09:00(本机时区);--force 绕过。口令只经环境变量注入。
import { execFileSync } from 'node:child_process';
import { pathToFileURL } from 'node:url';
import {
  loadPersona, loadState, saveState, loadShared, saveShared, markContacted, activePersonaName,
} from './persona.mjs';

const FORCE = process.argv.includes('--force');
const DRY = process.argv.includes('--dry-run');
const PERSONA = activePersonaName();
const P = loadPersona(PERSONA);
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
// 缓存挂在 state(gitignored 运行时文件,TTL ≤15 分钟),不新增任何落盘面
export async function signin(state) {
  if (state?.auth?.token && state?.auth?.expiresAt) {
    const remainMs = Date.parse(state.auth.expiresAt) - Date.now();
    if (remainMs > 120 * 1000) {
      return { access_token: state.auth.token, user_id: state.auth.userId, cached: true };
    }
  }
  const attempt = async () => api('/public/v1/auth/signin', {
    login_id: P.loginId, password: P.password, client_type: 'app', device_id: P.deviceId,
  });
  let r = await attempt();
  // 高频轮次会触发 IP 限流(10500,窗口 10 分钟),退避 60s 后重试一次
  if (r.code === 10500) {
    console.log('触发限流(10500),等待 60s 重试…');
    await new Promise((res) => setTimeout(res, 60000));
    r = await attempt();
  }
  if (r.code !== 0) throw new Error('登录失败: ' + JSON.stringify(r));
  if (state) {
    state.auth = { token: r.data.access_token, expiresAt: r.data.access_token_expires_at, userId: r.data.user_id };
  }
  return r.data;
}

// ── 当日已发计数:DB 模式以出站消息为准;API 模式以共享登记簿为准 ────────
export function dbOutboundToday() {
  if (DATA_SOURCE !== 'db') return 0;
  const midnight = new Date();
  midnight.setHours(0, 0, 0, 0);
  return Number(execFileSync('docker', [
    'exec', DB_CONTAINER, 'psql', '-U', 'postgres', '-d', 'hotspot', '-t', '-A', '-c',
    `SELECT count(*) FROM private_messages WHERE sender_user_id = '${P.userId}' AND sent_at >= '${midnight.toISOString()}'`,
  ], { encoding: 'utf8' }).trim() || '0');
}
export function sharedSentToday(shared) {
  const today = dayKey();
  return Object.values(shared.contacted).filter((c) => c.by === PERSONA && c.at.startsWith(today)).length;
}
export function effectiveDailySent(state, shared) {
  return Math.max(dailySent(state), DATA_SOURCE === 'db' ? dbOutboundToday() : sharedSentToday(shared));
}

// ── 候选数据源 ──────────────────────────────────────────────────────────
// DB 模式:直查库(排除自己/已删/封禁)
export function dbUsers() {
  return execFileSync('docker', [
    'exec', DB_CONTAINER, 'psql', '-U', 'postgres', '-d', 'hotspot', '-t', '-A', '-F', '|', '-c',
    `SELECT user_id, nickname, user_type, created_at FROM users WHERE is_deleted = false AND status = 'normal' AND user_id <> '${P.userId}' ORDER BY created_at ASC;`,
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

// DB 兜底:每个用户已收到的「本人格出站消息」条数(state 丢失时防重复欢迎)
export function dbOutboundCounts() {
  if (DATA_SOURCE !== 'db') return new Map();
  return execFileSync('docker', [
    'exec', DB_CONTAINER, 'psql', '-U', 'postgres', '-d', 'hotspot', '-t', '-A', '-F', '|', '-c',
    `SELECT CASE WHEN pc.user_a_id = '${P.userId}' THEN pc.user_b_id ELSE pc.user_a_id END AS other_id, count(*)
      FROM private_chats pc
      JOIN private_messages pm ON pm.chat_key = pc.chat_key
      WHERE (pc.user_a_id = '${P.userId}' OR pc.user_b_id = '${P.userId}')
        AND pm.sender_user_id = '${P.userId}'
      GROUP BY 1;`,
  ], { encoding: 'utf8' }).trim().split('\n').filter(Boolean).reduce((m, line) => {
    const [id, n] = line.split('|');
    m.set(id, Number(n));
    return m;
  }, new Map());
}

// 话术渲染:personas/<名>.json lines + {nick}/{tag} 占位符(值均经 sanitizeHandle)
function renderLine(tpl, nick, tag) {
  return tpl.replaceAll('{nick}', nick ? '，' + nick : '').replaceAll('{tag}', tag ?? '').replaceAll('{tagPart}', tag ? `你那个「#${tag}」的标签跟我太对口了,` : '');
}

async function main() {
  if (!P.password) { console.error('缺少 WELCOME_PASSWORD 环境变量'); process.exit(1); }
  if (!FORCE && !inWindow()) {
    console.log(`当前 ${new Date().toLocaleString('zh-CN')} 不在接待时段(23:00–次日09:00),跳过。加 --force 可强制执行。`);
    console.log(JSON.stringify({ sent: 0, skipped: 0, failed: [], out_of_window: true }));
    return;
  }
  const state = loadState();
  const shared = loadShared();
  const me = await signin(state);
  const token = me.access_token;
  console.log(`[${PERSONA}] 登录成功, user_id =`, me.user_id, `(数据源: ${DATA_SOURCE})`);

  let cap = 10;
  const cfg = await api('/public/v1/system/client_config', {});
  if (cfg.code === 0 && cfg.data?.limits?.stranger_message_daily_quota) {
    cap = cfg.data.limits.stranger_message_daily_quota;
  }

  const raw = DATA_SOURCE === 'db' ? dbUsers() : await apiRecentUsers();
  const outbound = dbOutboundCounts();
  // 双重防重:本人格 state + 跨人格共享登记簿 + DB 出站兜底(仅 db 模式)
  const users = raw.filter((u) =>
    !state.greeted[u.user_id] && !shared.contacted[u.user_id] && !outbound.has(u.user_id)
  );
  console.log(`待接待新用户 ${users.length} 个,今日剩余配额 ${cap - effectiveDailySent(state, shared)} 条`);
  if (!users.length) { console.log(JSON.stringify({ sent: 0, skipped: 0, failed: [] })); return; }

  let sent = 0, skipped = 0; const failed = [];
  for (const u of users) {
    if (effectiveDailySent(state, shared) >= cap) {
      console.log(`今日陌生人配额(${cap} 条)已用完,剩余 ${users.length - sent - skipped} 人留到下个窗口`);
      break;
    }
    const nick = u.nickname.startsWith('游客·') ? '' : addressableNick(u.nickname); // 语义怪/带情绪标点昵称降级不称呼(防投毒)
    const templates = P.lines.welcome;
    const content = renderLine(templates[sent % templates.length], nick);
    if (DRY) { console.log('[dry-run] ->', u.user_id, u.nickname); sent++; continue; }
    const r = await api('/api/v1/private_messages/send', {
      target_user_id: u.user_id, type: 'text', content,
      file_id: null, share_code: null, reply_to_message_id: null,
    }, token, crypto.randomUUID());
    if (r.code === 0) {
      sent++;
      bumpDaily(state);
      state.greeted[u.user_id] = { at: new Date().toISOString(), message_id: r.data?.message_id };
      markContacted(shared, u.user_id, PERSONA, { message_id: r.data?.message_id, kind: 'welcome' });
      console.log('已欢迎', u.user_id, u.nickname);
    } else if (r.code === 10200) {
      const reason = r.data?.forbidden_reason || 'unknown';
      if (reason === 'stranger_quota_exceeded') {
        console.log(`陌生人配额耗尽,剩余 ${users.length - sent - skipped} 人留待配额恢复`);
        break;
      }
      skipped++;
      state.greeted[u.user_id] = { at: new Date().toISOString(), skipped: reason };
      console.log('跳过', u.user_id, reason);
    } else {
      failed.push({ user_id: u.user_id, code: r.code, message: r.message });
      console.error('失败', u.user_id, JSON.stringify(r));
    }
    saveState(state);
    saveShared(shared);
  }
  console.log(JSON.stringify({ sent, skipped, failed }));
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  // 顶层异常包装:崩溃栈对值班无信息量,输出干净一行错误(下轮自动重试)
  main().catch((e) => { console.error('执行异常:', e?.message || e); process.exit(3); });
}

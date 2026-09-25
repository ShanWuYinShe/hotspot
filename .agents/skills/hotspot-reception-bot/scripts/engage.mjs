#!/usr/bin/env node
// HotSpot 接待机器人 —— 在线无人交互用户关怀(engage)
// 用法: WELCOME_PASSWORD=<人格口令> node engage.mjs [--dry-run] [--force] [--persona <名>]
// 「无人交互」判定:DB 模式为诚实保守口径(从未收到除本人格外任何人的私聊,瞬间无他人的评论/反应);
//                   API 模式为降级口径(在线普通用户 ∉ 已联系登记簿 ∉ 本账号已有会话)。
// 防打扰:关怀间隔 3 天;欢迎 24h 内不贴脸;跟进只做一次;跨人格共享联系登记簿。
import { execFileSync } from 'node:child_process';
import { pathToFileURL } from 'node:url';
import {
  loadPersona, loadState, saveState, loadShared, saveShared, markContacted, activePersonaName,
} from './persona.mjs';
import { api, signin, sanitizeHandle, addressableNick, inWindow, dailySent, bumpDaily, effectiveDailySent, dayKey } from './greet.mjs';

const FORCE = process.argv.includes('--force');
const DRY = process.argv.includes('--dry-run');
const PERSONA = activePersonaName();
const P = loadPersona(PERSONA);
const DATA_SOURCE = process.env.WELCOME_DATA_SOURCE || 'db';
const BASE = process.env.WELCOME_BASE_URL || 'http://localhost:8888';
const DB_CONTAINER = process.env.WELCOME_DB_CONTAINER || 'hotspot-postgres-1';
const ENGAGE_GAP_MS = 3 * 24 * 3600 * 1000;
const RECENT_WELCOME_MS = 24 * 3600 * 1000;

// 三层守卫(纯函数,可单测):①关怀 3 天间隔 ②跟进一次就收手 ③欢迎 24h 内不贴脸
export function engageBlocked(state, uid, now = Date.now()) {
  const g = state.greeted?.[uid];
  const e = state.engaged?.[uid];
  if (e && now - Date.parse(e.at) < ENGAGE_GAP_MS) return true;
  if (g && g.followed_up) return true;
  if (g && now - Date.parse(g.at) < RECENT_WELCOME_MS) return true;
  return false;
}

// DB 模式:在线且无人交互的用户(最新一条瞬间带一个标签用于话术)
const SQL = `
WITH cand AS (
  SELECT u.user_id, u.nickname, u.created_at,
         (SELECT t.tag_name FROM moments m
          JOIN moment_tags t ON t.moment_id = m.moment_id
          WHERE m.user_id = u.user_id AND m.is_deleted = false
          ORDER BY m.created_at DESC LIMIT 1) AS last_tag,
         EXISTS (SELECT 1 FROM moments m WHERE m.user_id = u.user_id AND m.is_deleted = false) AS has_moments
  FROM users u
  WHERE u.is_deleted = false AND u.status = 'normal' AND u.user_id <> '${P.userId}'
    AND u.presence_status = 'online' AND u.last_online > now() - interval '15 minutes'
)
SELECT c.user_id, c.nickname, c.last_tag, c.has_moments
FROM cand c
WHERE NOT EXISTS (
  SELECT 1 FROM private_chats pc
  JOIN private_messages pm ON pm.chat_key = pc.chat_key
  WHERE (pc.user_a_id = c.user_id OR pc.user_b_id = c.user_id)
    AND pm.sender_user_id <> c.user_id AND pm.sender_user_id <> '${P.userId}'
) AND NOT EXISTS (
  SELECT 1 FROM moments m JOIN comments cm ON cm.moment_id = m.moment_id
  WHERE m.user_id = c.user_id AND m.is_deleted = false AND cm.is_deleted = false
    AND cm.user_id <> c.user_id AND cm.user_id <> '${P.userId}'
) AND NOT EXISTS (
  SELECT 1 FROM moments m JOIN reactions r ON r.target_type = 'moment' AND r.target_id = m.moment_id
  WHERE m.user_id = c.user_id AND m.is_deleted = false
    AND r.user_id <> c.user_id AND r.user_id <> '${P.userId}'
) AND NOT EXISTS (
  SELECT 1 FROM private_chats pc2 JOIN private_messages pm2 ON pm2.chat_key = pc2.chat_key
  WHERE (pc2.user_a_id = c.user_id OR pc2.user_b_id = c.user_id)
    AND pm2.sender_user_id = c.user_id
)
ORDER BY c.created_at ASC;
`;

function lonelyUsersDb() {
  return execFileSync('docker', [
    'exec', DB_CONTAINER, 'psql', '-U', 'postgres', '-d', 'hotspot', '-t', '-A', '-F', '|', '-c', SQL,
  ], { encoding: 'utf8' }).trim().split('\n').filter(Boolean).map((line) => {
    const [user_id, nickname, last_tag, has_moments] = line.split('|');
    return { user_id, nickname, last_tag: last_tag || null, has_moments: has_moments === 't' };
  });
}

// API 模式(线上降级口径):在线普通用户,排除共享登记簿与本账号已有会话
async function onlineUsersApi() {
  const admin = await api('/public/v1/auth/signin', {
    login_id: process.env.WELCOME_ADMIN_LOGIN_ID || 'ops_admin',
    password: process.env.WELCOME_ADMIN_PASSWORD,
    client_type: 'app',
    device_id: process.env.WELCOME_ADMIN_DEVICE_ID || '00000000-0000-4000-8000-000000000001',
  });
  if (admin.code !== 0) throw new Error('管理员登录失败: ' + JSON.stringify(admin));
  const r = await api('/admin/api/v1/users/search', {
    user_id: null, nickname: null, login_id: null, status: 'normal', user_type: null,
    pagination: { limit: 100 },
  }, admin.data.access_token);
  if (r.code !== 0) throw new Error('admin users/search 失败: ' + JSON.stringify(r));
  const now = Date.now();
  return (r.data?.items ?? [])
    .filter((it) => it.user.presence_status === 'online'
      && it.last_active_at && now - Date.parse(it.last_active_at) < 15 * 60 * 1000
      && it.user.user_id !== P.userId)
    .map((it) => ({ user_id: it.user.user_id, nickname: it.user.nickname, last_tag: null, has_moments: false }));
}

// 本账号已有会话的对端(API 模式排除用;避免对本就在聊天的人陌生开场)
async function myChatPeers(token) {
  const r = await api('/api/v1/private_chats/fetch', { pagination: null }, token);
  if (r.code !== 0) return new Set();
  return new Set((r.data?.items ?? []).map((c) => c.other_user?.user_id).filter(Boolean));
}

function renderLine(tpl, nick, tag) {
  return tpl.replaceAll('{nick}', nick ? '，' + nick : '').replaceAll('{tag}', tag ?? '').replaceAll('{tagPart}', tag ? `你那个「#${tag}」的标签跟我太对口了,` : '');
}

async function main() {
  if (!P.password) { console.error('缺少 WELCOME_PASSWORD 环境变量'); process.exit(1); }
  if (!FORCE && !inWindow()) {
    console.log(`当前 ${new Date().toLocaleString('zh-CN')} 不在接待时段(23:00–次日09:00),跳过。`);
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

  const now = Date.now();
  const raw = DATA_SOURCE === 'db' ? lonelyUsersDb() : await onlineUsersApi();
  const peers = DATA_SOURCE === 'api' ? await myChatPeers(token) : null;
  const outbound = DATA_SOURCE === 'db' ? (await import('./greet.mjs')).dbOutboundCounts() : new Map();
  const users = raw.filter((u) => {
    if (engageBlocked(state, u.user_id, now)) return false;
    if (shared.contacted[u.user_id]) return false;                 // 跨人格:任何人格联系过就不陌生开场
    if (peers && peers.has(u.user_id)) return false;               // API 模式:本账号已有会话
    if (DATA_SOURCE === 'db' && state.greeted[u.user_id] && (outbound.get(u.user_id) ?? 0) >= 2) return false;
    return true;
  });
  console.log(`在线且无人交互、可关怀的用户 ${users.length} 个,今日已发 ${effectiveDailySent(state, shared)} / 上限 ${cap} 条`);
  if (!users.length) { console.log(JSON.stringify({ sent: 0, skipped: 0, failed: [] })); return; }

  let sent = 0, skipped = 0; const failed = [];
  for (const u of users) {
    if (effectiveDailySent(state, shared) >= cap) {
      console.log(`今日陌生人配额(${cap} 条)已用完,剩余 ${users.length - sent - skipped} 人留到下个窗口`);
      break;
    }
    const g = state.greeted[u.user_id];
    const isFollowUp = Boolean(g); // 欢迎过没回应 → 轻声跟进;没欢迎过(注册后未到窗口) → 顺带完成欢迎
    const safeTag = addressableNick(u.last_tag ?? '', 20); // 防投毒:用户可控标签净化后才进话术
    const lines = isFollowUp ? P.lines.followUp
      : (u.has_moments ? P.lines.engageNewWithMoments : P.lines.engageNew);
    const content = renderLine(lines[sent % lines.length], sanitizeHandle(u.nickname), safeTag);
    if (DRY) { console.log('[dry-run] ->', u.user_id, u.nickname, '|', content.slice(0, 24) + '…'); sent++; continue; }
    const r = await api('/api/v1/private_messages/send', {
      target_user_id: u.user_id, type: 'text', content,
      file_id: null, share_code: null, reply_to_message_id: null,
    }, token, crypto.randomUUID());
    if (r.code === 0) {
      sent++;
      bumpDaily(state);
      state.engaged[u.user_id] = { at: new Date().toISOString(), message_id: r.data?.message_id };
      markContacted(shared, u.user_id, PERSONA, { message_id: r.data?.message_id, kind: isFollowUp ? 'follow_up' : 'engage' });
      if (isFollowUp) state.greeted[u.user_id].followed_up = true; // 跟进只做一次
      else if (!g) state.greeted[u.user_id] = { at: new Date().toISOString(), via: 'engage' }; // 顺带欢迎
      console.log('已关怀', u.user_id, u.nickname);
    } else if (r.code === 10200) {
      const reason = r.data?.forbidden_reason || 'unknown';
      if (reason === 'stranger_quota_exceeded') {
        console.log(`陌生人配额耗尽,剩余 ${users.length - sent - skipped} 人留待配额恢复`);
        break;
      }
      skipped++;
      state.engaged[u.user_id] = { at: new Date().toISOString(), skipped: reason };
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

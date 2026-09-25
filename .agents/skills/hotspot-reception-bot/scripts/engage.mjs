#!/usr/bin/env node
// HotSpot 接待机器人 —— 在线无人交互用户关怀(engage)
// 用法: node engage.mjs [--dry-run] [--force]
//       WELCOME_PERSONA=<名> 可锁定单人格(手动模式,轮转跳过);不设则自动轮转分配
// 口令: WELCOME_PASSWORD_<大写人格名>,未设回退 WELCOME_PASSWORD;只经 env 注入
// 「无人交互」判定:DB 模式为诚实保守口径(从未收到除人格账号外任何人的私聊,瞬间无他人的评论/反应);
//                   API 模式为降级口径(在线普通用户 ∉ 已联系登记簿 ∉ 本账号已有会话)。
// 连续性铁律:跟进(follow_up)只由原欢迎者发出(shared.contacted[uid].by / 各人格 state.greeted 归属);
//             从未被任何人格欢迎的在线用户按 shared.rrIndex 轮转分配新会话归属。
// 防打扰:关怀间隔 3 天;欢迎 24h 内不贴脸;跟进只做一次;跨人格共享联系登记簿。
import { execFileSync } from 'node:child_process';
import { pathToFileURL } from 'node:url';
import {
  loadPersona, listPersonas, pickRoundRobin,
  loadState, saveState, loadShared, saveShared, markContacted, activePersonaName,
} from './persona.mjs';
import {
  api, signin, sanitizeHandle, addressableNick, inWindow, bumpDaily, effectiveDailySent, dbOutboundCounts,
} from './greet.mjs';

const FORCE = process.argv.includes('--force');
const DRY = process.argv.includes('--dry-run');
const PERSONA = activePersonaName();
const P = loadPersona(PERSONA); // 默认人格(向后兼容:参数化函数的缺省值)
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

// DB 模式:在线且无人交互的用户(最新一条瞬间带一个标签用于话术)。
// 「人格账号」不视为交互来源:排除名单从 personas/*.json 的 userId 集合生成。
function lonelyUsersSql(excludeIds) {
  const notIn = excludeIds.map((id) => `'${id}'`).join(',');
  return `
WITH cand AS (
  SELECT u.user_id, u.nickname, u.created_at,
         (SELECT t.tag_name FROM moments m
          JOIN moment_tags t ON t.moment_id = m.moment_id
          WHERE m.user_id = u.user_id AND m.is_deleted = false
          ORDER BY m.created_at DESC LIMIT 1) AS last_tag,
         EXISTS (SELECT 1 FROM moments m WHERE m.user_id = u.user_id AND m.is_deleted = false) AS has_moments
  FROM users u
  WHERE u.is_deleted = false AND u.status = 'normal' AND u.user_id NOT IN (${notIn})
    AND u.presence_status = 'online' AND u.last_online > now() - interval '15 minutes'
)
SELECT c.user_id, c.nickname, c.last_tag, c.has_moments
FROM cand c
WHERE NOT EXISTS (
  SELECT 1 FROM private_chats pc
  JOIN private_messages pm ON pm.chat_key = pc.chat_key
  WHERE (pc.user_a_id = c.user_id OR pc.user_b_id = c.user_id)
    AND pm.sender_user_id <> c.user_id AND pm.sender_user_id NOT IN (${notIn})
) AND NOT EXISTS (
  SELECT 1 FROM moments m JOIN comments cm ON cm.moment_id = m.moment_id
  WHERE m.user_id = c.user_id AND m.is_deleted = false AND cm.is_deleted = false
    AND cm.user_id <> c.user_id AND cm.user_id NOT IN (${notIn})
) AND NOT EXISTS (
  SELECT 1 FROM moments m JOIN reactions r ON r.target_type = 'moment' AND r.target_id = m.moment_id
  WHERE m.user_id = c.user_id AND m.is_deleted = false
    AND r.user_id <> c.user_id AND r.user_id NOT IN (${notIn})
) AND NOT EXISTS (
  SELECT 1 FROM private_chats pc2 JOIN private_messages pm2 ON pm2.chat_key = pc2.chat_key
  WHERE (pc2.user_a_id = c.user_id OR pc2.user_b_id = c.user_id)
    AND pm2.sender_user_id = c.user_id
)
ORDER BY c.created_at ASC;
`;
}

function lonelyUsersDb(excludeIds) {
  return execFileSync('docker', [
    'exec', DB_CONTAINER, 'psql', '-U', 'postgres', '-d', 'hotspot', '-t', '-A', '-F', '|', '-c', lonelyUsersSql(excludeIds),
  ], { encoding: 'utf8' }).trim().split('\n').filter(Boolean).map((line) => {
    const [user_id, nickname, last_tag, has_moments] = line.split('|');
    return { user_id, nickname, last_tag: last_tag || null, has_moments: has_moments === 't' };
  });
}

// API 模式(线上降级口径):在线普通用户,排除全部人格账号
async function onlineUsersApi(personas) {
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
  const ids = personas.map((p) => p.userId);
  const now = Date.now();
  return (r.data?.items ?? [])
    .filter((it) => it.user.presence_status === 'online'
      && it.last_active_at && now - Date.parse(it.last_active_at) < 15 * 60 * 1000
      && !ids.includes(it.user.user_id))
    .map((it) => ({ user_id: it.user.user_id, nickname: it.user.nickname, last_tag: null, has_moments: false }));
}

// 本账号已有会话的对端(API 模式陌生开场前排除;避免对本就在聊天的人陌生开场)
async function myChatPeers(token) {
  const r = await api('/api/v1/private_chats/fetch', { pagination: null }, token);
  if (r.code !== 0) return new Set();
  return new Set((r.data?.items ?? []).map((c) => c.other_user?.user_id).filter(Boolean));
}

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
    console.log(`当前 ${new Date().toLocaleString('zh-CN')} 不在接待时段(23:00–次日09:00),跳过。`);
    console.log(JSON.stringify({ sent: 0, skipped: 0, failed: [], out_of_window: true }));
    return;
  }
  const shared = loadShared();
  const states = {};
  for (const p of active) states[p.name] = loadState(p.name);

  let cap = 10;
  const cfg = await api('/public/v1/system/client_config', {});
  if (cfg.code === 0 && cfg.data?.limits?.stranger_message_daily_quota) {
    cap = cfg.data.limits.stranger_message_daily_quota;
  }

  const now = Date.now();
  const raw = DATA_SOURCE === 'db' ? lonelyUsersDb(personas.map((p) => p.userId)) : await onlineUsersApi(personas);
  const outbound = DATA_SOURCE === 'db' ? dbOutboundCounts(active) : new Map();

  // 连续性铁律分组:跟进(follow_up)只由原欢迎者发出;从未被任何人格接触的
  // 在线用户(fresh)按轮转分配新会话归属。by 以 shared 登记簿为事实源,
  // state.greeted 作补充(登记簿缺失时的历史兜底)。
  const followups = new Map(); // personaName -> 在线用户[](该人格欢迎过、可跟进)
  const engageOf = new Map();  // personaName -> 在线用户[](轮转新分配)
  for (const u of raw) {
    const by = shared.contacted[u.user_id]?.by;
    const stateOwner = active.find((p) => states[p.name].greeted[u.user_id] && !states[p.name].greeted[u.user_id].skipped)?.name ?? null;
    const owner = (by && names.includes(by)) ? by : stateOwner;
    if (owner) {
      if (engageBlocked(states[owner], u.user_id, now)) continue;
      if (!followups.has(owner)) followups.set(owner, []);
      followups.get(owner).push(u);
    } else if (!shared.contacted[u.user_id]) {
      const name = locked ? names[0] : pickRoundRobin(shared, names);
      if (!engageOf.has(name)) engageOf.set(name, []);
      engageOf.get(name).push(u);
    }
    // 其余(shared 登记簿有记录但归属不是本池人格)→ 不陌生开场,跳过
  }

  const sessions = {};
  for (const p of active) {
    sessions[p.name] = await signin(states[p.name], p);
    console.log(`[${p.name}] 登录成功, user_id = ${sessions[p.name].user_id} (数据源: ${DATA_SOURCE})`);
  }

  let sent = 0, skipped = 0; const failed = [];
  const sentBy = {};
  for (const p of active) {
    const name = p.name;
    const st = states[name];
    const fu = followups.get(name) ?? [];
    const en = engageOf.get(name) ?? [];
    if (!fu.length && !en.length) continue;
    console.log(`[${name}] 待关怀:跟进 ${fu.length} 人 + 新分配 ${en.length} 人`);
    const peers = DATA_SOURCE === 'api' ? await myChatPeers(sessions[name].access_token) : null;
    let perCount = 0;
    const queue = [...fu.map((u) => ({ u, follow: true })), ...en.map((u) => ({ u, follow: false }))];
    for (const { u, follow } of queue) {
      if (effectiveDailySent(st, shared, p) >= cap) {
        console.log(`[${name}] 今日陌生人配额(${cap} 条)已用完,剩余留到下个窗口`);
        break;
      }
      if (follow && DATA_SOURCE === 'db' && (outbound.get(name)?.get(u.user_id) ?? 0) >= 2) continue; // 出站兜底:已发多条收手
      if (!follow && peers && peers.has(u.user_id)) continue; // API 模式:本账号已有会话不做陌生开场
      const isFollowUp = follow; // 欢迎过没回应 → 轻声跟进;轮转新分配 → 顺带完成欢迎/关怀
      const safeTag = addressableNick(u.last_tag ?? '', 20); // 防投毒:用户可控标签净化后才进话术
      const lines = isFollowUp ? p.lines.followUp
        : (u.has_moments ? p.lines.engageNewWithMoments : p.lines.engageNew);
      const content = renderLine(lines[perCount % lines.length], sanitizeHandle(u.nickname), safeTag);
      if (DRY) { console.log('[dry-run]', name, '->', u.user_id, u.nickname, '|', content.slice(0, 24) + '…'); sent++; perCount++; sentBy[name] = perCount; continue; }
      const r = await api('/api/v1/private_messages/send', {
        target_user_id: u.user_id, type: 'text', content,
        file_id: null, share_code: null, reply_to_message_id: null,
      }, sessions[name].access_token, crypto.randomUUID());
      if (r.code === 0) {
        sent++; perCount++; sentBy[name] = perCount;
        bumpDaily(st);
        st.engaged[u.user_id] = { at: new Date().toISOString(), message_id: r.data?.message_id };
        markContacted(shared, u.user_id, name, { message_id: r.data?.message_id, kind: isFollowUp ? 'follow_up' : 'engage' });
        if (isFollowUp) st.greeted[u.user_id].followed_up = true; // 跟进只做一次
        else if (!st.greeted[u.user_id]) st.greeted[u.user_id] = { at: new Date().toISOString(), via: 'engage' }; // 顺带欢迎
        console.log(`[${name}] 已关怀`, u.user_id, u.nickname);
      } else if (r.code === 10200) {
        const reason = r.data?.forbidden_reason || 'unknown';
        if (reason === 'stranger_quota_exceeded') {
          console.log(`[${name}] 陌生人配额耗尽,${u.user_id} 留待配额恢复`);
          continue;
        }
        skipped++;
        st.engaged[u.user_id] = { at: new Date().toISOString(), skipped: reason };
        console.log(`[${name}] 跳过`, u.user_id, reason);
      } else {
        failed.push({ user_id: u.user_id, code: r.code, message: r.message, by: name });
        console.error('失败', u.user_id, JSON.stringify(r));
      }
      saveState(st, name);
      saveShared(shared);
    }
  }
  console.log(JSON.stringify({ sent, skipped, failed, by_persona: sentBy }));
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  // 顶层异常包装:崩溃栈对值班无信息量,输出干净一行错误(下轮自动重试)
  main().catch((e) => { console.error('执行异常:', e?.message || e); process.exit(3); });
}

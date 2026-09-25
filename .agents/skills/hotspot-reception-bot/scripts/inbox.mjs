#!/usr/bin/env node
// 收件箱:列出「球在对方手里」的会话(未读>0)+ 最近聊天历史,供窗口内逐条人工语境回复
// 用法: WELCOME_PASSWORD=<人格口令> node inbox.mjs [--persona <名>]
// 纯 API 实现(private_chats/fetch + private_messages/fetch),本地与线上通用。
// 回复前先读 persona.json 的 card 人物卡,保持人设一致。
import { pathToFileURL } from 'node:url';
import { loadPersona, activePersonaName } from './persona.mjs';
import { api, signin } from './greet.mjs';

const BASE = process.env.WELCOME_BASE_URL || 'http://localhost:8888';
const P = loadPersona(activePersonaName());

async function pendingInbox(token) {
  const chats = await api('/api/v1/private_chats/fetch', { pagination: null }, token);
  if (chats.code !== 0) throw new Error('private_chats/fetch 失败: ' + JSON.stringify(chats));
  const out = [];
  for (const c of chats.data?.items ?? []) {
    if (!(c.unread_count > 0)) continue;
    const peer = c.other_user?.user_id;
    if (!peer) continue;
    const hist = await api('/api/v1/private_messages/fetch', {
      target_user_id: peer, pagination: { limit: 8 },
    }, token);
    const items = hist.code === 0 ? (hist.data?.items ?? []).slice().reverse() : [];
    out.push({
      user_id: peer, nickname: c.other_user?.nickname ?? peer,
      history: items.map((m) => ({ who: m.sender?.user_id === P.userId ? P.nickname : (m.sender?.nickname ?? '?'), content: m.content ?? '', sent_at: m.sent_at })),
    });
  }
  return out;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const P2 = P;
  if (!P2.password) { console.error('缺少 WELCOME_PASSWORD 环境变量'); process.exit(1); }
  const me = await signin();
  const items = await pendingInbox(me.access_token);
  console.log(`[${activePersonaName()}] 待回复会话 ${items.length} 个`);
  for (const c of items) {
    console.log(`\n== ${c.nickname} (${c.user_id})`);
    for (const m of c.history) console.log(`  [${m.sent_at.slice(5, 16)}] ${m.who}: ${m.content}`);
  }
  if (!items.length) console.log('全部会话均已回复 ✓');
}

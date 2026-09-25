#!/usr/bin/env node
// 收件箱:遍历全部人格账号,聚合「球在对方手里」的会话(未读>0)+ 最近聊天历史,
// 供窗口内逐条人工语境回复。WELCOME_PERSONA=<名> 可只看单人格。
// 用法: node inbox.mjs
// 顺序 signin 各人格(间隔 2s 防限流);每节标明所属人格与人物卡路径(回复时读对应卡)。
// 纯 API 实现(private_chats/fetch + private_messages/fetch),本地与线上通用。
import { pathToFileURL } from 'node:url';
import { listPersonas, loadState, saveState } from './persona.mjs';
import { api, signin } from './greet.mjs';

async function pendingInbox(token, persona) {
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
      history: items.map((m) => ({ who: m.sender?.user_id === persona.userId ? persona.nickname : (m.sender?.nickname ?? '?'), content: m.content ?? '', sent_at: m.sent_at })),
    });
  }
  return out;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  // 顶层异常包装:崩溃栈对值班无信息量,输出干净一行错误(下轮自动重试)
  (async () => {
    const personas = listPersonas();
    const locked = process.env.WELCOME_PERSONA || '';
    const pool = locked ? personas.filter((p) => p.name === locked) : personas;
    if (locked && !pool.length) {
      console.error(`WELCOME_PERSONA=${locked} 不在人格池中(可用: ${personas.map((p) => p.name).join(', ')})`);
      process.exit(1);
    }
    let any = false;
    for (let i = 0; i < pool.length; i++) {
      const p = pool[i];
      if (!p.password) { console.log(`[${p.nickname}](${p.name}) 未设置口令(WELCOME_PASSWORD_${p.name.toUpperCase()}),跳过`); continue; }
      any = true;
      if (i > 0) await new Promise((res) => setTimeout(res, 2000)); // 人格间间隔 2s 防鉴权限流
      try {
        const st = loadState(p.name);
        const me = await signin(st, p);
        saveState(st, p.name);
        const items = await pendingInbox(me.access_token, p);
        console.log(`\n[${p.nickname}](${p.name}) 待回复会话 ${items.length} 个 | 人物卡: ${p.card}`);
        for (const c of items) {
          console.log(`\n== ${c.nickname} (${c.user_id})`);
          for (const m of c.history) console.log(`  [${m.sent_at.slice(5, 16)}] ${m.who}: ${m.content}`);
        }
        if (!items.length) console.log('全部会话均已回复 ✓');
      } catch (e) {
        // 单人格失败(限流/口令缺失/网络)不阻断其他人格聚合
        console.error(`[${p.name}] 收件箱获取失败: ${e?.message || e}`);
      }
    }
    if (!any) { console.error('没有任何已设置口令的人格'); process.exit(1); }
  })().catch((e) => { console.error('执行异常:', e?.message || e); process.exit(3); });
}

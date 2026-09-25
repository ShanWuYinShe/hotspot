#!/usr/bin/env node
// 单条回复工具:node reply.mjs --to <user_id> --text "<内容>" | --text-file <路径>
// 人格由 --persona/WELCOME_PERSONA 决定;回复计入当日配额,并写共享联系登记簿、标记会话已读。
import { pathToFileURL } from 'node:url';
import { readFileSync } from 'node:fs';
import { loadPersona, loadState, saveState, loadShared, saveShared, markContacted, activePersonaName } from './persona.mjs';
import { api, signin, inWindow, effectiveDailySent, bumpDaily } from './greet.mjs';

function arg(name) {
  const i = process.argv.indexOf(name);
  return i >= 0 ? process.argv[i + 1] : null;
}

async function main() {
  const P = loadPersona();
  if (!P.password) { console.error('缺少 WELCOME_PASSWORD 环境变量'); process.exit(1); }
  const to = arg('--to');
  const textFile = arg('--text-file');
  const text = textFile ? readFileSync(textFile, 'utf8').trim() : arg('--text');
  if (!to || !text) { console.error('用法: node reply.mjs --to <user_id> --text "<内容>" | --text-file <路径>'); process.exit(1); }
  // --force 是纯布尔 flag,不能用 arg()(取 flag 的下一个 token):放在命令末尾时
  // 取到 undefined 被当成「未加 --force」,仍被时段拦截;与 greet/engage 一致用 includes。
  if (!process.argv.includes('--force') && !inWindow()) {
    console.log('不在接待时段(23:00–次日09:00),加 --force 可强制回复。');
    process.exit(0);
  }
  const state = loadState();
  const shared = loadShared();
  const me = await signin(state);
  let cap = 10;
  const cfg = await api('/public/v1/system/client_config', {});
  if (cfg.code === 0 && cfg.data?.limits?.stranger_message_daily_quota) {
    cap = cfg.data.limits.stranger_message_daily_quota;
  }
  if (effectiveDailySent(state, shared) >= cap) {
    console.error(`今日陌生人配额(${cap} 条)已用完,明天再回或手工调整 state`);
    process.exit(3);
  }
  const r = await api('/api/v1/private_messages/send', {
    target_user_id: to, type: 'text', content: text,
    file_id: null, share_code: null, reply_to_message_id: null,
  }, me.access_token, crypto.randomUUID());
  if (r.code !== 0) { console.error('发送失败:', JSON.stringify(r)); process.exit(2); }
  bumpDaily(state);
  markContacted(shared, to, activePersonaName(), { message_id: r.data?.message_id, kind: 'reply' });
  if (!state.engaged[to]) state.engaged[to] = { at: new Date().toISOString(), via: 'reply' };
  // 回复后把会话标为已读,收件箱不再重复提示
  await api('/api/v1/private_messages/mark_read', { scope: 'session', target_user_id: to }, me.access_token).catch(() => {});
  saveState(state);
  saveShared(shared);
  console.log(`[${activePersonaName()}] 已回复`, to, '| message_id =', r.data?.message_id);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  // 顶层异常包装:崩溃栈对值班无信息量,输出干净一行错误(下轮自动重试)
  main().catch((e) => { console.error('执行异常:', e?.message || e); process.exit(3); });
}

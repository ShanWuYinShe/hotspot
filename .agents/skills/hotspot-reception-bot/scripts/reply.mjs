#!/usr/bin/env node
// 单条回复工具:node reply.mjs --to <user_id> --text "<内容>" | --text-file <路径> [--persona <名>]
// 人格解析优先级:--persona 显式指定 > WELCOME_PERSONA 锁定 > shared.contacted[to].by
// (连续性铁律,谁欢迎归谁)> 从未联系过 → 轮转分配并当场登记(by 非法如 system/admin
// → 默认林小满)。同一用户可能与多个人格各有会话,by 只是最早接触者——球在哪个
// 人格的会话里,就用 inbox 输出的归属 --persona 指定,否则会错发到 by 人格。
// 回复计入该人格当日配额,并写共享联系登记簿、标记会话已读。
import { pathToFileURL } from 'node:url';
import { readFileSync } from 'node:fs';
import {
  loadPersona, listPersonas, resolvePersonaFor,
  loadState, saveState, loadShared, saveShared, markContacted, activePersonaName,
} from './persona.mjs';
import { api, apiAuthed, signin, inWindow, effectiveDailySent, bumpDaily } from './greet.mjs';

function arg(name) {
  const i = process.argv.indexOf(name);
  return i >= 0 ? process.argv[i + 1] : null;
}

async function main() {
  const to = arg('--to');
  const personaArg = arg('--persona'); // 显式指定:最高优先(定位多会话归属,见文件头)
  const textFile = arg('--text-file');
  const text = textFile ? readFileSync(textFile, 'utf8').trim() : arg('--text');
  if (!to || !text) { console.error('用法: node reply.mjs --to <user_id> --text "<内容>" | --text-file <路径> [--persona <名>]'); process.exit(1); }
  const shared = loadShared();
  const personas = listPersonas();
  const names = personas.map((p) => p.name);
  const locked = process.env.WELCOME_PERSONA || '';
  if (locked && !names.includes(locked)) {
    console.error(`WELCOME_PERSONA=${locked} 不在人格池中(可用: ${names.join(', ')})`);
    process.exit(1);
  }
  if (personaArg && !names.includes(personaArg)) {
    console.error(`人格 ${personaArg} 不在人格池中(可用: ${names.join(', ')})`);
    process.exit(1);
  }
  // --force 是纯布尔 flag,不能用 arg()(取 flag 的下一个 token):放在命令末尾时
  // 取到 undefined 被当成「未加 --force」,仍被时段拦截;与 greet/engage 一致用 includes。
  if (!process.argv.includes('--force') && !inWindow()) {
    console.log('不在接待时段(23:00–次日09:00),加 --force 可强制回复。');
    process.exit(0);
  }
  // 人格归属:显式 --persona / 锁定优先;否则按登记簿归属(新用户轮转分配并登记)
  const name = resolvePersonaFor(shared, to, names, { explicit: personaArg || locked });
  const P = loadPersona(name);
  if (!P.password) {
    console.error(`缺少 ${name} 的口令(WELCOME_PASSWORD_${name.toUpperCase()})`);
    process.exit(1);
  }
  const state = loadState(name);
  await signin(state, P); // 保证已登录(state.auth 缓存命中则复用)
  let cap = 10;
  const cfg = await api('/public/v1/system/client_config', {});
  if (cfg.code === 0 && cfg.data?.limits?.stranger_message_daily_quota) {
    cap = cfg.data.limits.stranger_message_daily_quota;
  }
  if (effectiveDailySent(state, shared, P) >= cap) {
    console.error(`[${name}] 今日陌生人配额(${cap} 条)已用完,明天再回或手工调整 state`);
    process.exit(3);
  }
  const r = await apiAuthed(state, P, '/api/v1/private_messages/send', {
    target_user_id: to, type: 'text', content: text,
    file_id: null, share_code: null, reply_to_message_id: null,
  }, state.auth?.token, crypto.randomUUID());
  if (r.code !== 0) { console.error('发送失败:', JSON.stringify(r)); process.exit(2); }
  bumpDaily(state);
  // 登记簿刷新:by 保持归属人格不变(连续性铁律),时间戳/kind 更新为本次回复
  markContacted(shared, to, name, { message_id: r.data?.message_id, kind: 'reply' });
  if (!state.engaged[to]) state.engaged[to] = { at: new Date().toISOString(), via: 'reply' };
  // 回复后把会话标为已读,收件箱不再重复提示
  await apiAuthed(state, P, '/api/v1/private_messages/mark_read', { scope: 'session', target_user_id: to }, state.auth?.token).catch(() => {});
  saveState(state, name);
  saveShared(shared);
  console.log(`[${P.nickname}](${name}) 已回复`, to, '| message_id =', r.data?.message_id);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  // 顶层异常包装:崩溃栈对值班无信息量,输出干净一行错误(下轮自动重试)
  main().catch((e) => { console.error('执行异常:', e?.message || e); process.exit(3); });
}

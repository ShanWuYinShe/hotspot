#!/usr/bin/env node
// 在线保活:为全部人格维持 WebSocket 长连接(在线状态由 WS Auth 驱动,spec api_spec_websocket.md)
// 用法: node ws-online.mjs [--force]  (--force = 窗口外也在线,调试用)
// 行为:窗口(23:00–次日09:00)内每人格保持一条 WS(独立 wsDeviceId,避免顶掉脚本会话);
//       窗口外主动断开(人设即「睡觉」);断线自动重连(4001 先刷新 token;1011 指数退避)。
// 口令纪律:仅 env 注入,不落盘。日志:state/ws-online.log
import { appendFileSync, writeFileSync, existsSync, readFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  loadPersona, listPersonas, loadState, saveState, activePersonaName,
} from './persona.mjs';
import { api, signin, inWindow } from './greet.mjs';

const HERE = dirname(fileURLToPath(import.meta.url));
const STATE_DIR = join(dirname(HERE), 'state');
const FORCE = process.argv.includes('--force');
const ALWAYS_ONLINE = process.env.WELCOME_ALWAYS_ONLINE === '1';
const BASE = process.env.WELCOME_BASE_URL || 'http://localhost:8888';
const WS_URL = (process.env.WELCOME_WS_URL || BASE.replace(/^http/, 'ws')) + '/ws/v1';
const LOG = join(STATE_DIR, 'ws-online.log');
const CHECK_INTERVAL_MS = 30 * 1000;

if (!existsSync(STATE_DIR)) mkdirSync(STATE_DIR, { recursive: true });
function log(msg) {
  const line = `[${new Date().toISOString()}] ${msg}`;
  try { appendFileSync(LOG, line + '\n'); } catch {}
}

// 每人格独立的 ws 专属设备号(与脚本 device 分离,避免单设备单会话互顶)
function wsDeviceId(name) {
  const f = join(STATE_DIR, `ws-device.${name}.txt`);
  if (existsSync(f)) return readFileSync(f, 'utf8').trim();
  const id = crypto.randomUUID();
  writeFileSync(f, id);
  return id;
}
// ws 专属 token 缓存(与脚本 state.auth 分离,互不覆盖)
function wsAuthFile(name) { return join(STATE_DIR, `ws-auth.${name}.json`); }
function loadWsAuth(name) {
  const f = wsAuthFile(name);
  return existsSync(f) ? JSON.parse(readFileSync(f, 'utf8')) : {};
}
function saveWsAuth(name, auth) { writeFileSync(wsAuthFile(name), JSON.stringify(auth, null, 2)); }

async function personaToken(p) {
  const f = wsAuthFile(p.name);
  const cached = loadWsAuth(p.name);
  if (cached.token && cached.expiresAt && Date.parse(cached.expiresAt) - Date.now() > 120 * 1000) {
    return cached.token;
  }
  const stateStub = {};
  const me = await signin(stateStub, p, wsDeviceId(p.name));
  saveWsAuth(p.name, { token: me.access_token, expiresAt: me.access_token_expires_at, userId: me.user_id });
  return me.access_token;
}

// 单个人格的连接状态
const conns = new Map(); // name -> { ws, pingTimer, pongTimer, resumeToken, backoffMs }

function setStatus(name, online) {
  const f = join(STATE_DIR, 'ws-status.json');
  let s = {};
  try { s = existsSync(f) ? JSON.parse(readFileSync(f, 'utf8')) : {}; } catch {}
  s[name] = { online, at: new Date().toISOString() };
  writeFileSync(f, JSON.stringify(s, null, 2));
}

function connectPersona(p) {
  const name = p.name;
  const auth = loadWsAuth(name);
  const ws = new WebSocket(WS_URL);
  const c = { ws, pingTimer: null, pongTimer: null, resumeToken: auth.resumeToken || null, closing: false };
  conns.set(name, c);

  ws.addEventListener('open', () => {
    // 连接后 5s 内必须 Auth(app 客户端携带 token)
    ws.send(JSON.stringify({ type: 'Auth', payload: { client_type: 'app', access_token: auth.token || null, device_id: wsDeviceId(name), resume_token: c.resumeToken } }));
  });
  ws.addEventListener('message', (raw) => {
    let ev; try { ev = JSON.parse(String(raw.data)); } catch { return; }
    if (ev.type === 'ConnectionEstablished') {
      c.resumeToken = ev.payload.resume_token;
      auth.resumeToken = c.resumeToken; saveWsAuth(name, auth);
      const interval = (ev.payload.config?.ping_interval ?? 30) * 1000;
      console.log('PING-DEBUG: interval_ms =', interval, '| config =', JSON.stringify(ev.payload.config));
      clearInterval(c.pingTimer);
      c.pingTimer = setInterval(() => { try { const m = JSON.stringify({ type: 'Ping', payload: {} }); ws.send(m); console.log('PING-SENT @', new Date().toISOString()); } catch (e) { console.log('PING-ERROR:', e?.message || e); } }, interval);
      setStatus(name, true);
      log(`[${name}] WS 在线 (connection=${ev.payload.connection_id})`);
    } else if (ev.type === 'Pong') {
      clearTimeout(c.pongTimer);
    } else if (ev.type === 'SessionRevoked') {
      log(`[${name}] 会话被吊销(${ev.payload.reason}),刷新 token 重连`);
      try { ws.close(); } catch {}
    } else if (ev.type === 'Error') {
      log(`[${name}] WS Error: ${JSON.stringify(ev.payload)}`);
    }
  });
  ws.addEventListener('close', (code) => {
    clearInterval(c.pingTimer); clearTimeout(c.pongTimer);
    conns.delete(name);
    setStatus(name, false);
    if (c.closing) return;
    log(`[${name}] WS 断开(code=${code}),5s 后重连`);
    setTimeout(() => { if (inWindow() || ALWAYS_ONLINE || FORCE) connectPersona(p); }, 5000);
  });
  ws.addEventListener('error', () => { /* close 事件统一处理 */ });
}

function disconnectPersona(name) {
  const c = conns.get(name);
  if (!c) return;
  c.closing = true;
  clearInterval(c.pingTimer); clearTimeout(c.pongTimer);
  try { c.ws.close(1000); } catch {}
  conns.delete(name);
  setStatus(name, false);
  log(`[${name}] 窗口外,主动断开(睡觉)`);
}

async function main() {
  const personas = listPersonas();
  for (const p of personas) log(`[${p.name}] 在线保活启动(共 ${personas.length} 人格)`);
  for (;;) {
    const windowOpen = inWindow() || ALWAYS_ONLINE || FORCE;
    for (const p of personas) {
      const connected = conns.has(p.name);
      if (windowOpen && !connected) {
        try {
          await personaToken(p); // 预取/刷新 token(10500 时 signin 自带退避)
          connectPersona(p);
        } catch (e) {
          log(`[${p.name}] 连接失败: ${e?.message || e}`);
        }
      } else if (!windowOpen && connected) {
        disconnectPersona(p.name);
      }
    }
    await new Promise((res) => setTimeout(res, CHECK_INTERVAL_MS));
  }
}

main().catch((e) => { console.error('执行异常:', e?.message || e); process.exit(3); });
#!/usr/bin/env node
// 在线保活:为全部人格维持 WebSocket 长连接(在线状态由 WS Auth 驱动,spec api_spec_websocket.md)
// 用法: node ws-online.mjs [--force]  (--force = 窗口外也在线,调试用)
// 行为:窗口(23:00–次日09:00)内每人格保持一条 WS(独立 wsDeviceId,避免顶掉脚本会话);
//       窗口外主动断开(人设即「睡觉」);断线自动重连(吊销/4001 先刷 token,1011 指数退避)。
// 单实例守卫:心跳锁(主循环每 30s 触碰 state/ws-online.lock 的 mtime),启动见 90s 内
//       的新鲜锁直接退出——多实例并存会以同一 wsDeviceId 互踢形成断开/重连风暴
//       (2026-09-28 修复;此前反复启动曾累积 11 实例并存)。SIGTERM/SIGINT 会清锁,可立即重启。
// 口令纪律:仅 env 注入,不落盘。日志:state/ws-online.log
import { appendFileSync, writeFileSync, existsSync, readFileSync, statSync, unlinkSync, mkdirSync } from 'node:fs';
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
const PID_FILE = join(STATE_DIR, 'ws-online.pid');
const CHECK_INTERVAL_MS = 30 * 1000;

if (!existsSync(STATE_DIR)) mkdirSync(STATE_DIR, { recursive: true });
function log(msg) {
  const line = "[" + new Date().toISOString() + "] " + msg + "\n";
  try { appendFileSync(LOG, line); } catch {}
}

// —— 单实例守卫(心跳锁)——
// state/ws-online.lock 由主循环每 CHECK_INTERVAL_MS 触碰一次 mtime;启动时见到
// LOCK_FRESH_MS 内的新鲜锁即视为已有实例,直接退出;心跳已停(进程崩溃)则接管。
// 不用 ps:沙盒内 ps 被禁(Operation not permitted),且锁文件天然免疫 pid 复用。
const LOCK_FILE = join(STATE_DIR, 'ws-online.lock');
const LOCK_FRESH_MS = 3 * CHECK_INTERVAL_MS;
function acquireSingleton() {
  if (existsSync(LOCK_FILE)) {
    let old = 0, ageMs = Number.POSITIVE_INFINITY;
    try { old = Number.parseInt(readFileSync(LOCK_FILE, 'utf8').trim(), 10) || 0; } catch {}
    try { ageMs = Date.now() - statSync(LOCK_FILE).mtimeMs; } catch {}
    if (ageMs < LOCK_FRESH_MS) {
      log("已有实例在运行(pid=" + (old || "?") + ",心跳 " + Math.round(ageMs / 1000) + "s 前),本次启动退出");
      console.error("ws-online: 已有实例在运行(pid=" + (old || "?") + "),退出");
      process.exit(0);
    }
    log("接管陈旧锁(原 pid=" + (old || "?") + ",心跳已停 " + Math.round(ageMs / 1000) + "s)");
  }
  writeFileSync(LOCK_FILE, String(process.pid));
  writeFileSync(PID_FILE, String(process.pid)); // 供 kill $(cat ...) 停止用
}

// 每人格独立的 ws 专属设备号(与脚本 device 分离,避免单设备单会话互顶)
function wsDeviceId(name) {
  const f = join(STATE_DIR, "ws-device." + name + ".txt");
  if (existsSync(f)) return readFileSync(f, 'utf8').trim();
  const id = crypto.randomUUID();
  writeFileSync(f, id);
  return id;
}
// ws 专属 token 缓存(与脚本 state.auth 分离,互不覆盖)
function wsAuthFile(name) { return join(STATE_DIR, "ws-auth." + name + ".json"); }
function loadWsAuth(name) {
  const f = wsAuthFile(name);
  return existsSync(f) ? JSON.parse(readFileSync(f, 'utf8')) : {};
}
function saveWsAuth(name, auth) { writeFileSync(wsAuthFile(name), JSON.stringify(auth, null, 2)); }
// 吊销/鉴权失败后剥除缓存凭据,让下一次连接重新 signin
function invalidateWsAuth(name) {
  const a = loadWsAuth(name);
  if (a.token === undefined && a.expiresAt === undefined && a.resumeToken === undefined) return;
  delete a.token; delete a.expiresAt; delete a.resumeToken;
  saveWsAuth(name, a);
}

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
const conns = new Map(); // name -> { ws, pingTimer, pongTimer, resumeToken, closing }

function setStatus(name, online) {
  const f = join(STATE_DIR, 'ws-status.json');
  let s = {};
  try { s = existsSync(f) ? JSON.parse(readFileSync(f, 'utf8')) : {}; } catch {}
  s[name] = { online, at: new Date().toISOString() };
  writeFileSync(f, JSON.stringify(s, null, 2));
}

function connectPersona(p) {
  const name = p.name;
  // 防御:该人格已有连接(含半开旧连接)时先关掉,避免同 device 并行连接互踢
  const prev = conns.get(name);
  if (prev) {
    prev.closing = true;
    clearInterval(prev.pingTimer); clearTimeout(prev.pongTimer);
    try { prev.ws.close(); } catch {}
  }
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
      clearInterval(c.pingTimer); clearTimeout(c.pongTimer);
      c.pingTimer = setInterval(() => {
        try {
          ws.send(JSON.stringify({ type: 'Ping', payload: {} }));
          // 僵尸连接检测:一个 ping 周期(+10s 宽限)内无 Pong 则主动断开走重连
          clearTimeout(c.pongTimer);
          c.pongTimer = setTimeout(() => {
            log("[" + name + "] Ping 超时无 Pong,主动断开重连");
            try { ws.close(); } catch {}
          }, interval + 10 * 1000);
        } catch (e) { log("[" + p.name + "] Ping 发送异常: " + (e?.message || e)); }
      }, interval);
      setStatus(name, true);
      log("[" + name + "] WS 在线 (connection=" + ev.payload.connection_id + ")");
    } else if (ev.type === 'Pong') {
      clearTimeout(c.pongTimer);
    } else if (ev.type === 'SessionRevoked') {
      log("[" + name + "] 会话被吊销(" + ev.payload.reason + "),刷新 token 重连");
      invalidateWsAuth(name);
      try { ws.close(); } catch {}
    } else if (ev.type === 'Error') {
      log("[" + name + "] WS Error: " + JSON.stringify(ev.payload));
      // 鉴权类失败:剥除缓存凭据,否则 5s 快速重连会一直复用坏 token(2026-09-27 20000 风暴)
      if (ev.payload?.code === 20000 || ev.payload?.code === 4001) { invalidateWsAuth(name); try { ws.close(); } catch {} }
    }
  });
  ws.addEventListener('close', (ev) => {
    clearInterval(c.pingTimer); clearTimeout(c.pongTimer);
    if (conns.get(name) !== c) return; // 孤儿旧连接:不动当前连接状态、不触发重连
    conns.delete(name);
    setStatus(name, false);
    if (c.closing) return;
    const code = ev && typeof ev === 'object' ? (ev.code ?? '?') : ev;
    log("[" + name + "] WS 断开(code=" + code + "),5s 后重连");
    setTimeout(() => {
      if (!(inWindow() || ALWAYS_ONLINE || FORCE)) return;
      if (conns.has(name)) return;
      // 先刷 token 再重连:过期/吊销场景复用旧 token 会连环失败
      personaToken(p)
        .catch((e) => log("[" + name + "] 重连前刷 token 失败,改由主循环重试: " + (e?.message || e)))
        .then(() => { if (!conns.has(name)) connectPersona(p); });
    }, 5000);
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
  log("[" + name + "] 窗口外,主动断开(睡觉)");
}

async function main() {
  acquireSingleton();
  const personas = listPersonas();
  for (const p of personas) log("[" + p.name + "] 在线保活启动(pid=" + process.pid + ",共 " + personas.length + " 人格)");
  for (;;) {
    try { writeFileSync(LOCK_FILE, String(process.pid)); } catch {} // 心跳:刷新锁 mtime
    const windowOpen = inWindow() || ALWAYS_ONLINE || FORCE;
    for (const p of personas) {
      const connected = conns.has(p.name);
      if (windowOpen && !connected) {
        try {
          await personaToken(p); // 预取/刷新 token(10500 时 signin 自带退避)
          connectPersona(p);
        } catch (e) {
          log("[" + p.name + "] 连接失败: " + (e?.message || e));
        }
      } else if (!windowOpen && connected) {
        disconnectPersona(p.name);
      }
    }
    await new Promise((res) => setTimeout(res, CHECK_INTERVAL_MS));
  }
}

main().catch((e) => { console.error('执行异常:', e?.message || e); process.exit(3); });
for (const sig of ['SIGTERM', 'SIGINT']) process.on(sig, () => { try { unlinkSync(LOCK_FILE); } catch {} process.exit(0); });
process.on('uncaughtException', (e) => log('未捕获异常(进程保持存活): ' + (e?.message || e)));
process.on('unhandledRejection', (e) => log('未处理的Promise拒绝(保持存活): ' + (e?.message || e)));
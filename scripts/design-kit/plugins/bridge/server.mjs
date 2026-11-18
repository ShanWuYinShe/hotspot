// HotSpot Figma Bridge —— 本地 server（bun）
// ws://127.0.0.1:8787 给插件连；HTTP 端点：
//   POST /eval {code}                     → 插件沙盒执行任意 Plugin API 代码
//   POST /save-version {title,description} → 打版本历史（插件内置命令）
//   GET  /status                           → 连接态
import { writeFileSync } from 'node:fs';
const PENDING = new Map();
let pluginWs = null;
let seq = 0;

function forward(payload, timeoutMs) {
  if (!pluginWs || pluginWs.readyState !== 1) {
    return Promise.resolve({ ok: false, error: 'plugin not connected' });
  }
  const id = 'e' + (++seq);
  return new Promise((resolve) => {
    PENDING.set(id, resolve);
    setTimeout(() => {
      if (PENDING.has(id)) { PENDING.delete(id); resolve({ ok: false, error: 'timeout' }); }
    }, timeoutMs || 60000);
  });
  pluginWs.send(JSON.stringify({ ...payload, id }));
}

Bun.serve({
  port: 8787,
  async fetch(req, server) {
    const url = new URL(req.url);
    if (req.method === 'POST' && (url.pathname === '/eval' || url.pathname === '/save-version')) {
      const body = await req.json().catch(() => ({}));
      const payload = url.pathname === '/eval'
        ? { type: 'eval', code: body.code }
        : { type: 'save-version', title: body.title, description: body.description };
      const result = await forward(payload, body.timeoutMs);
      return Response.json(result);
    }
    if (url.pathname === '/status') {
      return Response.json({ pluginConnected: !!(pluginWs && pluginWs.readyState === 1) });
    }
    if (server.upgrade(req)) return;
    return new Response('hotspot figma bridge');
  },
  websocket: {
    open(ws) { pluginWs = ws; },
    message(ws, message) {
      let data; try { data = JSON.parse(message.toString()); } catch { return; }
      if (data.type === 'result' && PENDING.has(data.id)) {
        const r = PENDING.get(data.id); PENDING.delete(data.id);
        r(data);
      }
    },
    close(ws) { if (pluginWs === ws) pluginWs = null; }
  }
});
writeFileSync(new URL('./server.pid', import.meta.url), String(process.pid));
console.log('bridge listening on http://127.0.0.1:8787');

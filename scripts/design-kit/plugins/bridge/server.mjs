// HotSpot Figma Bridge —— 本地 server（bun）
// ws://127.0.0.1:8787 给插件连；HTTP POST /eval {code} 转发执行并回传结果；GET /status 查连接态。
import { writeFileSync } from 'node:fs';
const PENDING = new Map();
let pluginWs = null;
let seq = 0;

Bun.serve({
  port: 8787,
  async fetch(req, server) {
    const url = new URL(req.url);
    if (url.pathname === '/eval' && req.method === 'POST') {
      const body = await req.json().catch(() => ({}));
      if (!pluginWs || pluginWs.readyState !== 1) {
        return Response.json({ ok: false, error: 'plugin not connected' }, { status: 503 });
      }
      const id = 'e' + (++seq);
      const p = new Promise((resolve) => {
        PENDING.set(id, resolve);
        setTimeout(() => {
          if (PENDING.has(id)) { PENDING.delete(id); resolve({ ok: false, error: 'timeout' }); }
        }, body.timeoutMs || 60000);
      });
      pluginWs.send(JSON.stringify({ type: 'eval', id, code: body.code }));
      const result = await p;
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

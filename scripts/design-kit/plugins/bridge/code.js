// HotSpot Figma Bridge —— 桌面插件内 WebSocket 客户端
// 与本地 server (ws://127.0.0.1:8787) 保持连接，接收 {type:'eval', id, code} 并在插件沙盒执行，
// 回传 {type:'result', id, ok, result|error}。断线自动重连。窗口保持开启即保持桥接。
var WS_URL = 'ws://127.0.0.1:8787';
var ws = null;

function send(msg) {
  try { ws.send(JSON.stringify(msg)); } catch (e) {}
}

function safe(v) {
  try {
    var s = JSON.stringify(v, function (k, val) {
      if (val && typeof val === 'object' && val.type && val.id) return '[node:' + val.type + ':' + val.id + ']';
      return val;
    });
    if (s === undefined) return String(v);
    if (s.length > 200000) return s.slice(0, 200000) + '...(truncated)';
    return JSON.parse(s);
  } catch (e) { return String(v); }
}

function handle(data) {
  if (!data || data.type !== 'eval' || !data.id) return;
  try {
    var fn = new Function('return (async () => {' + data.code + '})()');
    Promise.resolve(fn()).then(function (v) {
      send({ type: 'result', id: data.id, ok: true, result: safe(v) });
    }, function (e) {
      send({ type: 'result', id: data.id, ok: false, error: String(e && e.message || e).slice(0, 800) });
    });
  } catch (e) {
    send({ type: 'result', id: data.id, ok: false, error: String(e && e.message || e).slice(0, 800) });
  }
}

function connect() {
  try {
    ws = new WebSocket(WS_URL);
  } catch (e) { setTimeout(connect, 3000); return; }
  ws.onopen = function () { send({ type: 'hello', plugin: 'hotspot-bridge', file: figma.root.name, at: Date.now() }); };
  ws.onmessage = function (ev) {
    var data;
    try { data = JSON.parse(ev.data); } catch (e) { return; }
    handle(data);
  };
  ws.onclose = function () { setTimeout(connect, 2000); };
  ws.onerror = function () { try { ws.close(); } catch (e) {} };
}
connect();

figma.showUI('<body style="margin:0;font:12px -apple-system,sans-serif;padding:10px 12px;color:#1a7f37">● HotSpot Bridge 运行中<br><span style="color:#888">保持此窗口开启即保持自动通道</span></body>', { width: 230, height: 64 });

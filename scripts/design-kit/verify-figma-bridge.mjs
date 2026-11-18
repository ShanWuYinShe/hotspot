#!/usr/bin/env node
/**
 * HotSpot Figma Bridge 插件与服务端自动化冒烟测试
 * 用法：node scripts/design-kit/verify-figma-bridge.mjs
 */
import { readFileSync, existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawn } from 'node:child_process';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const BRIDGE_DIR = join(ROOT, 'scripts', 'design-kit', 'plugins', 'bridge');

console.log('=== [1/3] 检查 Bridge 清单与静态文件 ===');
const manifestPath = join(BRIDGE_DIR, 'manifest.json');
if (!existsSync(manifestPath)) throw new Error('manifest.json 不存在');
const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));

if (manifest.main !== 'code.js') throw new Error('manifest.main 应为 code.js');
if (manifest.ui !== 'ui.html') throw new Error('manifest.ui 应为 ui.html');
if (!manifest.networkAccess || !Array.isArray(manifest.networkAccess.allowedDomains)) {
  throw new Error('manifest 缺少 networkAccess.allowedDomains');
}
const hasWsDomain = manifest.networkAccess.allowedDomains.some((d) => d.includes('127.0.0.1:8787'));
if (!hasWsDomain) throw new Error('manifest 缺少 127.0.0.1:8787 域名授权');

const codeJs = readFileSync(join(BRIDGE_DIR, 'code.js'), 'utf8');
if (codeJs.includes('new WebSocket(')) {
  throw new Error('code.js 运行在 QuickJS 沙盒中，禁止直接调用 new WebSocket（必须在 ui.html 中执行）');
}
if (!codeJs.includes('figma.ui.onmessage')) {
  throw new Error('code.js 缺少 figma.ui.onmessage 桥接消息监听');
}

const uiHtml = readFileSync(join(BRIDGE_DIR, 'ui.html'), 'utf8');
if (!uiHtml.includes('new WebSocket(')) {
  throw new Error('ui.html 缺少 WebSocket 客户端初始化');
}

console.log('✓ 静态文件与清单声明检查通过');

console.log('=== [2/3] 启动 Bridge Server 协议冒烟测试 ===');
const TEST_PORT = 8798;
const serverProcess = spawn('bun', [join(BRIDGE_DIR, 'server.mjs')], {
  env: { ...process.env, BRIDGE_PORT: String(TEST_PORT) },
  stdio: ['ignore', 'pipe', 'pipe']
});

let serverExited = false;
serverProcess.on('exit', () => { serverExited = true; });

// 等待 server 监听
async function waitServerReady() {
  for (let i = 0; i < 30; i++) {
    try {
      const res = await fetch(`http://127.0.0.1:${TEST_PORT}/status`);
      if (res.ok) return;
    } catch (e) {}
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error('Server 启动超时');
}

try {
  await waitServerReady();
  console.log(`✓ Server 在 http://127.0.0.1:${TEST_PORT} 启动就绪`);

  // 1. 初始未连接状态
  const initStatus = await (await fetch(`http://127.0.0.1:${TEST_PORT}/status`)).json();
  if (initStatus.pluginConnected !== false) throw new Error('初始 pluginConnected 应为 false');

  // 2. 模拟 UI iframe WebSocket 连接
  console.log('=== [3/3] 模拟 Figma 插件交互与收发闭环 ===');
  const ws = new WebSocket(`ws://127.0.0.1:${TEST_PORT}`);

  await new Promise((resolve, reject) => {
    ws.onopen = resolve;
    ws.onerror = reject;
  });

  // 发送 hello 握手
  ws.send(JSON.stringify({ type: 'hello', plugin: 'hotspot-bridge', file: 'HotSpot Design Kit Mock', at: Date.now() }));
  await new Promise((r) => setTimeout(r, 100));

  const connectedStatus = await (await fetch(`http://127.0.0.1:${TEST_PORT}/status`)).json();
  if (!connectedStatus.pluginConnected || connectedStatus.pluginFile !== 'HotSpot Design Kit Mock') {
    throw new Error('握手后 status 状态不符合预期: ' + JSON.stringify(connectedStatus));
  }

  // 3. 测试 /eval 请求与响应
  let receivedPayload = null;
  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    receivedPayload = data;
    if (data.type === 'eval') {
      // 模拟 code.js 执行后回传
      ws.send(JSON.stringify({ type: 'result', id: data.id, ok: true, result: 42 }));
    } else if (data.type === 'save-version') {
      ws.send(JSON.stringify({ type: 'result', id: data.id, ok: true, result: 'version saved: ' + data.title }));
    }
  };

  const evalRes = await (await fetch(`http://127.0.0.1:${TEST_PORT}/eval`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ code: '40 + 2', timeoutMs: 5000 })
  })).json();

  if (!evalRes.ok || evalRes.result !== 42) {
    throw new Error('eval 闭环测试失败: ' + JSON.stringify(evalRes));
  }
  console.log('✓ /eval 转发与结果接收闭环正常');

  // 4. 测试 /save-version
  const saveRes = await (await fetch(`http://127.0.0.1:${TEST_PORT}/save-version`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: '细节测试版', description: '测试描述', timeoutMs: 5000 })
  })).json();

  if (!saveRes.ok || !String(saveRes.result).includes('细节测试版')) {
    throw new Error('save-version 闭环测试失败: ' + JSON.stringify(saveRes));
  }
  console.log('✓ /save-version 转发与结果接收闭环正常');

  ws.close();
  await new Promise((r) => setTimeout(r, 150));

  const afterCloseStatus = await (await fetch(`http://127.0.0.1:${TEST_PORT}/status`)).json();
  if (afterCloseStatus.pluginConnected !== false) throw new Error('断线后 pluginConnected 应为 false');

  console.log('✓ 断线状态清理正常');
  console.log('\n🎉 Figma Local Bridge 全套冒烟测试 100% 通过！');
} finally {
  serverProcess.kill('SIGTERM');
}

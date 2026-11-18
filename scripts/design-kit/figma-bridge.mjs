#!/usr/bin/env node
/**
 * Figma Bridge CLI — 通用 Figma 本地桥自动化控制台
 * 支持在任何项目中一键管理 Bridge 守护进程、执行 Plugin API 脚本、打版本历史。
 *
 * 用法：
 *   node scripts/design-kit/figma-bridge.mjs status               # 查看服务与桌面插件连接态
 *   node scripts/design-kit/figma-bridge.mjs start                # 后台启动 Bridge 服务
 *   node scripts/design-kit/figma-bridge.mjs stop                 # 停止 Bridge 服务
 *   node scripts/design-kit/figma-bridge.mjs restart              # 重启 Bridge 服务
 *   node scripts/design-kit/figma-bridge.mjs eval "<code>"        # 在 Figma 中执行代码（服务未起自动拉起）
 *   node scripts/design-kit/figma-bridge.mjs eval -f <file.js>    # 执行指定 JS 脚本文件
 *   node scripts/design-kit/figma-bridge.mjs save-version -t "标题" -d "描述"
 *   node scripts/design-kit/figma-bridge.mjs verify               # 运行协议闭环自检
 */
import { existsSync, readFileSync, unlinkSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawn } from 'node:child_process';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const BRIDGE_DIR = join(dirname(fileURLToPath(import.meta.url)), 'plugins', 'bridge');
const PID_FILE = join(BRIDGE_DIR, 'server.pid');
const PORT = Number(process.env.BRIDGE_PORT) || 8787;
const BASE_URL = `http://127.0.0.1:${PORT}`;

async function getStatus() {
  try {
    const res = await fetch(`${BASE_URL}/status`, { signal: AbortSignal.timeout(1500) });
    if (res.ok) return await res.json();
  } catch (e) {}
  return null;
}

async function isServerRunning() {
  const status = await getStatus();
  return status !== null;
}

async function startServer() {
  if (await isServerRunning()) {
    console.log(`✓ Bridge 服务已在运行中 (${BASE_URL})`);
    return true;
  }

  const child = spawn('bun', [join(BRIDGE_DIR, 'server.mjs')], {
    env: { ...process.env, BRIDGE_PORT: String(PORT) },
    detached: true,
    stdio: 'ignore'
  });
  child.unref();

  // 等待启动就绪
  for (let i = 0; i < 30; i++) {
    await new Promise((r) => setTimeout(r, 100));
    if (await isServerRunning()) {
      console.log(`✓ Bridge 服务已后台启动 (${BASE_URL}) [PID: ${child.pid}]`);
      return true;
    }
  }
  console.error(`✗ Bridge 服务启动超时，请确认本机已安装 bun 并检查端口 ${PORT}`);
  return false;
}

async function stopServer() {
  let pids = [];
  if (existsSync(PID_FILE)) {
    try {
      const p = parseInt(readFileSync(PID_FILE, 'utf8').trim(), 10);
      if (p) pids.push(p);
    } catch (e) {}
  }

  // 通过 lsof 自动发现可能残留的端口占用进程
  try {
    const { execSync } = await import('node:child_process');
    const portPids = execSync(`lsof -ti :${PORT}`, { stdio: ['ignore', 'pipe', 'ignore'] })
      .toString().trim().split('\n').map((x) => parseInt(x.trim(), 10)).filter(Boolean);
    for (const p of portPids) if (!pids.includes(p)) pids.push(p);
  } catch (e) {}

  for (const pid of pids) {
    try {
      process.kill(pid, 'SIGTERM');
      console.log(`✓ 已停止 Bridge 服务进程 (PID: ${pid})`);
    } catch (e) {
      if (e.code !== 'ESRCH') console.warn(`kill ${pid} error:`, e.message);
    }
  }

  try { unlinkSync(PID_FILE); } catch (e) {}
  await new Promise((r) => setTimeout(r, 300));

  if (await isServerRunning()) {
    console.warn('⚠️ 端口仍有响应，可能存在其它外部进程占用。');
  } else {
    console.log('✓ Bridge 服务已完全停止');
  }
}

async function ensureReady() {
  if (!(await isServerRunning())) {
    console.log('ℹ️ 检测到 Bridge 服务未启动，正在自动启动服务...');
    const ok = await startServer();
    if (!ok) process.exit(1);
  }
  const st = await getStatus();
  if (!st || !st.pluginConnected) {
    console.warn('\n⚠️  Bridge 服务已就绪，但 Figma 桌面插件尚未连接！');
    console.warn('   请在 Figma 桌面端菜单中：Plugins → Development → 运行「Figma Local Bridge」');
    console.warn(`   插件路径：${join(BRIDGE_DIR, 'manifest.json')}\n`);
  }
  return st;
}

async function runEval(code) {
  await ensureReady();
  try {
    const res = await fetch(`${BASE_URL}/eval`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code })
    });
    const result = await res.json();
    if (result.ok) {
      console.log(typeof result.result === 'object' ? JSON.stringify(result.result, null, 2) : result.result);
    } else {
      console.error('✗ 执行失败:', result.error);
      process.exit(1);
    }
  } catch (e) {
    console.error('✗ 请求异常:', e.message);
    process.exit(1);
  }
}

async function runSaveVersion(title, description) {
  await ensureReady();
  try {
    const res = await fetch(`${BASE_URL}/save-version`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title, description: description || '' })
    });
    const result = await res.json();
    if (result.ok) {
      console.log(`✓ 版本里程碑已保存: ${title}`);
    } else {
      console.error('✗ 保存失败:', result.error);
      process.exit(1);
    }
  } catch (e) {
    console.error('✗ 请求异常:', e.message);
    process.exit(1);
  }
}

// 主命令分发
const args = process.argv.slice(2);
const cmd = args[0] || 'status';

switch (cmd) {
  case 'status': {
    const st = await getStatus();
    console.log('=== Figma Local Bridge 运行状态 ===');
    console.log(`服务地址:     ${BASE_URL}`);
    console.log(`服务状态:     ${st ? '🟢 运行中' : '🔴 未运行'}`);
    if (st) {
      console.log(`运行时长:     ${st.uptimeSec}s`);
      console.log(`桌面插件:     ${st.pluginConnected ? '🟢 已连接' : '🔴 未连接 (请在 Figma 运行插件)'}`);
      if (st.pluginFile) console.log(`当前打开文件: ${st.pluginFile}`);
      console.log(`队列请求:     ${st.pending}`);
    }
    console.log('===================================');
    break;
  }

  case 'start':
  case 'daemon':
    await startServer();
    break;

  case 'stop':
    await stopServer();
    break;

  case 'restart':
    await stopServer();
    await startServer();
    break;

  case 'eval': {
    let code = '';
    if (args[1] === '-f' || args[1] === '--file') {
      const filePath = args[2];
      if (!filePath || !existsSync(filePath)) {
        console.error('✗ 脚本文件不存在:', filePath);
        process.exit(1);
      }
      code = readFileSync(filePath, 'utf8');
    } else {
      code = args.slice(1).join(' ');
    }
    if (!code.trim()) {
      console.error('用法: figma-bridge eval "figma.root.name" 或 figma-bridge eval -f script.js');
      process.exit(1);
    }
    await runEval(code);
    break;
  }

  case 'save-version': {
    let title = '';
    let desc = '';
    for (let i = 1; i < args.length; i++) {
      if ((args[i] === '-t' || args[i] === '--title') && args[i + 1]) title = args[++i];
      else if ((args[i] === '-d' || args[i] === '--desc' || args[i] === '--description') && args[i + 1]) desc = args[++i];
    }
    if (!title) {
      console.error('用法: figma-bridge save-version -t "版本标题" [-d "更新描述"]');
      process.exit(1);
    }
    await runSaveVersion(title, desc);
    break;
  }

  case 'verify': {
    const verifyScript = join(ROOT, 'scripts', 'design-kit', 'verify-figma-bridge.mjs');
    const child = spawn('node', [verifyScript], { stdio: 'inherit' });
    child.on('exit', (code) => process.exit(code || 0));
    break;
  }

  default:
    console.log(`未知命令: ${cmd}
支持命令:
  status        查看 Bridge 状态
  start         后台启动服务
  stop          停止服务
  restart       重启服务
  eval <code>   在 Figma 执行代码（未起服务自动拉起）
  save-version  打版本历史 (-t "标题" -d "描述")
  verify        运行自动化冒烟自检`);
    process.exit(1);
}

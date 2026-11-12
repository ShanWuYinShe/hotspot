#!/usr/bin/env node
/**
 * 抓取 HotSpot 前端页面在设计所需的结构数据（DOM 盒模型 + 计算样式 + 内联 SVG + 图片 URL）。
 * 输出：.workwork/design-kit/spec/raw/<device>/<screen>.json（不进 Git）。
 *
 * 用法：
 *   node scripts/design-kit/capture-screens.mjs                 # 抓全部（公开页 -> 管理员登录 -> 应用页）
 *   node scripts/design-kit/capture-screens.mjs --device mobile # 只抓某端
 *   node scripts/design-kit/capture-screens.mjs --only home,settings
 *   node scripts/design-kit/capture-screens.mjs --force        # 覆盖已存在
 * 前置：dev compose 已启动（http://localhost:8888），agent-browser 可用。
 */
import { spawnSync } from 'node:child_process';
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const OUT = join(ROOT, '.workwork', 'design-kit', 'spec', 'raw');
const SHOTS = join(ROOT, '.workwork', 'design-kit', 'shots');
const EXTRACT = join(ROOT, 'scripts', 'design-kit', 'extract-dom.js');
const BASE = process.env.CAPTURE_BASE || 'http://localhost:8888';
const AB = process.env.AGENT_BROWSER_BIN || 'agent-browser';

const DEVICES = [
  { id: 'mobile', width: 390, height: 844, scale: 2 },
  { id: 'tablet', width: 834, height: 1112, scale: 2 },
  { id: 'desktop', width: 1440, height: 900, scale: 1 }
];

const PUBLIC_ROUTES = [
  { id: 'login', path: '/login', title: '登录' },
  { id: 'register', path: '/register', title: '注册' },
  { id: 'share-user', path: '/share/3TU34P738C5W', title: '分享落地页（用户）' },
  { id: 'share-moment', path: '/share/U4L98K5YFDNT', title: '分享落地页（瞬间）' }
];

const APP_ROUTES = [
  { id: 'home', path: '/', title: '首页信息流' },
  { id: 'nearby', path: '/nearby', title: '附近枢纽' },
  { id: 'nearby-people', path: '/nearby/people', title: '附近的人' },
  { id: 'nearby-groups', path: '/nearby/groups', title: '附近群组' },
  { id: 'nearby-moments', path: '/nearby/moments', title: '附近瞬间' },
  { id: 'messages', path: '/messages', title: '消息列表' },
  { id: 'messages-peer', path: '/messages/user_seed_sophia', title: '私聊会话' },
  { id: 'messages-group', path: '/messages/group/group_seed_1', title: '群聊会话' },
  { id: 'notify', path: '/notify', title: '通知中心' },
  { id: 'search', path: '/search', title: '搜索' },
  { id: 'me', path: '/me', title: '个人中心' },
  { id: 'me-edit', path: '/me/edit', title: '编辑资料' },
  { id: 'me-relationships', path: '/me/relationships', title: '关系' },
  { id: 'me-moments', path: '/me/moments', title: '我的瞬间' },
  { id: 'me-groups', path: '/me/groups', title: '我的群组' },
  { id: 'me-reports', path: '/me/reports', title: '我的举报' },
  { id: 'settings', path: '/settings', title: '设置' },
  { id: 'map', path: '/map', title: '地图选点' },
  { id: 'group-detail', path: '/groups/group_seed_1', title: '群详情' },
  { id: 'moment-detail', path: '/moments/moment_seed_01', title: '瞬间详情' },
  { id: 'user-profile', path: '/users/user_seed_sophia', title: '他人主页' },
  { id: 'admin', path: '/admin', title: '管理后台概览' },
  { id: 'admin-announcements', path: '/admin/announcements', title: '管理后台-公告' },
  { id: 'admin-audit', path: '/admin/audit', title: '管理后台-审计' },
  { id: 'admin-config', path: '/admin/config', title: '管理后台-配置' },
  { id: 'admin-groups', path: '/admin/groups', title: '管理后台-群组' },
  { id: 'admin-rbac', path: '/admin/rbac', title: '管理后台-权限' },
  { id: 'admin-reports', path: '/admin/reports', title: '管理后台-举报' },
  { id: 'admin-stats', path: '/admin/stats', title: '管理后台-统计' },
  { id: 'admin-tags', path: '/admin/tags', title: '管理后台-标签' },
  { id: 'admin-users', path: '/admin/users', title: '管理后台-用户' }
];

const args = process.argv.slice(2);
const flag = (name) => args.includes('--' + name);
const value = (name, def) => { const i = args.indexOf('--' + name); return i >= 0 ? args[i + 1] : def; };
const only = value('only', '').split(',').filter(Boolean);
const deviceFilter = value('device', '').split(',').filter(Boolean);
const force = flag('force');
const noShots = flag('no-shots');

function run(binArgs, input, timeout = 90000) {
  const res = spawnSync(AB, binArgs, { encoding: 'utf8', input, timeout, maxBuffer: 64 * 1024 * 1024 });
  if (res.error) throw res.error;
  return { code: res.status, stdout: res.stdout || '', stderr: res.stderr || '' };
}
const ab = (argv, input) => run(argv, input);
function ok(r) { return r.code === 0; }
function sleep(ms) { return new Promise((r) => setTimeout(r, ms)); }

function log(...a) { console.log(new Date().toISOString().slice(11, 19), ...a); }

function setViewport(device) {
  const r = ab(['set', 'viewport', String(device.width), String(device.height), String(device.scale)]);
  if (!ok(r)) throw new Error('set viewport failed: ' + r.stderr);
}

function evalScript(script) {
  const r = ab(['eval', '--stdin'], script);
  if (!ok(r)) throw new Error('eval failed: ' + (r.stderr || r.stdout).slice(0, 600));
  let text = r.stdout.trim();
  let v;
  try { v = JSON.parse(text); } catch (e) { throw new Error('eval output not JSON: ' + text.slice(0, 300)); }
  if (typeof v === 'string') { try { v = JSON.parse(v); } catch (e) {} }
  return v;
}

const READY_SCRIPT = `(() => {
  const main = document.querySelector('#main-content');
  const txt = (main ? main.innerText : document.body.innerText) || '';
  const skeleton = document.querySelectorAll('[aria-busy="true"], .skeleton, [data-skeleton]').length;
  const broken = /加载失败|HTTP 50\d|网络异常/.test(txt);
  const pending = [...document.images].filter((i) => !i.complete || !i.naturalWidth).length;
  return JSON.stringify({ len: txt.trim().length, skeleton, broken, pending, path: location.pathname, title: document.title });
})()`;

async function waitReady(route, timeoutMs = 7000) {
  const start = Date.now();
  let last = null;
  while (Date.now() - start < timeoutMs) {
    await sleep(500);
    try { last = evalScript(READY_SCRIPT); } catch (e) { continue; }
    if (last.len > 70 && !last.broken && !last.pending) return last;
  }
  return last || { len: 0, broken: true };
}

function ensureTheme() {
  const script = `(() => { localStorage.setItem('appTheme','vibrant'); localStorage.setItem('colorScheme','light'); document.documentElement.className='vibrant'; return 'ok'; })()`;
  try { evalScript(script); } catch (e) {}
}

async function login() {
  log('login as admin');
  ab(['cookies', 'clear']);
  ab(['open', BASE + '/login']);
  await sleep(1500);
  ab(['fill', '#auth-field-loginId', 'admin@hotspot.local']);
  ab(['fill', '#auth-field-password', 'Admin12345678']);
  ab(['press', 'Enter']);
  const start = Date.now();
  while (Date.now() - start < 12000) {
    await sleep(600);
    try {
      const j = evalScript(`(() => JSON.stringify({ path: location.pathname, txt: document.body.innerText.slice(0, 200) }))()`);
      if (j.path !== '/login' && !j.txt.includes('登录后')) return true;
    } catch (e) {}
  }
  const html = ab(['get', 'url']);
  throw new Error('login failed, current url: ' + html.stdout.trim());
}

async function captureRoute(device, route) {
  const file = join(OUT, device.id, route.id + '.json');
  if (!force && existsSync(file)) { log('skip', device.id, route.id); return; }
  ab(['open', BASE + route.path]);
  const ready = await waitReady(route);
  if (ready.broken) log('WARN content may be broken:', device.id, route.id);
  try {
    evalScript(`(() => { const m = document.querySelector('#main-content'); if (m) m.scrollTop = 0; window.scrollTo(0, 0); return 'ok'; })()`);
  } catch (e) {}
  await sleep(350);
  const raw = evalScript(readFileSync(EXTRACT, 'utf8'));
  raw.meta = { route: route.path, id: route.id, title: route.title, device: device.id, capturedAt: new Date().toISOString(), pageTitle: ready.title, readyLen: ready.len };
  mkdirSync(join(OUT, device.id), { recursive: true });
  writeFileSync(file, JSON.stringify(raw));
  if (!noShots) {
    mkdirSync(join(SHOTS, device.id), { recursive: true });
    ab(['screenshot', join(SHOTS, device.id, route.id + '.png')]);
  }
  log('captured', device.id, route.id, 'nodes=' + raw.nodes.length);
}

async function main() {
  const devices = DEVICES.filter((d) => !deviceFilter.length || deviceFilter.includes(d.id));
  const pub = only.length ? PUBLIC_ROUTES.filter((r) => only.includes(r.id)) : PUBLIC_ROUTES;
  const app = only.length ? APP_ROUTES.filter((r) => only.includes(r.id)) : APP_ROUTES;
  for (const device of devices) {
    setViewport(device);
    if (pub.length) {
      ab(['cookies', 'clear']);
      ab(['open', BASE + '/login']);
      await sleep(1200);
      ensureTheme();
      for (const route of pub) await captureRoute(device, route);
    }
    if (app.length) {
      await login();
      ensureTheme();
      for (const route of app) await captureRoute(device, route);
    }
  }
  ab(['close']);
  log('done');
}

main().catch((e) => { console.error('CAPTURE FAILED:', e.message); process.exit(1); });

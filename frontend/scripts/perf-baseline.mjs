#!/usr/bin/env node
// 前端体验性能基线：用 agent-browser 逐页采集 FCP / LCP / CLS / TTFB / 传输字节，
// 输出 JSON + Markdown 摘要，用于改动前后相对对照（本地 dev 口径，非生产网络）。
//
// 用法：
//   cd frontend && bun run perf:baseline
//   bun run perf:baseline -- --base http://localhost:8889 --pages /,/messages
//   bun run perf:baseline -- --compare .workwork/browser-check/perf-<旧时间戳>/report.json
//   bun run perf:baseline -- --base https://chaoset.com --no-login --pages /login,/register   # 生产公开页（不尝试登录）
// 前置：dev 环境已启动且已 seed；agent-browser 已安装（~/.local/bin/agent-browser 或 AB 环境变量）。
//
// 输出：<仓库根>/.workwork/browser-check/perf-<时间戳>/{report.json,summary.md}

import { execFileSync } from 'node:child_process';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(SCRIPT_DIR, '../..');

function argValue(name, fallback) {
	const i = process.argv.indexOf(name);
	return i >= 0 && process.argv[i + 1] ? process.argv[i + 1] : fallback;
}

const BASE = argValue('--base', process.env.PERF_BASE || 'http://localhost:8888');
const SESSION = argValue('--session', 'hotspot-perf-' + process.pid);
const COMPARE = argValue('--compare', '');
const NO_LOGIN = process.argv.includes('--no-login');
const PAGES = argValue(
	'--pages',
	'/,/nearby/people,/nearby/moments,/nearby/groups,/messages,/search,/notify,/me,/settings'
)
	.split(',')
	.map((p) => p.trim())
	.filter(Boolean);
const LOGIN_ID = process.env.QA_LOGIN_ID || 'qa@hotspot.local';
const PASSWORD = process.env.QA_PASSWORD || 'Qa12345678';

function findAb() {
	if (process.env.AB) return process.env.AB;
	for (const p of [
		process.env.HOME + '/.local/bin/agent-browser',
		process.env.HOME + '/.bun/bin/agent-browser'
	]) {
		if (existsSync(p)) return p;
	}
	return 'agent-browser';
}

const AB = findAb();
const INIT_SCRIPT = resolve(REPO_ROOT, '.workwork/browser-check/.perf-init-' + process.pid + '.js');

function ab(args) {
	return execFileSync(AB, ['--init-script', INIT_SCRIPT, '--session', SESSION, ...args], {
		encoding: 'utf8',
		stdio: ['ignore', 'pipe', 'pipe']
	}).trim();
}

function sleep(ms) {
	Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, ms);
}

function jsonFromEval(expr) {
	const out = ab(['eval', expr]);
	// agent-browser 对字符串返回值可能加一层 JSON 引号（"{\"k\":1}"），最多解两层。
	let value;
	try {
		value = JSON.parse(out);
	} catch {
		const start = out.indexOf('{');
		const end = out.lastIndexOf('}');
		if (start < 0 || end < start) throw new Error('eval 结果不是 JSON：' + out.slice(0, 160));
		value = JSON.parse(out.slice(start, end + 1));
	}
	if (typeof value === 'string') value = JSON.parse(value);
	return value;
}

function cleanup() {
	try {
		ab(['close']);
	} catch {}
}

const INIT_JS = [
	'window.__perf = { lcp: 0, cls: 0 };',
	'new PerformanceObserver(function (list) {',
	'  for (const e of list.getEntries()) window.__perf.lcp = e.startTime;',
	'}).observe({ type: "largest-contentful-paint", buffered: true });',
	'new PerformanceObserver(function (list) {',
	'  for (const e of list.getEntries()) { if (!e.hadRecentInput) window.__perf.cls += e.value; }',
	'}).observe({ type: "layout-shift", buffered: true });'
].join('\n');

const METRICS_JS =
	'(() => { const nav = performance.getEntriesByType("navigation")[0] || {}; const paints = performance.getEntriesByType("paint"); const fcpEntry = paints.find((p) => p.name === "first-contentful-paint"); const res = performance.getEntriesByType("resource"); const transfer = res.reduce((s, e) => s + (e.transferSize || 0), 0); return JSON.stringify({ url: location.pathname + location.search, fcp: fcpEntry ? Math.round(fcpEntry.startTime) : null, lcp: window.__perf && window.__perf.lcp ? Math.round(window.__perf.lcp) : null, cls: +(window.__perf ? window.__perf.cls : 0).toFixed(4), ttfb: nav.responseStart ? Math.round(nav.responseStart) : null, dcl: nav.domContentLoadedEventEnd ? Math.round(nav.domContentLoadedEventEnd) : null, load: nav.loadEventEnd ? Math.round(nav.loadEventEnd) : null, requests: res.length, transfer_kb: Math.round(transfer / 1024) }); })()';

function main() {
	mkdirSync(dirname(INIT_SCRIPT), { recursive: true });
	writeFileSync(INIT_SCRIPT, INIT_JS + '\n');

	console.log('HotSpot 前端体验性能基线');
	console.log('BASE=' + BASE + ' SESSION=' + SESSION);
	console.log('PAGES=' + PAGES.join(','));

	try {
		if (NO_LOGIN) {
			// 生产公开页模式：只测公开路由，禁止对线上发起种子账号登录尝试
			console.log('登录：--no-login（只测公开页面，不尝试登录）');
		} else {
			ab(['open', BASE + '/']);
			sleep(1200);
			const pathNow = jsonFromEval('JSON.stringify({ path: location.pathname })');
			if (pathNow.path.includes('login')) {
				ab(['open', BASE + '/login']);
				sleep(1200);
				const login = jsonFromEval(
					'(() => { const ins = document.querySelectorAll("input"); if (ins.length < 2) return JSON.stringify({ ok: false }); const set = (el, v) => { const s = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set; s.call(el, v); el.dispatchEvent(new Event("input", { bubbles: true })); }; set(ins[0], "' +
						LOGIN_ID +
						'"); set(ins[1], "' +
						PASSWORD +
						'"); document.querySelector("button[type=submit]").click(); return JSON.stringify({ ok: true }); })()'
				);
				if (!login.ok) throw new Error('登录页找不到账号/密码输入框');
				sleep(2500);
				console.log('登录：QA 账号已尝试登录');
			} else {
				console.log('登录：会话已存在（跳过登录）');
			}
		}

		const results = [];
		for (const page of PAGES) {
			ab(['open', BASE + page]);
			sleep(2500);
			const m = jsonFromEval(METRICS_JS);
			results.push(m);
			console.log(
				'  ' +
					page.padEnd(20) +
					' LCP=' +
					String(m.lcp == null ? '-' : m.lcp).padStart(5) +
					'ms CLS=' +
					String(m.cls) +
					' FCP=' +
					String(m.fcp == null ? '-' : m.fcp).padStart(5) +
					'ms 传输=' +
					m.transfer_kb +
					'KB'
			);
		}

		const stamp = new Date()
			.toISOString()
			.replace(/[-:]/g, '')
			.replace(/\.\d+Z$/, '')
			.replace('T', '-');
		const outDir = resolve(REPO_ROOT, '.workwork/browser-check/perf-' + stamp);
		mkdirSync(outDir, { recursive: true });
		const report = { base: BASE, generated_at: new Date().toISOString(), pages: results, compare: COMPARE || null };
		writeFileSync(resolve(outDir, 'report.json'), JSON.stringify(report, null, 2));

		let compareRows = [];
		if (COMPARE) {
			const prev = JSON.parse(readFileSync(resolve(COMPARE), 'utf8'));
			compareRows = results.map((cur) => {
				const old = prev.pages.find((p) => p.url === cur.url);
				if (!old) return { url: cur.url, deltaLcp: null, deltaCls: null, deltaKb: null };
				return {
					url: cur.url,
					deltaLcp: (cur.lcp == null ? 0 : cur.lcp) - (old.lcp == null ? 0 : old.lcp),
					deltaCls: +(cur.cls - old.cls).toFixed(4),
					deltaKb: cur.transfer_kb - old.transfer_kb
				};
			});
		}

		const lines = [
			'# 前端体验性能基线（本地 dev 口径）',
			'',
			'BASE=' + BASE + '  生成时间 ' + report.generated_at,
			'',
			'| 页面 | FCP(ms) | LCP(ms) | CLS | TTFB(ms) | 请求数 | 传输(KB) |',
			'|---|---|---|---|---|---|---|'
		];
		for (const r of results) {
			lines.push(
				'| ' + r.url + ' | ' + (r.fcp == null ? '-' : r.fcp) + ' | ' + (r.lcp == null ? '-' : r.lcp) + ' | ' + r.cls + ' | ' + (r.ttfb == null ? '-' : r.ttfb) + ' | ' + r.requests + ' | ' + r.transfer_kb + ' |'
			);
		}
		if (compareRows.length) {
			lines.push('', '## 对比：' + COMPARE + '（正数=变慢）');
			lines.push('', '| 页面 | ΔLCP(ms) | ΔCLS | Δ传输(KB) |', '|---|---|---|---|');
			for (const c of compareRows) {
				lines.push(
					'| ' + c.url + ' | ' + (c.deltaLcp === null ? '（新增页）' : c.deltaLcp) + ' | ' + (c.deltaCls === null ? '-' : c.deltaCls) + ' | ' + (c.deltaKb === null ? '-' : c.deltaKb) + ' |'
				);
			}
		}
		lines.push('', '> 本地 dev（Docker + 本机代理）数据仅作相对参考；生产需真实网络另测。');
		writeFileSync(resolve(outDir, 'summary.md'), lines.join('\n') + '\n');
		console.log('');
		console.log('输出：' + outDir);
		console.log('   report.json / summary.md');
	} finally {
		cleanup();
	}
}

try {
	main();
} catch (err) {
	cleanup();
	console.error('性能基线失败：' + (err && err.message ? err.message : err));
	process.exit(1);
}

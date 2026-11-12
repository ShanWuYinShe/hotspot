#!/usr/bin/env node
/**
 * 从运行中的前端抓取 6 个主题模式（3 主题 × 明暗）下所有 --cst-* token 的最终解析色，
 * 供生成 Figma Variables 与设计稿配色映射使用。
 * 输出：docs/design-kit/tokens/runtime.json
 * 用法：node scripts/design-kit/capture-tokens.mjs
 * 前置：dev compose 已启动（http://localhost:8888），agent-browser 可用。
 */
import { spawnSync } from 'node:child_process';
import { writeFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execSync } from 'node:child_process';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const OUT = join(ROOT, 'docs', 'design-kit', 'tokens', 'runtime.json');
const BASE = process.env.CAPTURE_BASE || 'http://localhost:8888';
const AB = process.env.AGENT_BROWSER_BIN || 'agent-browser';

const SCRIPT = `(() => {
  const names = new Set();
  for (const sheet of document.styleSheets) {
    let rules; try { rules = sheet.cssRules; } catch (e) { continue; }
    const walk = (list) => { for (const rule of list) { if (rule.style) { for (let i = 0; i < rule.style.length; i++) { const p = rule.style[i]; if (p && p.startsWith('--cst-')) names.add(p); } } if (rule.cssRules) walk(rule.cssRules); } };
    walk(rules);
  }
  const cv = document.createElement('canvas'); cv.width = 1; cv.height = 1;
  const ctx = cv.getContext('2d', { willReadFrequently: true });
  const probe = document.createElement('div'); probe.style.display = 'none'; document.body.appendChild(probe);
  const cache = new Map();
  const resolveColor = (value) => {
    if (!/^(#|rgb|hsl|color|oklab|oklch|lab|lch|var|color-mix)/.test(String(value).trim())) return null;
    if (cache.has(value)) return cache.get(value);
    let out = null;
    try {
      probe.style.color = '';
      probe.style.color = value;
      const computed = getComputedStyle(probe).color;
      ctx.clearRect(0, 0, 1, 1);
      ctx.fillStyle = '#010203';
      ctx.fillStyle = computed;
      ctx.fillRect(0, 0, 1, 1);
      const d = ctx.getImageData(0, 0, 1, 1).data;
      out = 'rgba(' + d[0] + ', ' + d[1] + ', ' + d[2] + ', ' + (d[3] / 255).toFixed(3) + ')';
    } catch (e) { out = null; }
    cache.set(value, out);
    return out;
  };
  const combos = [
    ['vibrant', 'light', ['vibrant']],
    ['vibrant', 'dark', ['vibrant', 'dark']],
    ['forest', 'light', ['forest']],
    ['forest', 'dark', ['forest', 'dark']],
    ['inkwash', 'light', ['inkwash']],
    ['inkwash', 'dark', ['inkwash', 'dark']]
  ];
  const root = document.documentElement;
  const previousClass = root.className;
  const data = {};
  for (const combo of combos) {
    const theme = combo[0], mode = combo[1], cls = combo[2];
    root.className = cls.join(' ');
    const cs = getComputedStyle(root);
    const vars = {};
    for (const name of names) {
      const raw = cs.getPropertyValue(name).trim();
      if (!raw) continue;
      vars[name] = { raw: raw, color: resolveColor(raw) };
    }
    data[theme + '-' + mode] = vars;
  }
  probe.remove();
  root.className = previousClass;
  return JSON.stringify({ count: names.size, combos: Object.keys(data), data: data });
})()`;

function ab(argv, input) {
  const res = spawnSync(AB, argv, { encoding: 'utf8', input, timeout: 90000, maxBuffer: 64 * 1024 * 1024 });
  if (res.error) throw res.error;
  return { code: res.status, stdout: res.stdout || '', stderr: res.stderr || '' };
}

let commit = 'dev';
try { commit = execSync('git rev-parse --short HEAD', { cwd: ROOT }).toString().trim(); } catch (e) {}

const open = ab(['open', BASE + '/']);
if (open.code !== 0) throw new Error('open failed: ' + open.stderr);
ab(['set', 'viewport', '1440', '900', '1']);
ab(['wait', '1500']);
const res = ab(['eval', '--stdin'], SCRIPT);
if (res.code !== 0) throw new Error('eval failed: ' + (res.stderr || res.stdout).slice(0, 500));
let parsed = JSON.parse(res.stdout.trim());
if (typeof parsed === 'string') parsed = JSON.parse(parsed);
parsed.meta = { generatedAt: new Date().toISOString(), commit: commit, base: BASE };
mkdirSync(dirname(OUT), { recursive: true });
writeFileSync(OUT, JSON.stringify(parsed, null, 2) + '\n');
console.log('tokens runtime written:', OUT, Object.keys(parsed.data).length, 'modes,', parsed.count, 'vars');

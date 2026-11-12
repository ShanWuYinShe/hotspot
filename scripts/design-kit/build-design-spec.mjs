#!/usr/bin/env node
/**
 * 将 capture-screens 抓取的原始 DOM 规格压缩、去重并映射到 design token，
 * 生成 Figma 插件与预览渲染器共用的设计规格：
 *   docs/design-kit/spec/spec.json        全量规格（tokens + 字典 + 屏幕）
 *   docs/design-kit/spec/<device>.json    按设备拆分（Figma 插件按端加载，控制体积）
 * 用法：node scripts/design-kit/build-design-spec.mjs
 */
import { readFileSync, writeFileSync, mkdirSync, readdirSync, existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createHash } from 'node:crypto';
import { execSync } from 'node:child_process';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const RAW = join(ROOT, '.workwork', 'design-kit', 'spec', 'raw');
const OUT = join(ROOT, '.workwork', 'design-kit', 'spec');
const DEVICES = ['mobile', 'tablet', 'desktop'];

let commit = 'dev';
try { commit = execSync('git rev-parse --short HEAD', { cwd: ROOT }).toString().trim(); } catch (e) {}

function parseRgba(s) {
  const m = String(s || '').match(/rgba?\(([^)]+)\)/);
  if (!m) return null;
  const p = m[1].split(/[ ,/]+/).filter(Boolean).map(Number);
  if (p.length < 3) return null;
  return { r: p[0], g: p[1], b: p[2], a: p[3] === undefined ? 1 : p[3] };
}
function hexOf(c) {
  return '#' + [c.r, c.g, c.b].map((v) => Math.round(v).toString(16).padStart(2, '0')).join('');
}
function parseHex(h) {
  if (!h || h[0] !== '#') return null;
  if (h.length === 7) return { r: parseInt(h.slice(1, 3), 16), g: parseInt(h.slice(3, 5), 16), b: parseInt(h.slice(5, 7), 16), a: 1 };
  return null;
}
const round = (n) => Math.round(n * 10) / 10;
const key3 = (c) => c.r + ',' + c.g + ',' + c.b;

// ---------- 主题 token → 颜色映射 ----------
const runtime = JSON.parse(readFileSync(join(ROOT, 'docs/design-kit/tokens/runtime.json'), 'utf8'));
const MODES = Object.keys(runtime.data);
const baseMode = runtime.data['vibrant-light'];
const tokenNames = Object.keys(baseMode).map((n) => n.replace('--cst-', ''));
const semantic = tokenNames.filter((n) => !n.startsWith('palette-'));
const palette = tokenNames.filter((n) => n.startsWith('palette-'));
const priority = (n) => (n.startsWith('palette-') ? 1 : 0);

const exactColorMap = new Map();
const rgbColorMap = new Map();
const tokenColors = [];
const tokenColorsFull = [];
for (const name of tokenNames) {
  const v = baseMode['--cst-' + name];
  if (!v || !v.color) continue;
  const c = parseRgba(v.color);
  if (!c) continue;
  const hex = hexOf(c);
  tokenColors.push({ name, hex, a: +c.a.toFixed(3) });
  tokenColorsFull.push({ name, hex });
  const ek = key3(c) + '|' + (+c.a.toFixed(2));
  if (!exactColorMap.has(ek) || priority(exactColorMap.get(ek)) > priority(name)) exactColorMap.set(ek, name);
  const rk = key3(c);
  if (!rgbColorMap.has(rk) || priority(rgbColorMap.get(rk)) > priority(name)) rgbColorMap.set(rk, name);
}
const EXCLUDED_TOKENS = new Set(['background', 'foreground', 'destructive', 'border', 'ring']);
const ROLE_ORDER = {
  text: ['content-inverse', 'content-inverse-muted', 'content-inverse-secondary', 'text-on-accent', 'text-on-status', 'text-primary', 'text-secondary', 'content-secondary', 'text-accent-on-soft', 'text-status-online-on-soft', 'text-status-warning-on-soft', 'text-status-danger-on-soft'],
  fill: ['surface-primary', 'surface-page', 'surface-island', 'surface-panel', 'surface-subtle', 'surface-raised', 'surface-scrim', 'surface-media-scrim-card', 'surface-media-scrim-strong', 'accent-default', 'accent-soft', 'accent-border', 'status-online', 'status-danger', 'status-warning', 'border-default'],
  border: ['border-default', 'border-strong', 'border-subtle', 'border-inverse', 'accent-border', 'status-danger', 'status-warning', 'status-online']
};
const tokenPool = tokenColors.filter((t) => !EXCLUDED_TOKENS.has(t.name));
const ROLE_EXCLUDE_PREFIX = {
  fill: ['content-', 'text-', 'border-'],
  text: ['surface-', 'palette-'],
  border: ['content-', 'text-', 'surface-']
};
const orderedByRole = {};
for (const role of Object.keys(ROLE_ORDER)) {
  const excludes = ROLE_EXCLUDE_PREFIX[role] || [];
  const pool = tokenPool.filter((t) => !excludes.some((p) => t.name.indexOf(p) === 0));
  const head = [];
  for (const name of ROLE_ORDER[role]) { const t = pool.find((x) => x.name === name); if (t) head.push(t); }
  orderedByRole[role] = head.concat(pool.filter((t) => !head.includes(t)));
}
orderedByRole.generic = tokenPool;
const colorCache = new Map();
function colorToToken(hex, alpha, role) {
  const c = parseHex(hex);
  const a = alpha === undefined ? 1 : alpha;
  if (!c) return { c: hex || '#000000', a: +a.toFixed(3) };
  const key = hex + '|' + a.toFixed(2) + '|' + (role || 'generic');
  if (colorCache.has(key)) return colorCache.get(key);
  const pool = orderedByRole[role] || orderedByRole.generic;
  let result = null;
  for (const t of pool) {
    const tc = parseHex(t.hex);
    if (tc.r === c.r && tc.g === c.g && tc.b === c.b && Math.abs(t.a - a) <= 0.02) { result = { c: t.name, a: +a.toFixed(3) }; break; }
  }
  if (!result) {
    for (const t of pool) {
      const tc = parseHex(t.hex);
      if (tc.r === c.r && tc.g === c.g && tc.b === c.b) { result = { c: t.name, a: +a.toFixed(3) }; break; }
    }
  }
  if (!result) {
    let best = null;
    let bestDist = 7;
    for (const t of pool) {
      const tc = parseHex(t.hex);
      const dist = Math.abs(tc.r - c.r) + Math.abs(tc.g - c.g) + Math.abs(tc.b - c.b);
      if (dist < bestDist) { bestDist = dist; best = t; }
    }
    if (best) result = { c: best.name, a: +a.toFixed(3) };
  }
  const out = result || { c: hex, a: +a.toFixed(3) };
  colorCache.set(key, out);
  return out;
}

// ---------- 字典去重 ----------
const svgDict = new Map();
const svgList = [];
function svgId(outer) {
  const h = createHash('sha1').update(outer).digest('hex').slice(0, 10);
  if (!svgDict.has(h)) { svgList.push(outer); svgDict.set(h, svgList.length - 1); }
  return svgDict.get(h);
}
const imageDict = new Map();
const imageList = [];
function imageId(url, thumb) {
  if (!url) return null;
  if (!imageDict.has(url)) { imageList.push({ url, data: thumb || null }); imageDict.set(url, imageList.length - 1); }
  return imageDict.get(url);
}
const shadowDict = new Map();
const shadowList = [];
function shadowId(str) {
  const h = createHash('sha1').update(str).digest('hex').slice(0, 10);
  if (shadowDict.has(h)) return shadowDict.get(h);
  const effects = [];
  for (const part of splitTop(str, ',')) {
    const m = part.trim().match(/(rgba?\([^)]*\)|#[0-9a-fA-F]{3,8})\s+(-?[\d.]+)px\s+(-?[\d.]+)px\s+([\d.]+)px(?:\s+(-?[\d.]+)px)?/);
    if (!m) continue;
    const col = parseHex(m[1]) ? { hex: m[1], a: 1 } : (function () { const q = parseRgba(m[1]); return q ? { hex: hexOf(q), a: q.a } : null; })();
    if (!col) continue;
    const tok = colorToToken(col.hex, col.a, 'fill');
    effects.push({ x: +m[2], y: +m[3], blur: +m[4], spread: m[5] === undefined ? 0 : +m[5], c: tok.c, a: tok.a });
  }
  if (!effects.length) return null;
  shadowList.push(effects);
  shadowDict.set(h, shadowList.length - 1);
  return shadowList.length - 1;
}
const gradientDict = new Map();
const gradientList = [];
function gradientId(str) {
  const h = createHash('sha1').update(str).digest('hex').slice(0, 10);
  if (gradientDict.has(h)) return gradientDict.get(h);
  const g = parseGradient(str);
  if (!g) return null;
  gradientList.push(g);
  gradientDict.set(h, gradientList.length - 1);
  return gradientList.length - 1;
}
function splitTop(s, sep) {
  const out = []; let depth = 0, cur = '';
  for (const ch of s) { if (ch === '(') depth++; if (ch === ')') depth--; if (ch === sep && depth === 0) { out.push(cur); cur = ''; } else cur += ch; }
  if (cur.trim()) out.push(cur);
  return out;
}
const ANGLE = { 'to top': 0, 'to top right': 45, 'to right': 90, 'to bottom right': 135, 'to bottom': 180, 'to bottom left': 225, 'to left': 270, 'to top left': 315 };
function parseGradient(str) {
  const layers = splitTop(str, ',').map((s) => s.trim()).filter(Boolean);
  const layer = layers.filter((s) => /gradient\(/.test(s)).pop();
  if (!layer) return null;
  const lg = layer.match(/linear-gradient\(([\s\S]*)\)$/);
  const rg = layer.match(/radial-gradient\(([\s\S]*)\)$/);
  if (!lg && !rg) return null;
  const parts = splitTop(lg ? lg[1] : rg[1], ',');
  let angle = 180;
  let stops = parts;
  const head = parts[0].trim();
  if (lg && /^-?[\d.]+deg$/.test(head)) { angle = parseFloat(head); stops = parts.slice(1); }
  else if (lg && ANGLE[head] !== undefined) { angle = ANGLE[head]; stops = parts.slice(1); }
  const parsed = [];
  for (const s of stops) {
    const m = s.trim().match(/^(rgba?\([^)]*\)|#[0-9a-fA-F]{3,8})\s*([\d.]+%)?$/);
    if (!m) continue;
    const q = parseRgba(m[1]);
    if (!q) continue;
    const tok = colorToToken(hexOf(q), q.a, 'fill');
    parsed.push({ c: tok.c, a: tok.a, p: m[2] ? parseFloat(m[2]) / 100 : null });
  }
  if (parsed.length < 2) return null;
  if (parsed[0].p === null) parsed[0].p = 0;
  if (parsed[parsed.length - 1].p === null) parsed[parsed.length - 1].p = 1;
  for (let i = 1; i < parsed.length - 1; i++) if (parsed[i].p === null) parsed[i].p = (parsed[i - 1].p + parsed[i + 1].p) / 2;
  return { type: lg ? 'linear' : 'radial', angle: angle, stops: parsed };
}

// ---------- 单节点压缩 ----------
function compactNode(n) {
  const o = { t: n.svg ? 'v' : n.img ? 'i' : n.canvas || n.video ? 'c' : n.text ? 't' : 'r', x: round(n.x), y: round(n.y), w: round(n.w), h: round(n.h) };
  if (n.bg) { const f = colorToToken(n.bg, n.bgA === undefined ? 1 : n.bgA, 'fill'); o.f = f.c; if (f.a < 1) o.fa = f.a; }
  if (n.bgImg) { const g = gradientId(n.bgImg); if (g !== null) o.g = g; }
  if (n.border) { const b = colorToToken(n.border.c, n.border.a, 'border'); o.bd = [n.border.w, b.c, b.a < 1 ? b.a : 1]; }
  if (n.r) o.r = n.r.map((v) => round(v));
  if (n.shadow) { const s = shadowId(n.shadow); if (s !== null) o.sh = s; }
  if (n.clip) { o.cl = [round(n.clip.x), round(n.clip.y), round(n.clip.w), round(n.clip.h)].concat(n.clip.r.map((v) => round(v))); }
  if (n.opacity !== undefined) o.o = +n.opacity.toFixed(2);
  if (n.text && n.w > 1.5 && n.h > 1.5) {
    const f = n.font || {};
    o.tx = n.text;
    o.s = f.size; o.fw = f.weight || 400; o.lh = f.lh || 0; if (f.ls) o.ls = f.ls;
    if (f.align && f.align !== 'start') o.al = f.align;
    const c = colorToToken(n.c || '#111111', n.cA === undefined ? 1 : n.cA, 'text');
    o.c = c.c; if (c.a < 1) o.ca = c.a;
    if (/[\u2e80-\u9fff\uf900-\ufaff]/.test(n.text)) o.cj = 1;
    if (n.deco) o.deco = n.deco;
  } else if (n.svg && n.svg.outer) {
    o.sv = svgId(n.svg.outer);
    if (n.c) { const c = colorToToken(n.c, n.cA === undefined ? 1 : n.cA, 'text'); o.c = c.c; if (c.a < 1) o.ca = c.a; }
  } else if (n.img) {
    const id = imageId(n.img, n.thumb || null);
    if (id !== null) o.im = id;
    if (n.fit && n.fit !== 'cover') o.fit = n.fit;
  }
  return o;
}

// ---------- 生成 ----------
mkdirSync(OUT, { recursive: true });
const allScreens = [];
const perDevice = {};
for (const device of DEVICES) {
  const dir = join(RAW, device);
  if (!existsSync(dir)) continue;
  const files = readdirSync(dir).filter((f) => f.endsWith('.json')).sort();
  const screens = [];
  for (const file of files) {
    const raw = JSON.parse(readFileSync(join(dir, file), 'utf8'));
    const meta = raw.meta || {};
    screens.push({
      id: meta.id || file.replace(/\.json$/, ''),
      title: meta.title || meta.id || file,
      route: meta.route || '',
      device,
      w: raw.vw,
      h: raw.vh,
      nodes: raw.nodes.map(compactNode)
    });
  }
  perDevice[device] = screens;
  allScreens.push(...screens);
}

const variables = [];
for (const name of tokenNames) {
  const values = {};
  let hasColor = false;
  for (const mode of MODES) {
    const v = runtime.data[mode] && runtime.data[mode]['--cst-' + name];
    if (v && v.color) { const c = parseRgba(v.color); if (c) { values[mode] = hexOf(c) + (c.a < 1 ? '|' + (+c.a.toFixed(3)) : ''); hasColor = true; } }
  }
  if (hasColor) variables.push({ name, values });
}

const spec = {
  meta: {
    generatedAt: new Date().toISOString(),
    commit,
    modes: MODES,
    devices: DEVICES.filter((d) => perDevice[d]),
    screenCount: allScreens.length,
    nodeCount: allScreens.reduce((a, s) => a + s.nodes.length, 0),
    svgCount: svgList.length,
    imageCount: imageList.length,
    gradientCount: gradientList.length,
    shadowCount: shadowList.length
  },
  variables,
  svgDict: svgList,
  imageDict: imageList,
  gradientDict: gradientList,
  shadowDict: shadowList,
  screens: allScreens
};

const tokenColorsByName = {};
for (const tc of tokenColorsFull) tokenColorsByName[tc.name] = tc.hex;

// 每台设备只打包自己引用的字典，控制 Figma 插件体积
function packDevice(screens) {
  const used = { sv: new Set(), im: new Set(), g: new Set(), sh: new Set() };
  for (const s of screens) for (const n of s.nodes) {
    if (n.sv !== undefined) used.sv.add(n.sv);
    if (n.im !== undefined) used.im.add(n.im);
    if (n.g !== undefined) used.g.add(n.g);
    if (n.sh !== undefined) used.sh.add(n.sh);
  }
  function pick(list, usedSet, key) {
    const map = new Map(); const out = [];
    for (const idx of [...usedSet].sort((a, b) => a - b)) { map.set(idx, out.length); out.push(list[idx]); }
    for (const s of screens) for (const n of s.nodes) if (n[key] !== undefined) n[key] = map.get(n[key]);
    return out;
  }
  const svgDict = pick(svgList, used.sv, 'sv');
  const imageDict = pick(imageList, used.im, 'im');
  const gradientDict = pick(gradientList, used.g, 'g');
  const shadowDict = pick(shadowList, used.sh, 'sh');
  return { svgDict, imageDict, gradientDict, shadowDict };
}

spec.tokenColors = tokenColorsByName;
writeFileSync(join(OUT, 'spec.json'), JSON.stringify(spec));
console.log('spec written:', OUT);
console.log('screens', spec.meta.screenCount, 'nodes', spec.meta.nodeCount, 'svgs', spec.meta.svgCount, 'images', spec.meta.imageCount, 'gradients', spec.meta.gradientCount, 'shadows', spec.meta.shadowCount);

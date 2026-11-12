#!/usr/bin/env node
/**
 * 将 capture-screens 抓取的原始 DOM 规格渲染为 SVG（用于设计稿预览与校验）。
 * 用法：node scripts/design-kit/render-svg.mjs <raw.json> <out.svg>
 * 图片按 URL 抓取并缓存到 .workwork/design-kit/image-cache。
 */
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import { dirname, join, extname } from 'node:path';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const CACHE = join(ROOT, '.workwork', 'design-kit', 'image-cache');

function esc(s) { return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;'); }
function round(n) { return Math.round(n * 10) / 10; }
function attrs(o) { return Object.entries(o).filter(([, v]) => v !== null && v !== undefined && v !== '' && v !== false).map(([k, v]) => ' ' + k + '="' + String(v).replace(/"/g, '&quot;') + '"').join(''); }

function parseColor(c) {
  if (!c) return null;
  if (c[0] === '#') {
    if (c.length === 7) return { r: parseInt(c.slice(1, 3), 16), g: parseInt(c.slice(3, 5), 16), b: parseInt(c.slice(5, 7), 16), a: 1 };
    if (c.length === 4) return { r: parseInt(c[1] + c[1], 16), g: parseInt(c[2] + c[2], 16), b: parseInt(c[3] + c[3], 16), a: 1 };
  }
  const m = c.match(/rgba?\(([^)]+)\)/);
  if (m) {
    const p = m[1].split(/[ ,/]+/).filter(Boolean).map(Number);
    return { r: p[0], g: p[1], b: p[2], a: p[3] === undefined ? 1 : p[3] };
  }
  const cm = c.match(/color\(srgb\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)(?:\s*\/\s*([\d.]+))?\)/);
  if (cm) return { r: Math.round(cm[1] * 255), g: Math.round(cm[2] * 255), b: Math.round(cm[3] * 255), a: cm[4] === undefined ? 1 : Number(cm[4]) };
  return null;
}
function paint(c) { const p = parseColor(c); return p ? 'rgb(' + p.r + ',' + p.g + ',' + p.b + ')' : 'none'; }

function splitTopLevel(s, sep) {
  const out = []; let depth = 0, cur = '';
  for (const ch of s) {
    if (ch === '(') depth++;
    if (ch === ')') depth--;
    if (ch === sep && depth === 0) { out.push(cur); cur = ''; } else cur += ch;
  }
  if (cur.trim()) out.push(cur);
  return out;
}

const ANGLE_KEYWORDS = { 'to top': 0, 'to top right': 45, 'to right': 90, 'to bottom right': 135, 'to bottom': 180, 'to bottom left': 225, 'to left': 270, 'to top left': 315 };

function gradientDefs(imgStr, id) {
  const layers = splitTopLevel(imgStr, ',').map((s) => s.trim()).filter(Boolean);
  const layer = [...layers].reverse().find((s) => /gradient\(/.test(s)) || imgStr;
  const lg = layer.match(/linear-gradient\(([\s\S]*)\)$/);
  const rg = layer.match(/radial-gradient\(([\s\S]*)\)$/);
  if (!lg && !rg) return null;
  const body = lg ? lg[1] : rg[1];
  const parts = splitTopLevel(body, ',');
  let angle = '180deg';
  let stops = parts;
  const head = parts[0].trim();
  if (lg && /^-?[\d.]+deg$/.test(head)) { angle = head; stops = parts.slice(1); }
  else if (lg && ANGLE_KEYWORDS[head] !== undefined) { angle = ANGLE_KEYWORDS[head] + 'deg'; stops = parts.slice(1); }
  const parsed = [];
  for (const s of stops) {
    const m = s.trim().match(/^(?:color\([^)]*\)|rgba?\([^)]*\)|#[0-9a-fA-F]{3,8}|[a-z]+)\s*([\d.]+%)?$/);
    if (!m) continue;
    const colorStr = s.trim().replace(/\s*[\d.]+%$/, '');
    const pos = m[1] ? parseFloat(m[1]) / 100 : null;
    const col = parseColor(colorStr);
    if (col) parsed.push({ r: col.r, g: col.g, b: col.b, a: col.a, pos: pos });
  }
  if (!parsed.length) return null;
  if (parsed[0].pos === null) parsed[0].pos = 0;
  if (parsed[parsed.length - 1].pos === null) parsed[parsed.length - 1].pos = 1;
  for (let i = 1; i < parsed.length - 1; i++) if (parsed[i].pos === null) parsed[i].pos = (parsed[i - 1].pos + parsed[i + 1].pos) / 2;
  let stopsXml = '';
  for (const p of parsed) stopsXml += '<stop' + attrs({ offset: round(p.pos * 100) + '%', 'stop-color': 'rgb(' + p.r + ',' + p.g + ',' + p.b + ')', 'stop-opacity': round(p.a) }) + '/>';
  if (lg) {
    const rad = (parseFloat(angle) - 90) * Math.PI / 180;
    const x1 = 50 - Math.cos(rad) * 50, y1 = 50 - Math.sin(rad) * 50;
    const x2 = 50 + Math.cos(rad) * 50, y2 = 50 + Math.sin(rad) * 50;
    return '<linearGradient' + attrs({ id: id, x1: round(x1) + '%', y1: round(y1) + '%', x2: round(x2) + '%', y2: round(y2) + '%' }) + '>' + stopsXml + '</linearGradient>';
  }
  return '<radialGradient' + attrs({ id: id, cx: '50%', cy: '50%', r: '75%' }) + '>' + stopsXml + '</radialGradient>';
}

function radiusValues(radius) {
  const m = (radius || '').match(/([\d.]+)px/g) || [];
  const v = m.map((x) => parseFloat(x));
  if (!v.length) return [0, 0, 0, 0];
  if (v.length === 1) return [v[0], v[0], v[0], v[0]];
  if (v.length === 2) return [v[0], v[1], v[0], v[1]];
  if (v.length === 3) return [v[0], v[1], v[2], v[1]];
  return v;
}

function roundedPath(x, y, w, h, radii) {
  const [tl, tr, br, bl] = radii.map((r) => Math.max(0, Math.min(r, Math.min(w, h) / 2)));
  if (!tl && !tr && !br && !bl) return 'M' + x + ' ' + y + 'H' + (x + w) + 'V' + (y + h) + 'H' + x + 'Z';
  return 'M' + (x + tl) + ' ' + y + 'H' + (x + w - tr) + (tr ? 'A' + tr + ' ' + tr + ' 0 0 1 ' + (x + w) + ' ' + (y + tr) : '') + 'V' + (y + h - br) + (br ? 'A' + br + ' ' + br + ' 0 0 1 ' + (x + w - br) + ' ' + (y + h) : '') + 'H' + (x + bl) + (bl ? 'A' + bl + ' ' + bl + ' 0 0 1 ' + x + ' ' + (y + h - bl) : '') + 'V' + (y + tl) + (tl ? 'A' + tl + ' ' + tl + ' 0 0 1 ' + (x + tl) + ' ' + y : '') + 'Z';
}

function fontFamilyFor(text) { return /[\u2e80-\u9fff\uf900-\ufaff]/.test(text) ? '"PingFang SC", "Noto Sans SC", sans-serif' : '"Inter", "PingFang SC", sans-serif'; }

function estimateWidth(ch, size, weight) {
  if (ch === ' ') return size * 0.28;
  if (/[\u2e80-\u9fff\uf900-\ufaff\uff00-\uffef]/.test(ch)) return size;
  if (/[A-Z0-9]/.test(ch)) return size * (weight >= 600 ? 0.62 : 0.58);
  if (/[a-z]/.test(ch)) return size * (weight >= 600 ? 0.56 : 0.5);
  if (/[.,:;'"!|]/.test(ch)) return size * 0.3;
  if (ch === '·') return size * 0.4;
  return size * 0.55;
}
function wrapText(text, size, weight, maxW) {
  if (!maxW || maxW < size) return [text];
  const lines = []; let cur = '', curW = 0;
  for (const ch of text) {
    const w = estimateWidth(ch, size, weight);
    if (cur && curW + w > maxW) { lines.push(cur); cur = ch; curW = w; }
    else { cur += ch; curW += w; }
  }
  if (cur) lines.push(cur);
  return lines;
}

function shadowFilter(shadow, id) {
  const m = shadow.match(/(rgba?\([^)]*\)|#[0-9a-fA-F]{3,8})\s+(-?[\d.]+)px\s+(-?[\d.]+)px\s+([\d.]+)px(?:\s+(-?[\d.]+)px)?/);
  if (!m) return null;
  const col = parseColor(m[1]);
  if (!col) return null;
  return '<filter' + attrs({ id: id, x: '-50%', y: '-50%', width: '200%', height: '200%' }) + '><feDropShadow' + attrs({ dx: m[2], dy: m[3], stdDeviation: parseFloat(m[4]) / 2, 'flood-color': 'rgb(' + col.r + ',' + col.g + ',' + col.b + ')', 'flood-opacity': round((col.a === undefined ? 1 : col.a) * 0.9) }) + '/></filter>';
}

async function imageHref(url) {
  try {
    mkdirSync(CACHE, { recursive: true });
    const h = createHash('sha1').update(url).digest('hex').slice(0, 16);
    for (const ext of ['.jpg', '.png', '.webp']) {
      const p = join(CACHE, h + ext);
      if (existsSync(p)) return 'data:image/' + ext.slice(1) + ';base64,' + readFileSync(p).toString('base64');
    }
    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), 8000);
    let res;
    try { res = await fetch(url, { signal: ctrl.signal }); } finally { clearTimeout(timer); }
    if (!res.ok) return null;
    const bytes = Buffer.from(await res.arrayBuffer());
    let ext = extname(new URL(url).pathname).toLowerCase() || '.jpg';
    if (ext === '.jpeg') ext = '.jpg';
    writeFileSync(join(CACHE, h + ext), bytes);
    return 'data:image/' + (ext === '.jpg' ? 'jpeg' : ext.slice(1)) + ';base64,' + bytes.toString('base64');
  } catch (e) { return null; }
}

export async function renderSvg(raw) {
  const vw = raw.vw, vh = raw.vh;
  const defs = []; const body = []; let uid = 0;
  const gradIdFor = new Map();
  for (const n of raw.nodes) {
    if (!n.bgImg) continue;
    const id = 'g' + (uid++);
    const d = gradientDefs(n.bgImg, id);
    if (d) { defs.push(d); gradIdFor.set(n, id); }
  }
  for (const n of raw.nodes) {
    const x = round(n.x), y = round(n.y), w = round(n.w), h = round(n.h);
    const op = n.opacity !== undefined ? n.opacity : null;
    const radii = radiusValues(n.radius);
    const rx = radii[0];
    const hasCorner = radii.some((r) => r > 0);
    let imgClip = null;
    if (n.img && hasCorner) { const cid = 'c' + (uid++); defs.push('<clipPath' + attrs({ id: cid }) + '><path d="' + roundedPath(x, y, w, h, radii) + '"/></clipPath>'); imgClip = 'url(#' + cid + ')'; }
    let filter = null;
    if (n.shadow) { const id = 's' + (uid++); const f = shadowFilter(n.shadow, id); if (f) { defs.push(f); filter = 'url(#' + id + ')'; } }
    let bgFill = 'none'; let bgOpacity = 1;
    const gid = gradIdFor.get(n);
    if (gid) bgFill = 'url(#' + gid + ')';
    else if (n.bg) { bgFill = paint(n.bg); bgOpacity = n.bgA === undefined ? 1 : n.bgA; }
    const borderW = n.border ? n.border.w : 0;
    const borderCol = n.border ? paint(n.border.c) : 'none';
    const borderOpacity = n.border && n.border.a !== undefined ? n.border.a : 1;
    const parts = [];
    if ((bgFill !== 'none' && bgOpacity > 0) || borderW > 0) {
      if (hasCorner) parts.push('<path' + attrs({ d: roundedPath(x, y, w, h, radii), fill: bgFill, 'fill-opacity': bgFill !== 'none' && bgOpacity < 1 ? bgOpacity : null, stroke: borderW > 0 ? borderCol : null, 'stroke-opacity': borderW > 0 ? borderOpacity : null, 'stroke-width': borderW > 0 ? borderW : null, filter: filter }) + '/>');
      else parts.push('<rect' + attrs({ x: x, y: y, width: w, height: h, fill: bgFill, 'fill-opacity': bgFill !== 'none' && bgOpacity < 1 ? bgOpacity : null, stroke: borderW > 0 ? borderCol : null, 'stroke-opacity': borderW > 0 ? borderOpacity : null, 'stroke-width': borderW > 0 ? borderW : null, filter: filter }) + '/>');
    }
    if (n.img) {
      const href = await imageHref(n.img);
      if (href) parts.push('<image' + attrs({ x: x, y: y, width: w, height: h, preserveAspectRatio: 'xMidYMid slice', href: href, 'clip-path': imgClip, filter: filter }) + '/>');
      else if (hasCorner) parts.push('<path' + attrs({ d: roundedPath(x, y, w, h, radii), fill: '#d8dee9', filter: filter }) + '/>');
      else parts.push('<rect' + attrs({ x: x, y: y, width: w, height: h, fill: '#d8dee9', filter: filter }) + '/>');
    } else if (n.svg && n.svg.outer) {
      const color = paint(n.c || '#888888');
      const outer = n.svg.outer.replace(/currentColor/g, color);
      const openMatch = outer.match(/<svg([^>]*)>/);
      const openAttrs = openMatch ? openMatch[1] : '';
      const get = (name) => { const m = openAttrs.match(new RegExp('(?:^|\\s)' + name + '="([^"]*)"')); return m ? m[1] : null; };
      const inner = outer.replace(/^[\s\S]*?<svg[^>]*>/, '').replace(/<\/svg>\s*$/, '');
      parts.push('<svg' + attrs({ x: x, y: y, width: w, height: h, viewBox: get('viewBox') || '0 0 24 24', preserveAspectRatio: 'xMidYMid meet', fill: get('fill') || 'none', stroke: get('stroke') || 'none', 'stroke-width': get('stroke-width') || null, 'stroke-linecap': get('stroke-linecap') || null, 'stroke-linejoin': get('stroke-linejoin') || null }) + '>' + inner + '</svg>');
    } else if (n.canvas || n.video) {
      parts.push('<rect' + attrs({ x: x, y: y, width: w, height: h, fill: n.bg ? paint(n.bg) : '#e2e8f0' }) + '/>');
    }
    if (n.text && n.w > 1.5 && n.h > 1.5) {
      const f = n.font || { size: 14, weight: 400, lh: 0 };
      const size = f.size || 14;
      const lh = f.lh > 0 ? f.lh : size * 1.35;
      const color = paint(n.color || n.c || '#111111');
      const colA = n.colorA === undefined ? 1 : n.colorA;
      const weight = f.weight || 400;
      const singleLine = n.input || n.h <= lh * 1.4;
      const lines = singleLine ? [n.text] : wrapText(n.text, size, weight, w);
      const anchor = f.align === 'center' ? 'middle' : f.align === 'right' ? 'end' : 'start';
      const tx = f.align === 'center' ? x + w / 2 : f.align === 'right' ? x + w : x;
      let tspans = '';
      for (let i = 0; i < lines.length; i++) {
        const ty = y + (lh - size) / 2 + size * 0.8 + i * lh;
        tspans += '<tspan' + attrs({ x: round(tx), y: round(ty) }) + '>' + esc(lines[i]) + '</tspan>';
      }
      parts.push('<text' + attrs({ 'font-family': fontFamilyFor(n.text), 'font-size': size, 'font-weight': weight, fill: color, 'fill-opacity': colA < 1 ? colA : null, 'letter-spacing': f.ls ? f.ls : null, 'text-anchor': anchor }) + '>' + tspans + '</text>');
    }
    if (!parts.length) continue;
    let wrapOpen = '', wrapClose = '';
    if (n.clip) { const cid = 'k' + (uid++); defs.push('<clipPath' + attrs({ id: cid }) + '><path d="' + roundedPath(n.clip.x, n.clip.y, n.clip.w, n.clip.h, n.clip.r) + '"/></clipPath>'); wrapOpen = '<g clip-path="url(#' + cid + ')">'; wrapClose = '</g>'; }
    body.push('<g' + attrs({ opacity: op }) + '>' + wrapOpen + parts.join('') + wrapClose + '</g>');
  }
  const clip = '<clipPath id="frame"><rect x="0" y="0" width="' + vw + '" height="' + vh + '"/></clipPath>';
  return '<?xml version="1.0" encoding="UTF-8"?>\n<svg xmlns="http://www.w3.org/2000/svg" width="' + vw + '" height="' + vh + '" viewBox="0 0 ' + vw + ' ' + vh + '"><defs>' + clip + defs.join('') + '</defs><g clip-path="url(#frame)">' + body.join('') + '</g></svg>';
}

if (process.argv[1] && process.argv[1].endsWith('render-svg.mjs')) {
  const rawPath = process.argv[2], outPath = process.argv[3];
  if (!rawPath || !outPath) { console.error('usage: render-svg.mjs <raw.json> <out.svg>'); process.exit(1); }
  const raw = JSON.parse(readFileSync(rawPath, 'utf8'));
  const svg = await renderSvg(raw);
  mkdirSync(dirname(outPath), { recursive: true });
  writeFileSync(outPath, svg);
  console.log('rendered', outPath, svg.length, 'bytes');
}

#!/usr/bin/env node
/**
 * 由设计规格（spec.json）渲染浏览器可看的预览：docs/design-kit/figma-plugin 之外的自查/评审入口。
 * 输出：.workwork/design-kit/preview/<mode>/<device>/<screen>.svg + index.html（不提交）
 * 用法：node scripts/design-kit/render-preview.mjs [--mode vibrant-light] [--device mobile] [--only home,settings]
 */
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { renderSvg } from './render-svg.mjs';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const spec = JSON.parse(readFileSync(join(ROOT, '.workwork/design-kit/spec/spec.json'), 'utf8'));
const runtime = JSON.parse(readFileSync(join(ROOT, 'docs/design-kit/tokens/runtime.json'), 'utf8'));
const args = process.argv.slice(2);
const value = (name, def) => { const i = args.indexOf('--' + name); return i >= 0 ? args[i + 1] : def; };
const mode = value('mode', 'vibrant-light');
const deviceFilter = value('device', '');
const only = value('only', '').split(',').filter(Boolean);
const OUT = join(ROOT, '.workwork/design-kit/preview');

const modeTokens = runtime.data[mode];
if (!modeTokens) throw new Error('unknown mode: ' + mode);
function tokenColor(name) {
  const entry = modeTokens['--cst-' + name];
  if (!entry || !entry.color) return null;
  const m = entry.color.match(/rgba?\(([^)]+)\)/);
  if (!m) return null;
  const p = m[1].split(/[ ,/]+/).filter(Boolean).map(Number);
  return { hex: '#' + [p[0], p[1], p[2]].map((v) => Math.round(v).toString(16).padStart(2, '0')).join(''), a: p[3] === undefined ? 1 : p[3] };
}
function resolve(name, alpha) {
  if (!name) return null;
  if (name.charAt(0) === '#') return { hex: name, a: alpha === undefined ? 1 : alpha };
  const t = tokenColor(name);
  if (!t) return { hex: '#888888', a: alpha === undefined ? 1 : alpha };
  return { hex: t.hex, a: alpha === undefined ? t.a : alpha };
}

function adapt(node) {
  const o = { x: node.x, y: node.y, w: node.w, h: node.h };
  if (node.f) { const c = resolve(node.f, node.fa); o.bg = c.hex; o.bgA = c.a; }
  if (node.g !== undefined) o.bgImg = gradientCss(spec.gradientDict[node.g]);
  if (node.bd) { const c = resolve(node.bd[1], node.bd[2]); o.border = { w: node.bd[0], c: c.hex, a: c.a }; }
  if (node.r) o.radius = node.r.map((v) => v + 'px').join(' ');
  if (node.sh !== undefined) o.shadow = shadowCss(spec.shadowDict[node.sh]);
  if (node.cl) o.clip = { x: node.cl[0], y: node.cl[1], w: node.cl[2], h: node.cl[3], r: node.cl.slice(4) };
  if (node.o !== undefined) o.opacity = node.o;
  if (node.tx !== undefined) {
    o.text = node.tx;
    o.font = { size: node.s, weight: node.fw, lh: node.lh || 0, ls: node.ls || 0, align: node.al || 'start' };
    const c = resolve(node.c, node.ca); if (c) { o.color = c.hex; o.colorA = c.a; }
    if (node.cj) o.font.family = 'PingFang SC';
  } else if (node.sv !== undefined) {
    o.svg = { outer: spec.svgDict[node.sv] };
    const c = resolve(node.c, node.ca); if (c) { o.c = c.hex; o.cA = c.a; }
  } else if (node.im !== undefined) {
    const image = spec.imageDict[node.im];
    o.img = image && image.data ? image.data : '';
    o.fit = node.fit || 'cover';
  } else if (node.t === 'r' && !node.f && !node.g && !node.bd) {
    o.bg = null;
  }
  if (node.t === 'c') o.canvas = true;
  return o;
}
function gradientCss(g) {
  const stops = g.stops.map((s) => {
    const c = resolve(s.c, s.a);
    return 'rgba(' + parseInt(c.hex.slice(1, 3), 16) + ', ' + parseInt(c.hex.slice(3, 5), 16) + ', ' + parseInt(c.hex.slice(5, 7), 16) + ', ' + c.a + ') ' + Math.round((s.p || 0) * 100) + '%';
  }).join(', ');
  return g.type === 'radial' ? 'radial-gradient(' + stops + ')' : 'linear-gradient(' + g.angle + 'deg, ' + stops + ')';
}
function shadowCss(list) {
  return list.map((s) => {
    const c = resolve(s.c, s.a);
    return 'rgba(' + parseInt(c.hex.slice(1, 3), 16) + ', ' + parseInt(c.hex.slice(3, 5), 16) + ', ' + parseInt(c.hex.slice(5, 7), 16) + ', ' + c.a + ') ' + s.x + 'px ' + s.y + 'px ' + s.blur + 'px ' + s.spread + 'px';
  }).join(', ');
}

mkdirSync(OUT, { recursive: true });
const devices = deviceFilter ? [deviceFilter] : spec.meta.devices;
const links = [];
for (const device of devices) {
  const dir = join(OUT, mode, device);
  mkdirSync(dir, { recursive: true });
  const screens = spec.screens.filter((s) => s.device === device && (!only.length || only.includes(s.id)));
  for (const screen of screens) {
    const raw = { vw: screen.w, vh: screen.h, nodes: screen.nodes.map(adapt) };
    const svg = await renderSvg(raw);
    writeFileSync(join(dir, screen.id + '.svg'), svg);
    links.push({ device, id: screen.id, title: screen.title, route: screen.route, w: screen.w, h: screen.h, file: mode + '/' + device + '/' + screen.id + '.svg' });
  }
}
const html = '<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><title>HotSpot Design Preview · ' + mode + '</title><style>body{font-family:Inter,"PingFang SC",sans-serif;background:#f5f7fa;margin:0;padding:24px;color:#16233a}h1{font-size:24px;margin:0 0 8px}p{color:#52647f;font-size:13px}.grid{display:flex;flex-wrap:wrap;gap:32px;margin-top:24px}.shot{background:#fff;border-radius:16px;padding:10px;box-shadow:0 8px 30px rgba(20,40,80,.08)}.shot img{display:block;width:100%;border-radius:10px}.shot small{display:block;margin-top:8px;color:#52647f}</style></head><body><h1>HotSpot 设计预览 · ' + mode + '</h1><p>由 spec.json 渲染；点击对应 Figma 插件页可得到可编辑图层。</p><div class="grid">' +
  links.map((l) => '<div class="shot" style="width:' + (l.device === 'desktop' ? 720 : l.device === 'tablet' ? 420 : 260) + 'px"><a href="' + l.file + '"><img src="' + l.file + '" loading="lazy" alt="' + l.title + '"></a><small>' + l.device + ' · ' + l.title + ' · ' + l.route + '</small></div>').join('') +
  '</div></body></html>';
writeFileSync(join(OUT, mode, 'index.html'), html);
console.log('preview rendered', links.length, 'screens ->', OUT + '/' + mode + '/index.html');

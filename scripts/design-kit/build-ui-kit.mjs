#!/usr/bin/env node
/**
 * 由 tokens.json 生成可交互的 UI 设计规范页 docs/design-kit/ui-kit.html。
 * 用法：node scripts/design-kit/build-ui-kit.mjs
 */
import { readFileSync, writeFileSync, readdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execSync } from 'node:child_process';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const KIT = join(ROOT, 'docs/design-kit');
const tokens = JSON.parse(readFileSync(join(KIT, 'tokens.json'), 'utf8'));
const g = tokens.global;
const THEMES = ['vibrant', 'forest', 'inkwash'];
const THEME_LABEL = { vibrant: '星际 Vibrant', forest: '清野 Forest', inkwash: '水墨 Inkwash' };
const MODES = ['light', 'dark'];
const MODE_LABEL = { light: '浅色', dark: '深色' };

function esc(s) {
	return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
let commit = 'dev';
try {
	commit = execSync('git rev-parse --short HEAD', { cwd: ROOT }).toString().trim();
} catch (e) {}

function themeVars(theme, mode) {
	const set = tokens.themes[theme][mode];
	const vars = [];
	for (const [key, token] of Object.entries(set.color.semantic)) vars.push('  --k-' + key + ': ' + token.$value + ';');
	for (const [key, token] of Object.entries(set.color.palette)) vars.push('  --k-palette-' + key + ': ' + token.$value + ';');
	return '[data-theme="' + theme + '"][data-mode="' + mode + '"] {\n' + vars.join('\n') + '\n}';
}

function swatch(name, value) {
	const isColor = /^#|^rgb|^hsl/.test(String(value).trim());
	const style = isColor ? 'background:' + value + ';' : 'background:repeating-linear-gradient(45deg,#ddd,#ddd 6px,#fff 6px,#fff 12px)';
	return '<div class="swatch"><span class="chip" style="' + style + '"></span><code>' + esc(name) + '</code><small>' + esc(value) + '</small></div>';
}

function swatchGroup(theme, mode) {
	const set = tokens.themes[theme][mode];
	const palette = Object.entries(set.color.palette).map(([k, t]) => swatch(k, t.$value)).join('');
	const semantic = Object.entries(set.color.semantic).map(([k, t]) => swatch(k, t.$value)).join('');
	return '<section class="swatch-theme"><h4>' + THEME_LABEL[theme] + ' · ' + MODE_LABEL[mode] + '</h4>' +
		'<p class="muted">原始色板 palette</p><div class="swatches">' + palette + '</div>' +
		'<p class="muted">语义色 semantic</p><div class="swatches">' + semantic + '</div></section>';
}

function fontRow(name, px) {
	return '<div class="type-row"><code>text-' + esc(name) + '</code><span class="type-sample" style="font-size:' + px + '">HotSpot 城市此刻</span><small>' + px + '</small></div>';
}
function spacingRow(name, px) {
	const width = Math.max(2, Number(String(px).replace('px', '')) * 4);
	return '<div class="space-row"><code>' + esc(name) + '</code><span class="space-bar" style="width:' + width + 'px"></span><small>' + esc(px) + '</small></div>';
}
function radiusRow(name, px) {
	return '<div class="radius-row"><span class="radius-box" style="border-radius:' + px + '"></span><code>' + esc(name) + '</code><small>' + esc(px) + '</small></div>';
}

const typeRows = Object.entries(g.typography.fontSize).map(([k, t]) => fontRow(k, t.$value)).join('');
const spacingRows = Object.entries(g.spacing).map(([k, t]) => spacingRow(k, t.$value)).join('');
const radiusRows = Object.entries(g.radius).map(([k, t]) => radiusRow(k, t.$value)).join('');
const borderRows = Object.entries(g.borderWidth).map(([k, t]) => '<div class="space-row"><code>' + k + '</code><span style="flex:0 0 auto;border-top:' + t.$value + ' solid var(--k-text-primary);width:120px"></span><small>' + t.$value + '</small></div>').join('');
const opacityRows = Object.entries(g.opacity).map(([k, t]) => '<div class="space-row"><code>' + k + '</code><span class="space-bar" style="width:' + Number(t.$value) * 160 + 'px;opacity:' + t.$value + ';background:var(--k-accent-default)"></span><small>' + t.$value + '</small></div>').join('');
const zRows = Object.entries(g.zIndex).map(([k, t]) => '<div class="space-row"><code>z-' + k + '</code><small>' + t.$value + '</small></div>').join('');
const blurRows = Object.entries(g.blur).map(([k, t]) => '<div class="space-row"><code>blur-' + k + '</code><small>' + t.$value + '</small></div>').join('');
const motionRows = Object.entries(g.motion).map(([k, t]) => '<div class="space-row"><code>' + k + '</code><small>' + esc(t.$value) + '</small></div>').join('');

const paletteSections = [];
for (const theme of THEMES) for (const mode of MODES) paletteSections.push(swatchGroup(theme, mode));

const harmony = g.harmonyReference;
const harmonyRows = Object.entries(harmony.fontSize).map(([k, v]) => '<tr><td>FONT.SIZE.' + k + '</td><td>' + v + '</td></tr>').join('');
const harmonySpacing = Object.entries(harmony.spacing).map(([k, v]) => '<tr><td>SPACING.' + k + '</td><td>' + v + '</td></tr>').join('');
const harmonyRadius = Object.entries(harmony.radius).map(([k, v]) => '<tr><td>RADIUS.' + k + '</td><td>' + v + '</td></tr>').join('');

const pagesDir = join(KIT, 'pages');
// 截图不入库：由 scripts/design-kit/capture-pages.sh 生成到 .workwork/design-kit/pages，
// 上传到 Figma/Stitch 后即可删除本地副本。
const files = (() => {
	try {
		return readdirSync(pagesDir).filter((f) => /\.(png|jpg)$/.test(f)).sort();
	} catch (e) {
		return [];
	}
})();
function gallery(prefix, title, note) {
	const items = files.filter((f) => f.startsWith(prefix)).map((f) => {
		const label = f.replace(/^(web|app)-/, '').replace(/\.(png|jpg)$/, '');
		return '<figure><a href="pages/' + f + '" target="_blank"><img src="pages/' + f + '" loading="lazy" alt="' + esc(label) + '"></a><figcaption>' + esc(label) + '</figcaption></figure>';
	});
	if (items.length === 0) {
		return '<h3>' + esc(title) + '</h3><p class="muted">' + esc(note) +
			'</p><div class="note">仓库不存页面截图。运行 <code>bash scripts/design-kit/capture-pages.sh</code> 生成到 <code>.workwork/design-kit/pages</code>，再把截图上传到 Figma / Stitch 管理。</div>';
	}
	return '<h3>' + esc(title) + '</h3><p class="muted">' + esc(note) + '</p><div class="gallery">' + items.join('') + '</div>';
}
const webGallery = gallery('web-', 'Web 端页面（390 × 844）', '实时页面截图，深色主题；与 docs/design-kit/pages 同源。');
const appGallery = gallery('app-', 'HarmonyOS 端页面（模拟器/真机）', '浅色与深色混合，覆盖同一 IA 的关键页面。');

const CSS = [
':root { color-scheme: light; }',
'* { box-sizing: border-box; }',
'body { margin: 0; font-family: Inter, -apple-system, "PingFang SC", "Noto Sans SC", sans-serif; background: var(--k-surface-page); color: var(--k-text-primary); }',
'header.kit-header { padding: 40px 32px 24px; border-bottom: 1px solid var(--k-border-default); }',
'header.kit-header h1 { margin: 0 0 8px; font-size: 30px; letter-spacing: -0.01em; }',
'header.kit-header p { margin: 0; color: var(--k-text-secondary); max-width: 860px; line-height: 1.6; }',
'.badges { display: flex; gap: 8px; flex-wrap: wrap; margin-top: 16px; }',
'.badge { padding: 4px 10px; border-radius: 999px; background: var(--k-accent-soft); color: var(--k-accent-default); font-size: 11px; font-weight: 600; }',
'.toolbar { position: sticky; top: 0; z-index: 50; display: flex; gap: 12px; align-items: center; padding: 12px 32px; background: color-mix(in srgb, var(--k-surface-primary) 88%, transparent); backdrop-filter: blur(16px); border-bottom: 1px solid var(--k-border-default); }',
'.toolbar button { border: 1px solid var(--k-border-default); background: var(--k-surface-primary); color: var(--k-text-primary); padding: 6px 14px; border-radius: 999px; font-size: 12px; cursor: pointer; }',
'.toolbar button[aria-pressed="true"] { background: var(--k-accent-default); color: var(--k-on-accent); border-color: transparent; }',
'main { max-width: 1180px; margin: 0 auto; padding: 32px; }',
'section.kit-section { margin: 0 0 56px; }',
'section.kit-section > h2 { font-size: 20px; margin: 0 0 6px; }',
'section.kit-section > p.lead { margin: 0 0 20px; color: var(--k-text-secondary); line-height: 1.6; }',
'.muted { color: var(--k-text-secondary); font-size: 12px; margin: 10px 0 6px; }',
'code { font-family: "SF Mono", ui-monospace, monospace; font-size: 11px; background: var(--k-accent-soft); color: var(--k-accent-default); padding: 1px 6px; border-radius: 6px; }',
'.swatch-theme { margin: 0 0 28px; }',
'.swatch-theme h4 { margin: 0 0 4px; font-size: 14px; }',
'.swatches { display: grid; grid-template-columns: repeat(auto-fill, minmax(190px, 1fr)); gap: 8px; }',
'.swatch { display: grid; grid-template-columns: 28px 1fr; grid-template-rows: auto auto; column-gap: 8px; align-items: center; padding: 6px; border-radius: 10px; background: var(--k-surface-primary); border: 1px solid var(--k-border-default); }',
'.swatch .chip { grid-row: span 2; width: 28px; height: 28px; border-radius: 8px; border: 1px solid var(--k-border-default); }',
'.swatch code { background: transparent; padding: 0; color: var(--k-text-primary); }',
'.swatch small { color: var(--k-text-secondary); font-size: 10px; word-break: break-all; }',
'.type-row, .space-row, .radius-row { display: flex; align-items: center; gap: 14px; padding: 6px 0; border-bottom: 1px dashed var(--k-border-default); }',
'.type-row small, .space-row small, .radius-row small { margin-left: auto; color: var(--k-text-secondary); }',
'.type-sample { flex: 1; }',
'.radius-box { width: 44px; height: 44px; background: var(--k-accent-soft); border: 1px solid var(--k-accent-border); }',
'.components { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px; }',
'.demo-card { background: var(--k-surface-primary); border: 1px solid var(--k-border-default); border-radius: 20px; padding: 16px; box-shadow: var(--k-elevation-card); }',
'.demo-card h5 { margin: 0 0 12px; font-size: 12px; color: var(--k-text-secondary); font-weight: 600; }',
'.btn { border: 0; border-radius: 999px; padding: 10px 18px; font-size: 13px; font-weight: 600; cursor: pointer; display: inline-flex; align-items: center; gap: 6px; }',
'.btn-primary { background: var(--k-accent-default); color: var(--k-on-accent); }',
'.btn-secondary { background: var(--k-surface-primary); color: var(--k-text-primary); border: 1px solid var(--k-border-default); }',
'.btn-soft { background: var(--k-accent-soft); color: var(--k-accent-default); }',
'.btn-ghost { background: transparent; color: var(--k-text-secondary); }',
'.btn-danger { background: var(--k-status-danger); color: #fff; }',
'.btn[disabled] { opacity: 0.45; cursor: not-allowed; }',
'.btn-sm { padding: 6px 12px; font-size: 12px; }',
'.btn-lg { padding: 12px 22px; font-size: 14px; }',
'.field { display: grid; gap: 6px; margin-bottom: 10px; }',
'.field span { font-size: 12px; color: var(--k-text-secondary); }',
'.input { border: 1px solid var(--k-field-border, var(--k-border-default)); background: var(--k-field-bg, var(--k-surface-primary)); color: var(--k-text-primary); border-radius: 16px; padding: 10px 14px; font-size: 13px; outline: none; }',
'.input:focus { border-color: var(--k-field-border-focus, var(--k-accent-default)); box-shadow: 0 0 0 3px var(--k-accent-soft); }',
'.input-error { border-color: var(--k-status-danger); }',
'.pills { display: flex; gap: 6px; flex-wrap: wrap; align-items: center; }',
'.pill { padding: 6px 14px; border-radius: 999px; font-size: 12px; font-weight: 600; background: var(--k-surface-primary); border: 1px solid var(--k-border-default); color: var(--k-text-secondary); }',
'.pill-active { background: var(--k-accent-default); color: var(--k-on-accent); border-color: transparent; }',
'.pill-soft { background: var(--k-accent-soft); color: var(--k-accent-default); border-color: transparent; }',
'.seg { display: inline-flex; background: var(--k-surface-primary); border: 1px solid var(--k-border-default); border-radius: 999px; padding: 4px; gap: 4px; }',
'.seg button { border: 0; background: transparent; color: var(--k-text-secondary); padding: 7px 16px; border-radius: 999px; font-size: 12px; font-weight: 600; cursor: pointer; }',
'.seg button[aria-pressed="true"] { background: var(--k-accent-default); color: var(--k-on-accent); }',
'.underline { display: flex; gap: 20px; border-bottom: 1px solid var(--k-border-default); }',
'.underline button { border: 0; background: transparent; padding: 8px 2px; font-size: 13px; color: var(--k-text-secondary); position: relative; cursor: pointer; }',
'.underline button[aria-pressed="true"] { color: var(--k-text-primary); font-weight: 600; }',
'.underline button[aria-pressed="true"]::after { content: ""; position: absolute; left: 0; right: 0; bottom: -1px; height: 2px; background: var(--k-accent-default); }',
'.badge-accent { background: var(--k-accent-soft); color: var(--k-accent-default); }',
'.badge-online { background: color-mix(in srgb, var(--k-status-online) 14%, transparent); color: var(--k-status-online); }',
'.badge-warning { background: color-mix(in srgb, var(--k-status-warning) 14%, transparent); color: var(--k-status-warning); }',
'.badge-danger { background: color-mix(in srgb, var(--k-status-danger) 14%, transparent); color: var(--k-status-danger); }',
'.row { display: flex; align-items: center; gap: 10px; padding: 10px 12px; border-radius: 14px; }',
'.row:hover { background: var(--k-accent-soft); }',
'.avatar { width: 40px; height: 40px; border-radius: 999px; background: var(--k-accent-soft); color: var(--k-accent-default); display: grid; place-items: center; font-weight: 600; }',
'.dot { width: 10px; height: 10px; border-radius: 999px; background: var(--k-status-online); border: 2px solid var(--k-surface-primary); }',
'.card-hero { background: linear-gradient(135deg, var(--k-accent-soft), transparent); border: 1px solid var(--k-border-default); border-radius: 24px; padding: 18px; }',
'.sheet { background: var(--k-surface-panel); border-radius: 28px 28px 0 0; padding: 16px; box-shadow: 0 -8px 30px rgba(0,0,0,0.25); }',
'.toast { background: var(--k-surface-panel); color: var(--k-text-primary); border: 1px solid var(--k-border-default); border-radius: 16px; padding: 12px 16px; box-shadow: var(--k-elevation-card); }',
'.empty { text-align: center; padding: 24px; color: var(--k-text-secondary); }',
'.skeleton { height: 12px; border-radius: 6px; background: linear-gradient(90deg, var(--k-accent-soft), var(--k-surface-primary), var(--k-accent-soft)); background-size: 200% 100%; animation: shimmer 1.6s linear infinite; }',
'@keyframes shimmer { from { background-position: 200% 0; } to { background-position: -200% 0; } }',
'.gallery { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 16px; }',
'.gallery figure { margin: 0; }',
'.gallery img { width: 100%; border-radius: 16px; border: 1px solid var(--k-border-default); box-shadow: var(--k-elevation-card); display: block; }',
'.gallery figcaption { font-size: 11px; color: var(--k-text-secondary); margin-top: 6px; }',
'table { width: 100%; border-collapse: collapse; font-size: 12px; }',
'th, td { text-align: left; padding: 6px 8px; border-bottom: 1px solid var(--k-border-default); }',
'th { color: var(--k-text-secondary); font-weight: 600; }',
'.cols { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 16px; }',
'.note { background: var(--k-accent-soft); border: 1px solid var(--k-accent-border); border-radius: 16px; padding: 14px 16px; font-size: 12px; line-height: 1.7; }',
'footer { padding: 24px 32px 48px; color: var(--k-text-secondary); font-size: 11px; border-top: 1px solid var(--k-border-default); }'
];

const BODY = [
'<header class="kit-header">',
'  <h1>HotSpot UI Design Kit</h1>',
'  <p>Web（SvelteKit + Tailwind）与 HarmonyOS（ArkTS）共用的设计规范单一事实源。Token 由 <code>frontend/src/styles/theme</code> 与 <code>harmonyapp/.../DesignConstants.ets</code> 生成，页面截图取自本地实时运行环境；后续迭代请修改代码 token 后重新生成本页。</p>',
'  <div class="badges"><span class="badge">3 主题 × 明暗</span><span class="badge">9 档字号</span><span class="badge">5 档圆角</span><span class="badge">Web + HarmonyOS 双端</span><span class="badge">commit ' + commit + '</span></div>',
'</header>',
'<div class="toolbar">',
'  <strong style="font-size:12px">预览主题</strong>',
'  <button data-set-theme="vibrant" aria-pressed="true">星际</button>',
'  <button data-set-theme="forest" aria-pressed="false">清野</button>',
'  <button data-set-theme="inkwash" aria-pressed="false">水墨</button>',
'  <span style="width:1px;height:20px;background:var(--k-border-default)"></span>',
'  <button data-set-mode="light" aria-pressed="true">浅色</button>',
'  <button data-set-mode="dark" aria-pressed="false">深色</button>',
'</div>',
'<main>',
'<section class="kit-section"><h2>色彩 Colors</h2>',
'<p class="lead">三层 token：原始色板 palette → 语义 semantic → 组件消费。所有页面只消费语义色；切换主题/明暗时语义色整体替换，组件不需要任何改动。</p>',
paletteSections.join(''),
'</section>',
'<section class="kit-section"><h2>字体 Typography</h2>',
'<p class="lead">字号少档位收敛（禁止 1–2px 近邻档并存）：辅助 11、正文 12、强调 14、标题 18+。字重 400/500/600/700。</p>',
typeRows,
'<div class="cols" style="margin-top:16px">',
'<table><thead><tr><th>字重</th><th>值</th></tr></thead><tbody><tr><td>regular</td><td>400</td></tr><tr><td>medium</td><td>500</td></tr><tr><td>semibold</td><td>600</td></tr><tr><td>bold</td><td>700</td></tr></tbody></table>',
'<table><thead><tr><th>行高比例</th><th>值</th></tr></thead><tbody><tr><td>DEFAULT</td><td>1.5</td></tr><tr><td>CARD</td><td>1.7</td></tr><tr><td>COPY</td><td>1.75</td></tr></tbody></table>',
'<table><thead><tr><th>字距</th><th>值</th></tr></thead><tbody><tr><td>EYEBROW</td><td>1.4</td></tr><tr><td>LABEL</td><td>1.2</td></tr><tr><td>FIELD</td><td>1.28</td></tr><tr><td>TITLE</td><td>-0.12</td></tr></tbody></table>',
'</div></section>',
'<section class="kit-section"><h2>圆角 / 间距 / 边框 Radius / Spacing / Border</h2>',
'<p class="lead">圆角 4 档 + 全圆：12 / 16 / 20 / 24 / 9999（sheet 28）。间距 4px 网格（2/4/6/8/10/12/14/16/18/20/24/32）。</p>',
'<div class="cols"><div>' + radiusRows + '</div><div>' + spacingRows + '</div><div>' + borderRows + '</div></div></section>',
'<section class="kit-section"><h2>阴影 / 透明度 / 动效 / 模糊 / 层级</h2>',
'<div class="cols"><div>' + opacityRows + '</div><div>' + motionRows + blurRows + '</div><div>' + zRows + '</div></div></section>',
'<section class="kit-section"><h2>组件 Components</h2>',
'<p class="lead">以下组件全部消费当前预览主题的语义 token，切换上方主题即可验证暗色/主题适配。</p>',
'<div class="components">',
'<div class="demo-card"><h5>按钮 Buttons</h5><div class="pills"><button class="btn btn-primary">主按钮</button><button class="btn btn-secondary">次按钮</button><button class="btn btn-soft">柔和</button><button class="btn btn-ghost">幽灵</button><button class="btn btn-danger">危险</button></div>',
'<div class="pills" style="margin-top:12px"><button class="btn btn-primary btn-sm">小号</button><button class="btn btn-primary">中号</button><button class="btn btn-primary btn-lg">大号</button><button class="btn btn-primary" disabled>禁用</button></div></div>',
'<div class="demo-card"><h5>输入 Inputs</h5><label class="field"><span>默认</span><input class="input" value="输入内容"></label><label class="field"><span>错误</span><input class="input input-error" value="格式不正确"></label><label class="field"><span>禁用</span><input class="input" value="不可编辑" disabled></label></div>',
'<div class="demo-card"><h5>徽章 / 胶囊 Badges</h5><div class="pills"><span class="badge badge-accent">accent</span><span class="badge badge-online">在线</span><span class="badge badge-warning">提醒</span><span class="badge badge-danger">危险</span><span class="badge">neutral</span></div>',
'<div class="pills" style="margin-top:12px"><span class="pill">全部</span><span class="pill pill-active">已选</span><span class="pill pill-soft">筛选</span><span class="pill">#云龙湖</span><span class="pill">#摄影</span></div></div>',
'<div class="demo-card"><h5>Tabs</h5><div class="pills"><button class="pill pill-active">胶囊选中</button><button class="pill">胶囊</button></div><div class="seg" style="margin-top:12px"><button aria-pressed="true">为你推荐</button><button aria-pressed="false">正在关注</button></div><div class="underline" style="margin-top:12px"><button aria-pressed="true">简介</button><button aria-pressed="false">成员</button><button aria-pressed="false">公告</button></div></div>',
'<div class="demo-card"><h5>列表行 / 头像 List &amp; Avatar</h5><div class="row"><span class="avatar">鹿</span><div style="flex:1"><div style="font-size:13px;font-weight:600">鹿鸣</div><div style="font-size:11px;color:var(--k-text-secondary)">@user_seed_lumos · 济源市南天门</div></div><span class="dot"></span></div><div class="row"><span class="avatar">阿</span><div style="flex:1"><div style="font-size:13px;font-weight:600">阿禾</div><div style="font-size:11px;color:var(--k-text-secondary)">记录城市光影</div></div><span class="badge badge-accent">关注</span></div></div>',
'<div class="demo-card"><h5>卡片 / Hero Card</h5><div class="card-hero"><div style="font-size:11px;letter-spacing:1.2px;color:var(--k-text-secondary)">附近</div><div style="font-size:20px;font-weight:700;margin:6px 0">实时附近</div><div style="display:flex;gap:18px"><div><b style="font-size:22px">8</b><div style="font-size:11px;color:var(--k-text-secondary)">人</div></div><div><b style="font-size:22px">6</b><div style="font-size:11px;color:var(--k-text-secondary)">瞬间</div></div><div><b style="font-size:22px">2</b><div style="font-size:11px;color:var(--k-text-secondary)">在线</div></div></div></div></div>',
'<div class="demo-card"><h5>底部弹层 / 对话框 / Toast</h5><div class="sheet"><div style="width:36px;height:4px;border-radius:999px;background:var(--k-border-default);margin:0 auto 12px"></div><div style="font-size:15px;font-weight:700">封禁并移出</div><div style="font-size:12px;color:var(--k-text-secondary);margin:6px 0 12px">确定将「游客」封禁并移出群组？封禁后对方将无法再加入。</div><div class="pills"><button class="btn btn-secondary" style="flex:1">取消</button><button class="btn btn-danger" style="flex:1">封禁并移出</button></div></div><div class="toast" style="margin-top:12px">「用户管理」请在 Web 管理后台完成</div></div>',
'<div class="demo-card"><h5>空态 / 骨架</h5><div class="empty"><div style="font-size:28px">☁️</div><div style="font-size:13px;font-weight:600;margin-top:6px">暂无内容</div><div style="font-size:11px">换个筛选条件试试</div></div><div class="skeleton" style="width:80%;margin:10px auto"></div><div class="skeleton" style="width:60%;margin:6px auto"></div></div>',
'</div></section>',
'<section class="kit-section"><h2>页面 Pages</h2>',
'<p class="lead">页面截图为实时环境捕获（Web 390×844、HarmonyOS 模拟器/真机）。它们用于对齐布局、间距、层级与组件组合，不作为像素级唯一依据——像素级以 token 与代码为准。</p>',
webGallery,
appGallery,
'</section>',
'<section class="kit-section"><h2>跨端对齐 Parity</h2>',
'<p class="lead">Web 字号/圆角/间距与 HarmonyOS DesignConstants 一一对应；改任一端需同步另一端并跑校验脚本（见 README）。</p>',
'<div class="cols"><table><thead><tr><th>HarmonyOS 字号</th><th>vp</th></tr></thead><tbody>' + harmonyRows + '</tbody></table>',
'<table><thead><tr><th>HarmonyOS 圆角</th><th>vp</th></tr></thead><tbody>' + harmonyRadius + '</tbody></table>',
'<table><thead><tr><th>HarmonyOS 间距</th><th>vp</th></tr></thead><tbody>' + harmonySpacing + '</tbody></table></div></section>',
'<section class="kit-section"><h2>导入 Figma / Stitch 与迭代流程</h2>',
'<div class="note"><p><b>Figma</b>：① 新建文件；② 安装 Tokens Studio 插件并导入 <code>tokens/global.json</code> 与 <code>tokens/themes/*.json</code>（一个主题一个 set）；③ 把 <code>pages/*.png|jpg</code> 拖入画布作为参考层（或逐页 Place image）；④ 用 Components 区截图/说明建立 Figma 组件库（按钮/输入/列表/Tab/弹层）。</p>',
'<p><b>Stitch</b>：在 Stitch 新建项目，上传 <code>pages/</code> 中的截图作为参考图，并粘贴 README 的「设计约束」段落作为生成提示；后续迭代以本页 token 与截图为准，Stitch 产出需回填到代码 token 后重新生成本页。</p>',
'<p><b>迭代</b>：改代码 token（frontend themes / semantic / shared 或 harmonyapp DesignConstants）→ 重新运行 <code>node scripts/design-kit/build-tokens.mjs</code> 与 <code>node scripts/design-kit/build-ui-kit.mjs</code> → 重新截图 → 提交。校验：<code>bun run check:theme</code>、<code>python3 harmonyapp/scripts/check_status_parity.py</code>。</p></div></section>',
'</main>',
'<footer>HotSpot UI Design Kit · generated from commit ' + commit + ' · tokens 由代码生成，勿手工编辑 tokens.json / ui-kit.html。</footer>',
'<script>',
'(function () {',
'  var root = document.documentElement;',
'  root.dataset.theme = "vibrant";',
'  root.dataset.mode = "light";',
'  function bind(group, attr) {',
'    var buttons = document.querySelectorAll("[data-set-" + group + "]");',
'    buttons.forEach(function (btn) {',
'      btn.addEventListener("click", function () {',
'        root.dataset[attr] = btn.getAttribute("data-set-" + group);',
'        buttons.forEach(function (b) { b.setAttribute("aria-pressed", String(b === btn)); });',
'      });',
'    });',
'  }',
'  bind("theme", "theme");',
'  bind("mode", "mode");',
'})();',
'</script>'
];

const html = '<!doctype html>\n<html lang="zh-CN" data-theme="vibrant" data-mode="light">\n<head>\n<meta charset="utf-8">\n<meta name="viewport" content="width=device-width, initial-scale=1">\n<title>HotSpot UI Design Kit</title>\n<style>\n' +
	THEMES.flatMap((t) => MODES.map((m) => themeVars(t, m))).join('\n') + '\n' + CSS.join('\n') +
	'\n</style>\n</head>\n<body>\n' + BODY.join('\n') + '\n</body>\n</html>\n';

writeFileSync(join(KIT, 'ui-kit.html'), html);
console.log('ui-kit.html generated:', html.length, 'bytes,', files.length, 'page shots');


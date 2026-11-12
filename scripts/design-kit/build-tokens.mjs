#!/usr/bin/env node
/**
 * 生成跨端 UI 设计 token（Web CSS ↔ HarmonyOS DesignConstants）。
 * 输出：
 *   docs/design-kit/tokens/global.json
 *   docs/design-kit/tokens/themes/<theme>-<mode>.json
 *   docs/design-kit/tokens.json
 * 用法：node scripts/design-kit/build-tokens.mjs
 */
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const THEME_DIR = join(ROOT, 'frontend/src/styles/theme');
const OUT_DIR = join(ROOT, 'docs/design-kit');
const THEMES = ['vibrant', 'forest', 'inkwash'];

function read(file) {
	return readFileSync(join(THEME_DIR, file), 'utf8');
}

/** 解析 CSS 中所有「selector { --cst-*: value; }」块。 */
function parseBlocks(css) {
	const blocks = [];
	const re = /([^{}]+){([^{}]*)}/g;
	let m;
	while ((m = re.exec(css))) {
		const selector = m[1].replace(/\/\*[\s\S]*?\*\//g, '').trim();
		const body = m[2];
		const decls = {};
		const dre = /--cst-([a-z0-9-]+)\s*:\s*([^;]+);/g;
		let d;
		while ((d = dre.exec(body))) decls['--cst-' + d[1]] = d[2].trim();
		if (Object.keys(decls).length) blocks.push({ selector, decls });
	}
	return blocks;
}

function pickThemeVars(theme) {
	const css = read('themes/' + theme + '.css');
	const blocks = parseBlocks(css);
	const light = {};
	const dark = {};
	for (const { selector, decls } of blocks) {
		if (!selector.includes('.' + theme)) continue;
		if (selector.includes('.dark')) Object.assign(dark, decls);
		else Object.assign(light, decls);
	}
	return { light, dark };
}

/** 解析 semantic.css 的 :root 别名映射。 */
function parseSemantic() {
	const blocks = parseBlocks(read('semantic.css'));
	const map = {};
	for (const { selector, decls } of blocks) {
		if (!selector.includes(':root')) continue;
		Object.assign(map, decls);
	}
	return map;
}

/** 解析 shared.css 的 shape / border / motion / blur / z-index。 */
function parseShared() {
	const blocks = parseBlocks(read('shared.css'));
	const map = {};
	for (const { decls } of blocks) Object.assign(map, decls);
	return map;
}

/** 解析前端字号刻度（rem → px，基准 16）。 */
function parseTypeScale() {
	const css = read('theme-registry.css');
	const sizes = {};
	const re = /--text-([a-z0-9]+)\s*:\s*([0-9.]+)rem/g;
	let m;
	while ((m = re.exec(css))) sizes[m[1]] = Math.round(parseFloat(m[2]) * 16 * 100) / 100;
	return sizes;
}

/** 解析 harmonyapp FONT.SIZE / SPACING / RADIUS 常量，用于跨端对照。 */
function parseHarmony() {
	const ets = readFileSync(join(ROOT, 'harmonyapp/entry/src/main/ets/constants/DesignConstants.ets'), 'utf8');
	const fontBlock = ets.match(/SIZE:\s*\{([\s\S]*?)\}/);
	const font = {};
	if (fontBlock) {
		const re = /([A-Z0-9_]+):\s*([0-9.]+)/g;
		let m;
		while ((m = re.exec(fontBlock[1]))) font[m[1]] = Number(m[2]);
	}
	const spacingBlock = ets.match(/const SPACING:[\s\S]*?=\s*\{([\s\S]*?)\};/);
	const spacing = {};
	if (spacingBlock) {
		const re = /([A-Z0-9_]+):\s*([0-9.]+)/g;
		let m;
		while ((m = re.exec(spacingBlock[1]))) spacing[m[1]] = Number(m[2]);
	}
	const radiusBlock = ets.match(/const RADIUS:\s*RadiusSystem\s*=\s*\{([\s\S]*?)\};/);
	const radius = {};
	if (radiusBlock) {
		const re = /([A-Z]+):\s*([0-9.]+)/g;
		let m;
		while ((m = re.exec(radiusBlock[1]))) radius[m[1]] = Number(m[2]);
	}
	return { font, spacing, radius };
}

const semantic = parseSemantic();
const shared = parseShared();
const typeScale = parseTypeScale();
// theme-registry 只显式定义 xxs~md（压档收敛），lg 及以上沿用 Tailwind 默认（18/20/24/30/36），
// 与鸿蒙 FONT.SIZE 的 LG_PLUS/XL/XXL/XXXL/HERO 一一对应，这里补齐成完整刻度。
const TAILWIND_SIZES = { lg: 18, xl: 20, '2xl': 24, '3xl': 30, '4xl': 36 };
for (const [k, v] of Object.entries(TAILWIND_SIZES)) if (!typeScale[k]) typeScale[k] = v;
const harmony = parseHarmony();

/** 把 var(--cst-palette-x) / var(--cst-x) 解析为最终值（最多 6 层）。 */
function resolve(value, vars, depth = 0) {
	if (depth > 6) return value;
	const m = value.match(/^var\((--cst-[a-z0-9-]+)\)$/);
	if (!m) return value;
	const next = vars[m[1]];
	if (next === undefined) return value;
	return resolve(next, vars, depth + 1);
}

function toHex(value) {
	return value;
}

function buildThemeTokens(theme, mode) {
	const vars = { ...semantic, ...shared, ...pickThemeVars(theme)[mode] };
	const palette = {};
	const colors = {};
	for (const [key, raw] of Object.entries(vars)) {
		if (!key.startsWith('--cst-palette-')) continue;
		palette[key.replace('--cst-palette-', '')] = toHex(resolve(raw, vars));
	}
	for (const [key, raw] of Object.entries(vars)) {
		if (!key.startsWith('--cst-')) continue;
		if (key.startsWith('--cst-palette-')) continue;
		if (!/(color|bg|surface|text|border|accent|status|navigation|content|glow|scrim|overlay|shadow)/.test(key)) continue;
		const resolved = resolve(raw, vars);
		if (!resolved.includes('var(')) colors[key.replace('--cst-', '')] = resolved;
	}
	return { palette, colors };
}

const globalTokens = {
	$description: 'HotSpot 跨端设计 token（Web CSS 为源，HarmonyOS DesignConstants 已对齐）',
	typography: {
		fontFamily: {
			sans: { $type: 'fontFamily', $value: 'Inter, -apple-system, PingFang SC, Noto Sans SC, sans-serif' }
		},
		fontSize: Object.fromEntries(
			Object.entries(typeScale).map(([k, v]) => [k, { $type: 'dimension', $value: v + 'px' }])
		),
		fontWeight: {
			regular: { $type: 'fontWeight', $value: 400 },
			medium: { $type: 'fontWeight', $value: 500 },
			semibold: { $type: 'fontWeight', $value: 600 },
			bold: { $type: 'fontWeight', $value: 700 }
		},
		lineHeight: Object.fromEntries(
			Object.entries(harmony.font)
				.filter(([k]) => ['XS', 'SM', 'MD', 'LG', 'XL'].includes(k))
				.map(([k, v]) => [k.toLowerCase(), { $type: 'dimension', $value: v + 'px' }])
		)
	},
	radius: Object.fromEntries(
		Object.entries(harmony.radius).map(([k, v]) => [k.toLowerCase(), { $type: 'dimension', $value: v + 'px' }])
	),
	spacing: Object.fromEntries(
		Object.entries(harmony.spacing).map(([k, v]) => [k.toLowerCase(), { $type: 'dimension', $value: v + 'px' }])
	),
	borderWidth: {
		hairline: { $type: 'dimension', $value: shared['--cst-border-width-hairline'] || '1px' },
		regular: { $type: 'dimension', $value: shared['--cst-border-width-regular'] || '1.5px' },
		strong: { $type: 'dimension', $value: shared['--cst-border-width-strong'] || '2px' },
		heavy: { $type: 'dimension', $value: shared['--cst-border-width-heavy'] || '3px' }
	},
	opacity: Object.fromEntries(
		Object.entries(shared)
			.filter(([k]) => k.startsWith('--cst-opacity-'))
			.map(([k, v]) => [k.replace('--cst-opacity-', ''), { $type: 'number', $value: Number(v) }])
	),
	motion: Object.fromEntries(
		Object.entries(shared)
			.filter(([k]) => k.startsWith('--cst-motion-') || k.startsWith('--cst-ease-'))
			.map(([k, v]) => [k.replace('--cst-', ''), { $type: 'string', $value: v }])
	),
	blur: Object.fromEntries(
		Object.entries(shared)
			.filter(([k]) => k.startsWith('--cst-blur-'))
			.map(([k, v]) => [k.replace('--cst-blur-', ''), { $type: 'dimension', $value: v }])
	),
	zIndex: Object.fromEntries(
		Object.entries(shared)
			.filter(([k]) => k.startsWith('--cst-z-'))
			.map(([k, v]) => [k.replace('--cst-z-', ''), { $type: 'number', $value: Number(v) }])
	),
	harmonyReference: {
		$description: 'HarmonyOS DesignConstants 对照（vp/fp），用于跨端对齐',
		fontSize: harmony.font,
		spacing: harmony.spacing,
		radius: harmony.radius
	}
};

mkdirSync(join(OUT_DIR, 'tokens', 'themes'), { recursive: true });
writeFileSync(join(OUT_DIR, 'tokens', 'global.json'), JSON.stringify(globalTokens, null, 2) + '\n');

const combined = { global: globalTokens, themes: {} };
for (const theme of THEMES) {
	combined.themes[theme] = {};
	for (const mode of ['light', 'dark']) {
		const { palette, colors } = buildThemeTokens(theme, mode);
		const set = {
			$description: theme + ' / ' + mode + '（Web --cst-* 解析值）',
			color: {
				palette: Object.fromEntries(
					Object.entries(palette).map(([k, v]) => [k, { $type: 'color', $value: v }])
				),
				semantic: Object.fromEntries(
					Object.entries(colors).map(([k, v]) => [k.replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, ''), { $type: 'color', $value: v }])
				)
			}
		};
		combined.themes[theme][mode] = set;
		writeFileSync(
			join(OUT_DIR, 'tokens', 'themes', theme + '-' + mode + '.json'),
			JSON.stringify(set, null, 2) + '\n'
		);
	}
}
writeFileSync(join(OUT_DIR, 'tokens.json'), JSON.stringify(combined, null, 2) + '\n');
console.log('tokens generated:', Object.keys(typeScale).length, 'font sizes,', THEMES.length * 2, 'theme sets');

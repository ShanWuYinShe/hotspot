// 跨端氛围画布校验：web `--cst-ambient-*` ↔ 鸿蒙 `ThemeConstants.AMBIENT` 逐值一致。
//
// web 用 `color-mix(in srgb, var(--cst-palette-X) N%, transparent)` 生成透明色；
// 鸿蒙 ArkTS 无法表达引用式透明叠加，只能预合成 `#AARRGGBB`：
//   A = round(N% × 255)，RGB = palette 色。
// 本脚本按同一公式从 web token 复算期望值，与 ThemeConstants.AMBIENT 的 6 组合 × 12 色比对，
// 防止「web 调了 alpha、鸿蒙没重算」这类静默漂移。
//
// 用法：node scripts/check-cross-platform-ambient.mjs（由 `bun run check:theme` 串联）
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const frontendRoot = dirname(dirname(fileURLToPath(import.meta.url)));
const repoRoot = dirname(frontendRoot);

const THEME_FILE = {
	vibrant: join(frontendRoot, 'src/styles/theme/themes/vibrant.css'),
	forest: join(frontendRoot, 'src/styles/theme/themes/forest.css'),
	inkwash: join(frontendRoot, 'src/styles/theme/themes/inkwash.css')
};
const ETS_PATH = join(repoRoot, 'harmonyapp/entry/src/main/ets/core/theme/ThemeConstants.ets');

const COMBOS = [
	['VIBRANT_LIGHT', 'vibrant', 'light'],
	['VIBRANT_DARK', 'vibrant', 'dark'],
	['DEFAULT_CLEAN_LIGHT', 'forest', 'light'],
	['DEFAULT_CLEAN_DARK', 'forest', 'dark'],
	['INKWASH_LIGHT', 'inkwash', 'light'],
	['INKWASH_DARK', 'inkwash', 'dark']
];
const FIELDS = [
	['glow1', 'glow-1'],
	['glow2', 'glow-2'],
	['sky', 'sky'],
	['line', 'line'],
	['dot', 'dot'],
	['sun', 'sun'],
	['ink1', 'ink-1'],
	['ink2', 'ink-2'],
	['ink3', 'ink-3'],
	['emblem', 'emblem'],
	['horizon', 'horizon'],
	['vignette', 'vignette']
];

/** 解析 themes/*.css 的块 → Map<选择器, Map<变量, 值>> */
function parseBlocks(file) {
	const lines = readFileSync(file, 'utf8').split('\n');
	const blocks = new Map();
	let current = null;
	for (const line of lines) {
		const m = line.match(/^(:root|\.(?:vibrant|forest|inkwash)(?:\.dark)?)\s*\{/);
		if (m) {
			current = m[1].trim();
			if (!blocks.has(current)) blocks.set(current, new Map());
			continue;
		}
		if (!current) continue;
		for (const t of line.matchAll(/(--cst-[a-zA-Z0-9-]+)\s*:\s*([^;]+);/g)) {
			blocks.get(current).set(t[1], t[2].trim());
		}
		if (/^\}/.test(line)) current = null;
	}
	return blocks;
}

/** 主题 × 明暗的有效变量表：light 合并 :root 与 .<theme>，dark 取 .<theme>.dark */
function effectiveVars(blocks, theme, mode) {
	if (mode === 'light') {
		const merged = new Map(blocks.get(':root') ?? []);
		for (const [k, v] of blocks.get('.' + theme) ?? []) merged.set(k, v);
		return merged;
	}
	return blocks.get('.' + theme + '.dark') ?? new Map();
}

/** 按 web 公式复算鸿蒙预合成值；无法解析时返回 null（由调用方报错） */
function expectedArgb(formula, vars) {
	const mix = formula.match(
		/color-mix\(in srgb,\s*var\((--cst-palette-[a-z0-9-]+)\)\s*([\d.]+)%,\s*transparent\)/
	);
	if (mix) {
		const palette = vars.get(mix[1]);
		if (!palette || !/^#[0-9A-Fa-f]{6}$/.test(palette)) return null;
		const alpha = Math.round((parseFloat(mix[2]) / 100) * 255);
		return '#' + alpha.toString(16).padStart(2, '0').toUpperCase() + palette.slice(1).toUpperCase();
	}
	if (formula === 'transparent') return '#00000000';
	const rgba = formula.match(/rgba\(\s*([\d.]+)\s*,\s*([\d.]+)\s*,\s*([\d.]+)\s*,\s*([\d.]+)\s*\)/);
	if (rgba) {
		const alpha = Math.round(parseFloat(rgba[4]) * 255);
		const channel = (n) => Math.round(parseFloat(n)).toString(16).padStart(2, '0').toUpperCase();
		return (
			'#' +
			alpha.toString(16).padStart(2, '0').toUpperCase() +
			channel(rgba[1]) +
			channel(rgba[2]) +
			channel(rgba[3])
		);
	}
	return null;
}

const problems = [];
const themeBlocks = {};
for (const [theme, file] of Object.entries(THEME_FILE)) themeBlocks[theme] = parseBlocks(file);

const ets = readFileSync(ETS_PATH, 'utf8');
const etsAmbient = new Map();
for (const m of ets.matchAll(
	/readonly (VIBRANT_LIGHT|VIBRANT_DARK|DEFAULT_CLEAN_LIGHT|DEFAULT_CLEAN_DARK|INKWASH_LIGHT|INKWASH_DARK): AmbientColorConfig = \{([\s\S]*?)\n  \};/g
)) {
	const fields = new Map();
	for (const f of m[2].matchAll(/([a-zA-Z0-9]+):\s*'(#[0-9A-Fa-f]{8})'/g)) fields.set(f[1], f[2]);
	etsAmbient.set(m[1], fields);
}

for (const [block, theme, mode] of COMBOS) {
	const vars = effectiveVars(themeBlocks[theme], theme, mode);
	const actual = etsAmbient.get(block);
	if (!actual) {
		problems.push(`ThemeConstants.ets 缺少 AMBIENT.${block}`);
		continue;
	}
	for (const [field, key] of FIELDS) {
		const token = '--cst-ambient-' + key;
		const formula = vars.get(token);
		if (!formula) {
			problems.push(`${theme} ${mode}: themes/*.css 缺少 ${token}`);
			continue;
		}
		const expected = expectedArgb(formula, vars);
		if (!expected) {
			problems.push(`${theme} ${mode}: 无法解析 ${token} 的取值「${formula}」`);
			continue;
		}
		const got = actual.get(field);
		if (!got) {
			problems.push(`AMBIENT.${block} 缺少字段 ${field}`);
			continue;
		}
		if (got.toUpperCase() !== expected) {
			problems.push(`氛围画布漂移：${theme}_${mode}.${field} web 期望 ${expected} ≠ 鸿蒙 ${got.toUpperCase()}`);
		}
	}
}

if (problems.length) {
	console.error('跨端氛围画布不一致（frontend themes/*.css --cst-ambient-* ↔ ThemeConstants.ets AMBIENT）：');
	for (const p of problems) console.error('  ' + p);
	console.error(`\n共 ${problems.length} 处不一致。修法：以 web token 为准，按 A = round(N% × 255) 重算鸿蒙预合成值。`);
	process.exit(1);
}
console.log('跨端氛围画布一致 OK：6 组合 × 12 色（web --cst-ambient-* ↔ 鸿蒙 AMBIENT 预合成值）。');

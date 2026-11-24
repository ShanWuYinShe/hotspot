// 跨端整卡遮罩（card scrim）校验：web `--cst-surface-card-scrim-*` 的 rgba 渐变色标
// ↔ 鸿蒙 ThemeConstants.ets ColorConfig 的 cardScrim* 预合成 `#AARRGGBB`。
//
// ArkTS 无法表达引用式透明叠加，鸿蒙侧只能存预合成 ARGB；web 改 alpha 或主题色调后
// 鸿蒙不同步就会静默漂移（压图白字对比度直接受影响），故逐值校验。
//
// 映射（web 渐变色标顺序 → 鸿蒙字段）：
//   scrim-top    rgba 0.86 → cardScrimTop；rgba 0.64（40%）→ cardScrimTopMid
//   scrim-bottom rgba 0.94/0.96 → cardScrimBottomStrong；0.86/0.88（30%）→ cardScrimBottomMid；
//                rgba 0.64/0.68（60%）→ cardScrimBottomSoft
//
// 用法：node scripts/check-cross-platform-card-scrim.mjs（由 `bun run check:theme` 串联）
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const frontendRoot = dirname(dirname(fileURLToPath(import.meta.url)));
const repoRoot = dirname(frontendRoot);
const ETS_PATH = join(repoRoot, 'harmonyapp/entry/src/main/ets/core/theme/ThemeConstants.ets');

const COMBOS = [
	{ block: 'VIBRANT_LIGHT_COLORS', theme: 'vibrant', mode: 'light' },
	{ block: 'VIBRANT_DARK_COLORS', theme: 'vibrant', mode: 'dark' },
	{ block: 'DEFAULT_CLEAN_LIGHT_COLORS', theme: 'forest', mode: 'light' },
	{ block: 'DEFAULT_CLEAN_DARK_COLORS', theme: 'forest', mode: 'dark' },
	{ block: 'INKWASH_LIGHT_COLORS', theme: 'inkwash', mode: 'light' },
	{ block: 'INKWASH_DARK_COLORS', theme: 'inkwash', mode: 'dark' }
];

function readCssBlock(theme, mode) {
	const css = readFileSync(join(frontendRoot, 'src/styles/theme/themes/' + theme + '.css'), 'utf8');
	const selector = mode === 'light' ? '.' + theme + ' {' : '.' + theme + '.dark {';
	const start = css.indexOf(selector);
	if (start < 0) throw new Error(theme + '.css 缺少 ' + selector);
	const end = css.indexOf('\n}', start);
	return css.slice(start, end < 0 ? undefined : end);
}

/** 从 linear-gradient 取 rgba 色标（跳过 transparent），按出现顺序返回 #AARRGGBB */
function gradientStops(block, varName) {
	const match = block.match(new RegExp('--' + varName + ':\\s*linear-gradient\\(([^;]+)\\);'));
	if (!match) throw new Error('缺少 --' + varName);
	const stops = [];
	const re = /rgba\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*,\s*([\d.]+)\s*\)/g;
	let m;
	while ((m = re.exec(match[1])) !== null) {
		const alpha = Math.round(Number(m[4]) * 255)
			.toString(16)
			.padStart(2, '0')
			.toUpperCase();
		const rgb = [m[1], m[2], m[3]].map((v) => Number(v).toString(16).padStart(2, '0').toUpperCase());
		stops.push('#' + alpha + rgb.join(''));
	}
	return stops;
}

function readEtsBlock(blockName) {
	const ets = readFileSync(ETS_PATH, 'utf8');
	const start = ets.indexOf('static readonly ' + blockName + ': ColorConfig = {');
	if (start < 0) throw new Error('ThemeConstants.ets 缺少 ' + blockName);
	const end = ets.indexOf('\n  };', start);
	return ets.slice(start, end < 0 ? undefined : end);
}

function field(block, name) {
	const m = block.match(new RegExp(name + ":\\s*'([^']+)'"));
	if (!m) throw new Error('ColorConfig 缺少 ' + name);
	return m[1].toUpperCase();
}

const problems = [];
const checked = [];

for (const combo of COMBOS) {
	const cssBlock = readCssBlock(combo.theme, combo.mode);
	const etsBlock = readEtsBlock(combo.block);
	const label = combo.theme + '_' + combo.mode;

	const top = gradientStops(cssBlock, 'cst-surface-card-scrim-top');
	const bottom = gradientStops(cssBlock, 'cst-surface-card-scrim-bottom');
	const expected = [
		['cardScrimTop', top[0]],
		['cardScrimTopMid', top[1]],
		['cardScrimBottomStrong', bottom[0]],
		['cardScrimBottomMid', bottom[1]],
		['cardScrimBottomSoft', bottom[2]]
	];

	for (const [name, want] of expected) {
		if (!want) {
			problems.push(label + ': web 渐变色标不足，无法推导 ' + name);
			continue;
		}
		const actual = field(etsBlock, name);
		if (actual !== want) {
			problems.push(label + '.' + name + ': 鸿蒙 ' + actual + ' ≠ 期望 ' + want);
		} else {
			checked.push(label + '.' + name);
		}
	}
}

if (problems.length > 0) {
	console.error('跨端整卡遮罩不一致（web --cst-surface-card-scrim-* rgba ↔ ThemeConstants.ets cardScrim*）：');
	for (const p of problems) console.error('  - ' + p);
	process.exit(1);
}

console.log('跨端整卡遮罩一致 OK：6 组合 × 5 字段（' + checked.length + ' 项，web scrim rgba ↔ 鸿蒙 cardScrim 预合成 ARGB）。');

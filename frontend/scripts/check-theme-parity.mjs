/**
 * 主题变量一致性检查：vibrant / forest / inkwash 三个主题（light + dark）的 --cst-*
 * 变量集合必须完全一致，且 semantic.css 引用的 --cst-palette-* 原语必须在每个主题中定义。
 *
 * 背景：主题缺失变量时不会报错，而是静默回落到另一主题/模式的值，导致跨主题
 * 视觉漂移。此前靠 AGENTS.md 手写约束，这里固化为可执行检查。
 *
 * 用法：bun run check:theme（已挂入 bun run check）。
 */

import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const root = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'styles', 'theme');

const THEMES = ['vibrant', 'forest', 'inkwash'];
/** vibrant 为锚：:root 默认即 vibrant（首帧/无脚本 fallback），其余主题逐主题与之对比 */
const ANCHOR = 'vibrant';

function parseBlocks(file, theme) {
	const lines = readFileSync(join(root, file), 'utf8').split('\n');
	const blocks = new Map();
	let current = null;
	// :root 仅 vibrant 使用（首帧/无脚本 fallback）；其余主题块形如 .forest { / .forest.dark {
	const selector = new RegExp(`^(?:(:root)|(\\.${theme}(?:\\.dark)?))\\s*\\{`);
	for (const line of lines) {
		const m = line.match(selector);
		if (m) {
			current = (m[1] ?? m[2]).trim();
			if (!blocks.has(current)) blocks.set(current, new Set());
		}
		if (!current) continue;
		for (const v of line.matchAll(/(--cst-[a-zA-Z0-9-]+)\s*:/g)) {
			blocks.get(current).add(v[1]);
		}
	}
	return blocks;
}

const palettes = {};
for (const theme of THEMES) {
	const blocks = parseBlocks(`themes/${theme}.css`, theme);
	palettes[theme] = {
		light: new Set([...(blocks.get(':root') ?? []), ...(blocks.get(`.${theme}`) ?? [])]),
		dark: blocks.get(`.${theme}.dark`) ?? new Set()
	};
}

let failed = false;

function diff(label, anchor, other) {
	const missing = [...anchor].filter((v) => !other.has(v));
	const extra = [...other].filter((v) => !anchor.has(v));
	if (missing.length || extra.length) {
		failed = true;
		console.error(`[主题一致性] ${label} 变量集合不一致（以 ${ANCHOR} 为锚）：`);
		if (missing.length) console.error(`  缺失：${missing.join(', ')}`);
		if (extra.length) console.error(`  多余：${extra.join(', ')}`);
	}
}

for (const theme of THEMES) {
	if (theme === ANCHOR) continue;
	diff(`${theme} light`, palettes[ANCHOR].light, palettes[theme].light);
	diff(`${theme} dark`, palettes[ANCHOR].dark, palettes[theme].dark);
}

// semantic.css 引用的 --cst-palette-* 原语必须在每个主题的 light+dark 中都有定义
const semantic = readFileSync(join(root, 'semantic.css'), 'utf8');
const requiredPrimitives = new Set([...semantic.matchAll(/var\((--cst-palette-[a-zA-Z0-9-]+)\)/g)].map((m) => m[1]));
for (const theme of THEMES) {
	const merged = new Set([...palettes[theme].light, ...palettes[theme].dark]);
	const missing = [...requiredPrimitives].filter((v) => !merged.has(v));
	if (missing.length) {
		failed = true;
		console.error(`[主题一致性] ${theme} 缺少 semantic.css 引用的原语：${missing.join(', ')}`);
	}
}

if (failed) {
	console.error('\n请补齐对应主题的 --cst-* 定义后再提交。');
	process.exit(1);
}

console.log(
	`主题一致性 OK：${THEMES.join('/')} light+dark 变量集合一致（light ${palettes[ANCHOR].light.size} 个 / dark ${palettes[ANCHOR].dark.size} 个），semantic 原语 ${requiredPrimitives.size} 个均已定义。`
);

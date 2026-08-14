/**
 * 主题变量一致性检查：vibrant / forest 两个主题（light + dark）的 --cst-* 变量
 * 集合必须完全一致，且 semantic.css 引用的 --cst-palette-* 原语必须在每个主题中定义。
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

function parseBlocks(file) {
	const lines = readFileSync(join(root, file), 'utf8').split('\n');
	const blocks = new Map();
	let current = null;
	for (const line of lines) {
		const selector = line.match(/^(?:(:root)|(\.vibrant(?:\.dark)?)|(\.forest(?:\.dark)?))\s*\{/);
		if (selector) {
			current = (selector[1] ?? selector[2] ?? selector[3]).trim();
			if (!blocks.has(current)) blocks.set(current, new Set());
		}
		if (!current) continue;
		for (const m of line.matchAll(/(--cst-[a-zA-Z0-9-]+)\s*:/g)) {
			blocks.get(current).add(m[1]);
		}
	}
	return blocks;
}

const vibrant = parseBlocks('themes/vibrant.css');
const forest = parseBlocks('themes/forest.css');

const vibrantLight = new Set([...(vibrant.get(':root') ?? []), ...(vibrant.get('.vibrant') ?? [])]);
const forestLight = forest.get('.forest') ?? new Set();
const vibrantDark = vibrant.get('.vibrant.dark') ?? new Set();
const forestDark = forest.get('.forest.dark') ?? new Set();

let failed = false;

function diff(label, left, right) {
	const missing = [...left].filter((v) => !right.has(v));
	const extra = [...right].filter((v) => !left.has(v));
	if (missing.length || extra.length) {
		failed = true;
		console.error(`[主题一致性] ${label} 变量集合不一致：`);
		if (missing.length) console.error(`  缺失（forest 相对 vibrant）：${missing.join(', ')}`);
		if (extra.length) console.error(`  多余（forest 相对 vibrant）：${extra.join(', ')}`);
	}
}

diff('light', vibrantLight, forestLight);
diff('dark', vibrantDark, forestDark);

// semantic.css 引用的 --cst-palette-* 原语必须在每个主题的 light+dark 中都有定义
const semantic = readFileSync(join(root, 'semantic.css'), 'utf8');
const requiredPrimitives = new Set([...semantic.matchAll(/var\((--cst-palette-[a-zA-Z0-9-]+)\)/g)].map((m) => m[1]));
for (const [name, blocks] of [
	['vibrant', new Set([...vibrantLight, ...vibrantDark])],
	['forest', new Set([...forestLight, ...forestDark])]
]) {
	const missing = [...requiredPrimitives].filter((v) => !blocks.has(v));
	if (missing.length) {
		failed = true;
		console.error(`[主题一致性] ${name} 缺少 semantic.css 引用的原语：${missing.join(', ')}`);
	}
}

if (failed) {
	console.error('\n请补齐对应主题的 --cst-* 定义后再提交。');
	process.exit(1);
}

console.log(
	`主题一致性 OK：vibrant/forest light+dark 变量集合一致（light ${vibrantLight.size} 个 / dark ${vibrantDark.size} 个），semantic 原语 ${requiredPrimitives.size} 个均已定义。`
);

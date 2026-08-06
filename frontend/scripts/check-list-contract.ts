/**
 * 列表骨架契约静态检查（防「多边框/行级卡片化」复发）。
 *
 * 背景：ListItem 的 joined 行（app-list-row）与 ListContainer（embedded/bordered）
 * 已定义行间分隔与容器级唯一外框，调用方不得再给行叠加卡片样式，否则每个 item
 * 会呈现独立圆角矩形边缘（历史 bug：/map 列表 6 处 rounded-control）。
 *
 * 规则：
 *   A. ListItem 调用点（<ListItem ...> 开标签，支持跨行）的 class 属性不得含
 *      rounded/border/shadow/ring 卡片类——joined 行的圆角、边框、阴影由
 *      ListContainer 的 divide 线与 ListItem 的 hover-fill 承担；卡片化行必须
 *      使用 tone="card"（组件自带圆角边框），且只放独立卡片容器（gap 分隔），
 *      不得放入 ListContainer bordered（容器已有唯一外框）。
 *   B. ListItem 调用点的 style 字符串属性不得含 border/box-shadow/outline 声明。
 *
 * 运行：bun scripts/check-list-contract.ts（已挂入 `bun run check`）。
 * 误报处理：新增合法用法前先到本文件/AGENTS.md 对齐契约，不要直接加白名单。
 */
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { extname, join, relative } from 'node:path';

const SRC = join(import.meta.dirname, '..', 'src');

/** 行级卡片类：joined 行禁止出现的 Tailwind 类族（含 rounded-full/rounded-[..]/border-t 等变体） */
const CARD_CLASS_RE = /\b(?:rounded|border|shadow|ring)[\w-]*/g;
/** 行级卡片声明：style 字符串内禁止出现的 CSS 属性 */
const CARD_STYLE_RE = /(?:^|[;"'}\s])(?:border|box-shadow|outline)\s*:/g;

function walk(dir: string): string[] {
	return readdirSync(dir).flatMap((f) => {
		const p = join(dir, f);
		return statSync(p).isDirectory() ? walk(p) : [p];
	}).filter((p) => extname(p) === '.svelte');
}

const problems: string[] = [];

for (const file of walk(SRC)) {
	const lines = readFileSync(file, 'utf8').split('\n');
	for (let i = 0; i < lines.length; i++) {
		// 仅匹配 ListItem 开标签（`<ListItem` 可独立成行，属性换行书写；$ 覆盖行尾）
		if (!/<ListItem(?=[\s>]|$)/.test(lines[i])) continue;
		// 收集完整开标签（可能跨行），直到 >（属性值内不会出现 >）
		let tag = lines[i];
		if (!tag.trimStart().startsWith('<ListItem')) continue; // 排除 </ListItem> 闭合行
		let j = i;
		while (!/>/.test(tag) && j < lines.length - 1) {
			j += 1;
			tag += '\n' + lines[j];
		}
		const rel = relative(SRC, file);
		const cls = tag.match(/class="([^"]*)"/);
		if (cls) {
			for (const m of cls[1].matchAll(CARD_CLASS_RE)) {
				problems.push(
					`${rel}:${i + 1} ListItem class 含行级卡片类「${m[0]}」——joined 行禁止叠加边框/圆角/阴影，` +
						'卡片化请用 tone="card"（独立卡片容器）或 ListContainer bordered（容器级唯一外框）'
				);
			}
		}
		const styleAttr = tag.match(/style="([^"]*)"/);
		if (styleAttr) {
			for (const m of styleAttr[1].matchAll(CARD_STYLE_RE)) {
				problems.push(`${rel}:${i + 1} ListItem style 含行级边框声明「${m[0].trim()}」`);
			}
		}
	}
}

if (problems.length > 0) {
	console.error(`✗ 列表骨架契约违规 ${problems.length} 处：`);
	for (const p of problems) console.error('  ' + p);
	process.exit(1);
}
console.log('✓ 列表骨架契约检查通过（无行级边框/圆角/阴影残留）');

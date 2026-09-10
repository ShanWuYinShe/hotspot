/**
 * Tailwind 类名有效性检查（死类名防线）。
 *
 * 背景：Tailwind v4 对「未注册的 utility」既不报错也不生成 CSS——类名写错
 * （`shadow-raised-xs` 从未注册、`bg-surface-subtle` 缺 --color-* 注册、
 * `text-text-tertiary` 语义名不存在、`scrollbar-none` 拼错成 no-scrollbar）
 * 会静默失效，页面只是「少了阴影/底色」而不报任何错。svelte-check、eslint、
 * check-theme-parity 三道护栏全都看不见消费端的类名，本项目已因此踩坑两次
 * （--cst-text-on-status 未注册；本次 8 个死类名跨全部设备与主题组合失效）。
 *
 * 检查方式（只做「不可能存在」的判定，宁漏不误报）：
 *   1. 从 .svelte 的 class 属性（含三元/模板字面量里的类名串）提取候选类名；
 *   2. 剥掉变体前缀（hover: / md: / group-hover: / data-[...]: / ! 等）与
 *      透明度修饰（/70）、任意值（[...]）；
 *   3. 按「类名前缀 → Tailwind 命名空间」取合法键集合 =
 *      项目 theme-registry.css 的 @theme inline 键 ∪ Tailwind 自带 theme.css 的默认键
 *      ∪ 该前缀的功能性关键字白名单（如 text-center、border-2、rounded-full）；
 *   4. 键集合为空 → 该命名空间缺失（布局/功能类，不管）；键不在集合中 → 报错。
 *
 * 用法：bun run check:theme（已挂入该链与 bun run check）。
 */

import { readFileSync, readdirSync, statSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const srcDir = join(root, 'src');

/* ── 1. 合法键集合 ─────────────────────────────────────────────────── */

/** 解析 `--<ns>-<key>: ...` 形式，ns 取自候选命名空间（长前缀优先）。 */
const NAMESPACES = [
	'color', 'inset-shadow', 'text-shadow', 'shadow', 'text', 'font-weight', 'font',
	'tracking', 'leading', 'radius', 'blur', 'ease', 'animate', 'opacity', 'spacing',
	'container', 'breakpoint', 'transition-duration', 'border-width', 'aspect'
].sort((a, b) => b.length - a.length);

function collectKeys(css) {
	/** @type {Map<string, Set<string>>} */
	const byNs = new Map();
	// 逐字符扫描而非按行：tw-animate-css 等文件是压缩单行
	for (const m of css.matchAll(/(?:^|[;{\s])--([a-z0-9-]+)\s*:/g)) {
		const name = m[1];
		const ns = NAMESPACES.find((n) => name.startsWith(`${n}-`));
		if (!ns) continue;
		const key = name.slice(ns.length + 1);
		if (!byNs.has(ns)) byNs.set(ns, new Set());
		byNs.get(ns).add(key);
	}
	return byNs;
}

const projectKeys = collectKeys(readFileSync(join(srcDir, 'styles', 'theme', 'theme-registry.css'), 'utf8'));
const builtinKeys = collectKeys(readFileSync(join(root, 'node_modules', 'tailwindcss', 'theme.css'), 'utf8'));
// tw-animate-css（tailwind.css 已 @import）：animate-in / animate-out 等键由该包提供
const extraKeys = collectKeys(readFileSync(join(root, 'node_modules', 'tw-animate-css', 'dist', 'tw-animate.css'), 'utf8'));

const keysOf = (ns) =>
	new Set([...(projectKeys.get(ns) ?? []), ...(builtinKeys.get(ns) ?? []), ...(extraKeys.get(ns) ?? [])]);

/* ── 2. 类名前缀 → 命名空间 + 功能性关键字白名单 ───────────────────── */

/** 颜色前缀：键走 color 命名空间。 */
const COLOR_PREFIXES = [
	'bg', 'text', 'border', 'ring', 'ring-offset', 'fill', 'stroke', 'divide', 'outline',
	'accent', 'caret', 'decoration', 'from', 'via', 'to'
];

/** 功能性关键字：不是 token 键，但同前缀下合法（Tailwind 内置语义类）。 */
const FUNCTIONAL = {
	// Tailwind 在 utilities 层特判的颜色关键字（不走 --color-* 主题变量）
	__color: ['transparent', 'current', 'inherit'],
	bg: ['clip-border', 'clip-padding', 'clip-content', 'clip-text', 'origin-border', 'origin-padding',
		'origin-content', 'repeat', 'no-repeat', 'repeat-x', 'repeat-y', 'repeat-round', 'repeat-space',
		'fixed', 'local', 'scroll', 'cover', 'contain', 'bottom', 'top', 'left', 'right', 'center',
		'left-bottom', 'left-top', 'right-bottom', 'right-top', 'none', 'auto'],
	text: ['center', 'left', 'right', 'justify', 'start', 'end', 'wrap', 'nowrap', 'balance', 'pretty',
		'ellipsis', 'clip', 'truncate'],
	border: ['collapse', 'separate', 'solid', 'dashed', 'dotted', 'double', 'hidden', 'none',
		'x', 'y', 't', 'r', 'b', 'l', 's', 'e', '0', '2', '4', '8'],
	ring: ['inset', '0', '1', '2', '4', '8'],
	'ring-offset': ['0', '1', '2', '4', '8'],
	divide: ['x', 'y', 'solid', 'dashed', 'dotted', 'double', 'none', 'reverse'],
	outline: ['none', 'hidden', 'solid', 'dashed', 'dotted', 'double', '0', '1', '2', '4', '8'],
	decoration: ['solid', 'dashed', 'dotted', 'double', 'wavy', '0', '1', '2', '4', '8',
		'underline', 'overline', 'line-through', 'no-underline', 'auto', 'from-font'],
	fill: ['none'],
	stroke: ['none', '0', '1', '2'],
	shadow: ['none', 'inner'],
	rounded: ['none', 'full', 't', 'r', 'b', 'l', 's', 'e', 'tl', 'tr', 'br', 'bl', 'ss', 'se', 'es', 'ee'],
	leading: ['0', 'none'],
	tracking: ['normal'],
	opacity: ['initial'],
	duration: ['initial'],
	ease: ['initial', 'linear'],
	animate: ['none'],
	blur: ['none']
};

/** 数字键（Tailwind 允许任意数字刻度）：opacity-40 / duration-200 / blur-2 / border-2 等。 */
const NUMERIC_OK = new Set(['opacity', 'duration', 'blur', 'leading', 'border-width']);

/**
 * 类名前缀 → 可接受的命名空间（任一命中即合法）。
 * 多命名空间的前缀（text/border/font）是歧义前缀：`text-sm` 是字号、`text-text-primary`
 * 是颜色；`border-2` 是宽度、`border-hairline` 走项目注册的 border-width；
 * `font-medium` 是字重、`font-sans` 是字族。
 */
const NS_BY_PREFIX = {
	bg: ['color'],
	text: ['color', 'text'],
	border: ['color', 'border-width'],
	ring: ['color'],
	'ring-offset': ['color'],
	fill: ['color'],
	stroke: ['color'],
	divide: ['color'],
	outline: ['color'],
	accent: ['color'],
	caret: ['color'],
	decoration: ['color'],
	from: ['color'],
	via: ['color'],
	to: ['color'],
	shadow: ['shadow'],
	rounded: ['radius'],
	tracking: ['tracking'],
	leading: ['leading'],
	opacity: ['opacity'],
	duration: ['transition-duration'],
	ease: ['ease'],
	animate: ['animate'],
	blur: ['blur'],
	font: ['font', 'font-weight']
};

/** 前缀匹配按长度降序，保证 ring-offset 先于 ring 命中。 */
const PREFIXES = Object.keys(NS_BY_PREFIX).sort((a, b) => b.length - a.length);

/** 方向段（rounded-t-lg / border-l-0 / divide-x-2 等）：剥掉后按剩余键校验。 */
const DIRECTIONS = {
	rounded: ['tl', 'tr', 'br', 'bl', 'ss', 'se', 'es', 'ee', 't', 'r', 'b', 'l', 's', 'e'],
	border: ['x', 'y', 't', 'r', 'b', 'l', 's', 'e'],
	divide: ['x', 'y']
};

function stripDirection(prefix, key) {
	const dirs = DIRECTIONS[prefix];
	if (!dirs) return key;
	for (const d of dirs) {
		if (key.startsWith(`${d}-`)) return key.slice(d.length + 1);
	}
	return key;
}

/* ── 2b. 项目自定义工具类（utilities.css / layout.css / themes）─────────
   `scrollbar-none` 这类「拼错项目自定义类」Tailwind 侧同样静默失效
   （项目定义的是 .no-scrollbar）。这里用一组不会与 Tailwind 命名空间冲突的
   词干做「必须是已定义 .class」的强校验。 */
const CUSTOM_STEMS = [
	'hover-', 'tap-', 'card-', 'surface-', 'image-hover-frame', 'avatar-entry',
	'nav-glass', 'focus-ring', 'hit-slop', 'count-pop', 'list-row-', 'page-header-',
	'scrollbar', 'link-btn', 'header-action', 'input-surface', 'icon-badge',
	'circle-hover', 'sticky-header', 'ambient-glow', 'tab-item', 'context-rail',
	'rail-icon', 'eyebrow', 'badge', 'spinner', 'desktop-', 'tablet-', 'content-mode-',
	'page-canvas', 'no-scrollbar'
];

/** 项目样式层定义的全部 .class 名（含 .svelte 内 scoped <style> 块）。 */
function collectDefinedClasses() {
	const defined = new Set();
	const addFromCss = (css) => {
		for (const m of css.matchAll(/\.((?:\\.|[-\w])+)/g)) defined.add(m[1].replace(/\\/g, ''));
	};
	const dir = join(srcDir, 'styles');
	const readAll = (d) => {
		for (const entry of readdirSync(d)) {
			const full = join(d, entry);
			if (statSync(full).isDirectory()) readAll(full);
			else if (entry.endsWith('.css')) addFromCss(readFileSync(full, 'utf8'));
		}
	};
	readAll(dir);
	// 组件 scoped 样式（如 RailIcon 的 .rail-icon）
	for (const file of walk(srcDir)) {
		const source = readFileSync(file, 'utf8');
		for (const m of source.matchAll(/<style[^>]*>([\s\S]*?)<\/style>/g)) addFromCss(m[1]);
	}
	return defined;
}

const DEFINED_CLASSES = collectDefinedClasses();

/* ── 3. 从源码提取候选类名 ─────────────────────────────────────────── */

function walk(dir, out = []) {
	for (const entry of readdirSync(dir)) {
		const full = join(dir, entry);
		if (statSync(full).isDirectory()) {
			if (entry === '__tests__' || entry === 'node_modules') continue;
			walk(full, out);
		} else if (entry.endsWith('.svelte')) {
			out.push(full);
		}
	}
	return out;
}

/** 取出 class= 属性值文本（支持 "..."、'...'、{...} 花括号配平）。 */
function classAttributeTexts(source) {
	const texts = [];
	const re = /\bclass\s*=\s*/g;
	let m;
	while ((m = re.exec(source))) {
		let i = m.index + m[0].length;
		const q = source[i];
		if (q === '"' || q === "'") {
			const end = source.indexOf(q, i + 1);
			if (end > 0) texts.push(source.slice(i + 1, end));
			re.lastIndex = end + 1;
		} else if (q === '{') {
			let depth = 0;
			let j = i;
			for (; j < source.length; j++) {
				if (source[j] === '{') depth++;
				else if (source[j] === '}') {
					depth--;
					if (depth === 0) break;
				}
			}
			texts.push(source.slice(i + 1, j));
			re.lastIndex = j + 1;
		}
	}
	return texts;
}

/** 从一个 class 属性文本中提取裸类名（去变体前缀/修饰符）。
 *  同时覆盖两种写法：静态 class="a b c"（引号已被 classAttributeTexts 剥掉，
 *  整段即类名列表）与表达式 class={cn('a', x && 'b')}（类名在引号字符串里）。 */
function extractCandidates(text) {
	const chunks = [];
	let rest = '';
	let last = 0;
	const re = /'(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"|`(?:[^`\\]|\\.)*`/g;
	let m;
	while ((m = re.exec(text))) {
		rest += text.slice(last, m.index);
		chunks.push(m[0].slice(1, -1));
		last = m.index + m[0].length;
	}
	rest += text.slice(last);
	// 引号外的静态部分：先剔除 {…} 与 (…) 内的 JS 表达式，剩下的才可能是类名
	chunks.push(rest.replace(/\{[^{}]*\}/g, ' '));

	const out = [];
	for (const chunk of chunks) {
		for (const raw of chunk.split(/\s+/)) {
			const token = raw.trim();
			if (!token) continue;
			// 只接受「像类名」的 token：JS 标识符/运算符/函数调用一律跳过
			if (!/^[a-z0-9!:_\-.[\]/%,'"=]+$/.test(token)) continue;
			if (/[()$&|?,]/.test(token)) continue;
			out.push(token);
		}
	}
	return out;
}

function normalize(token) {
	let t = token;
	// 变体前缀：hover: / md: / group-hover: / data-[state=open]: / focus-visible: ...
	// 变体段内允许 [..] 与字母数字与 - _ . = " '
	let guard = 0;
	for (;;) {
		const m = t.match(/^([a-z-]+|\[[^\]]*\]|data-\[[^\]]*\]|aria-\[[^\]]*\]|has-\[[^\]]*\]|not-\[[^\]]*\]|group-\[[^\]]*\]|peer-\[[^\]]*\]|supports-\[[^\]]*\]|nth-\[[^\]]*\]):/);
		if (!m || guard++ > 8) break;
		t = t.slice(m[0].length);
	}
	t = t.replace(/^!/, '');
	// 负值前缀
	t = t.replace(/^-/, '');
	// 透明度修饰 / 任意值
	t = t.replace(/\/.*$/, '');
	if (t.includes('[')) return null; // 任意值：交由开发者自行保证（token 引用形式）
	return t;
}

/* ── 4. 校验 ───────────────────────────────────────────────────────── */

const problems = [];

for (const file of walk(srcDir)) {
	const source = readFileSync(file, 'utf8');
	// 去掉注释，避免把注释里提到的类名当真实用法
	const cleaned = source
		.replace(/<!--[\s\S]*?-->/g, '')
		.replace(/\/\*[\s\S]*?\*\//g, '')
		.replace(/(^|[^:])\/\/[^\n]*/g, '$1');

	for (const attrText of classAttributeTexts(cleaned)) {
		for (const token of extractCandidates(attrText)) {
			const cls = normalize(token);
			if (!cls) continue;

			// 规则 2：项目自定义工具类必须真实存在（防 scrollbar-none 类拼写漂移）
			if (
				CUSTOM_STEMS.some((stem) => cls === stem || cls.startsWith(stem)) &&
				!DEFINED_CLASSES.has(cls)
			) {
				problems.push({ file, cls, kind: 'custom' });
				continue;
			}

			const prefix = PREFIXES.find((p) => cls.startsWith(`${p}-`));
			if (!prefix) continue; // 不在受检命名空间内（布局/功能类）
			const rawKey = cls.slice(prefix.length + 1);
			if (!rawKey) continue;
			if ((FUNCTIONAL[prefix] ?? []).includes(rawKey)) continue;
			if (FUNCTIONAL.__color.includes(rawKey) && NS_BY_PREFIX[prefix].includes('color')) continue;

			const key = stripDirection(prefix, rawKey);
			const namespaces = NS_BY_PREFIX[prefix];
			const ok = namespaces.some((ns) => {
				if (NUMERIC_OK.has(ns) && /^\d+(\.\d+)?$/.test(key)) return true;
				return keysOf(ns).has(key);
			});
			if (!ok) problems.push({ file, cls, kind: 'token' });
		}
	}
}

/* ── 5. 输出 ───────────────────────────────────────────────────────── */

if (problems.length) {
	console.error(`[类名有效性] 发现 ${problems.length} 个不存在的 Tailwind 类名（Tailwind v4 会静默失效）：`);
	const byCls = new Map();
	for (const p of problems) {
		if (!byCls.has(p.cls)) byCls.set(p.cls, []);
		byCls.get(p.cls).push(p.file);
	}
	for (const [cls, files] of [...byCls.entries()].sort()) {
		console.error(`  ${cls}  (${files.length} 处)`);
		for (const f of files.slice(0, 4)) console.error(`      ${f.slice(root.length + 1)}`);
		if (files.length > 4) console.error(`      … 另 ${files.length - 4} 处`);
	}
	console.error('\n修法：补 theme-registry.css 的 @theme inline 注册 + themes/*.css 的 --cst-* 定义，');
	console.error('或改用既有语义类。详见 frontend/AGENTS.md「样式自查清单」第 6 条。');
	process.exit(1);
}

console.log('类名有效性 OK：源码中消费的 token 类名均已注册（含变体前缀与功能类白名单）。');

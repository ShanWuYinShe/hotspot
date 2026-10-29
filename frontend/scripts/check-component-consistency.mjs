#!/usr/bin/env node
// UI 一致性检查（2026-10-29）：同类型 UI 必须走同一原语/同一类契约。
// 目标：任何「搜索/单行输入/多行文本/下拉/复选/文件上传」在所有页面视觉与行为一致；
// 新页面若绕过原语（裸 input/textarea/select、手写 .input-surface、覆盖原语几何），
// 本检查在 bun run check（及 CI frontend job）阶段直接失败。
//
// 例外必须登记在 scripts/ui-consistency-allowlist.json（rule + file + reason），
// 不允许用 ignore 注释绕过；allowlist 变更需在 docs/frontend-ui-audit.md 记录理由。
//
// 说明：这是源码级契约检查（jsdom 无法验证跨页视觉一致），规则与 docs/ui-components.md
// 「同类型一致性契约」一一对应。

import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative, sep } from 'node:path';

const ROOT = process.cwd();
const SRC = join(ROOT, 'src');
const ALLOWLIST_PATH = join(ROOT, 'scripts', 'ui-consistency-allowlist.json');

const PRIMITIVES = {
	search: 'src/components/forms/SearchInput.svelte',
	input: 'src/components/forms/Input.svelte',
	textarea: 'src/components/forms/Textarea.svelte',
	select: 'src/components/forms/Select.svelte',
	checkbox: 'src/components/forms/Checkbox.svelte',
	imagePicker: 'src/components/forms/ImagePicker.svelte',
	tabBar: 'src/components/navigation/TabBar.svelte'
};

const RULES = {
	'raw-search': '搜索字段必须使用 forms/SearchInput（禁止裸 input/textarea + 搜索语义）',
	'search-override': 'SearchInput 不允许调用方覆盖 padding/圆角/字号/投影（几何由原语唯一决定）',
	'search-aria': 'SearchInput 必须传 ariaLabel（按页面语义命名）',
	'raw-input': '单行输入必须使用 forms/Input（聊天/评论输入条等已登记例外）',
	'raw-textarea': '多行文本必须使用 forms/Textarea（聊天气泡内联编辑已登记例外）',
	'raw-select': '下拉选择必须使用 forms/Select（field/ghost 两种变体）',
	'raw-checkbox': '复选框必须使用 forms/Checkbox',
	'raw-file-input': '文件上传必须走 forms/ImagePicker（头像/封面徽标已登记例外）',
	'input-surface-outside': '.input-surface 只能由输入族原语消费，页面不得手写输入表面',
	'tab-role-outside': 'Tab 必须使用 navigation/TabBar（页面不得自绘 role=tab/tablist）',
	'raw-badge': '徽章/标签必须使用 data-display/Badge（压图角标用 .media-badge，未读数用 UnreadBadge）',
	'button-radius-override': 'Button 不允许调用方覆盖圆角（派 A 固定 rounded-md；派 B 用 FollowButton/chip）',
	'untiered-radius': '圆角只允许登记档位（禁止 rounded-2xl/3xl 等未注册档）',
	'handwritten-cta': '实底 accent 胶囊 CTA 必须用 Button（禁止手写 <button> 拼语言）',
	'raw-avatar-img': '头像必须使用 shared/Avatar（裸 <img rounded-full> 会丢占位/兜底/渐显）',
	'sheet-radius-override': 'Dialog 圆角由组件档位决定（浮层卡片用 floating，禁止调用方覆盖 rounded-*）'
};

function loadAllowlist() {
	try {
		const raw = JSON.parse(readFileSync(ALLOWLIST_PATH, 'utf8'));
		return Array.isArray(raw) ? raw : [];
	} catch {
		return [];
	}
}
const allowlist = loadAllowlist();
const isAllowed = (rule, file) =>
	allowlist.some((e) => e.rule === rule && (e.file === file || (e.glob && file.startsWith(e.glob.replace(/\*$/, '')))));

function walk(dir, out = []) {
	for (const name of readdirSync(dir)) {
		const full = join(dir, name);
		const st = statSync(full);
		if (st.isDirectory()) {
			// 测试文件里的 "input-surface" 是断言字符串而非样式消费，不纳入一致性扫描
			if (name === '__tests__') continue;
			walk(full, out);
		} else if (/\.(svelte|ts)$/.test(name)) out.push(full);
	}
	return out;
}

/** 把内容替换为等长空格（保留换行，行号不变），用于屏蔽注释/脚本块。 */
function mask(text, pattern) {
	return text.replace(pattern, (m) => m.replace(/[^\n]/g, ' '));
}

/** 屏蔽 HTML 注释 + <script> 块（标签规则用；类名规则仍扫描脚本里的字符串常量）。 */
function maskForMarkup(text) {
	let out = mask(text, /<!--[\s\S]*?-->/g);
	out = mask(out, /<script[^>]*>[\s\S]*?<\/script>/g);
	out = mask(out, /<style[^>]*>[\s\S]*?<\/style>/g);
	return out;
}

/** 从 index 开始找到标签结束的 '>'（跳过引号字符串与 {} 表达式内的箭头 '>'）。 */
function readTag(text, start) {
	let quote = null;
	let brace = 0;
	for (let i = start; i < text.length; i++) {
		const ch = text[i];
		if (quote) {
			if (ch === quote) quote = null;
			continue;
		}
		if (ch === '"' || ch === "'" || ch === '\u0060') {
			quote = ch;
			continue;
		}
		if (ch === '{') {
			brace++;
			continue;
		}
		if (ch === '}') {
			if (brace > 0) brace--;
			continue;
		}
		if (ch === '>' && brace === 0) return text.slice(start, i + 1);
	}
	return text.slice(start, Math.min(text.length, start + 400));
}

/** 粗略提取标签的 class 属性 tokens（支持 class="..." / class={...}）。 */
function classTokens(tag) {
	const m = tag.match(/\bclass=(?:"([^"]*)"|'([^']*)'|\{([^}]*)\})/);
	if (!m) return [];
	const raw = m[1] ?? m[2] ?? m[3] ?? '';
	return raw.split(/\s+/).filter(Boolean);
}

function lineOf(text, index) {
	let line = 1;
	for (let i = 0; i < index && i < text.length; i++) if (text[i] === '\n') line++;
	return line;
}

const violations = [];
function report(rule, file, line, detail) {
	violations.push({ rule, file, line, detail });
}

const files = walk(SRC);
for (const abs of files) {
	const file = relative(ROOT, abs).split(sep).join('/');
	const text = readFileSync(abs, 'utf8');
	const markup = file.endsWith('.svelte') ? maskForMarkup(text) : text;

	// ── 搜索语义的裸控件（R1） ──
	if (file !== PRIMITIVES.search && file.endsWith('.svelte')) {
		const re = /<(input|textarea)\b/g;
		let m;
		while ((m = re.exec(markup))) {
			const tag = readTag(markup, m.index);
			if (/placeholder[^>]*(搜索|[Ss]earch)|aria-label[^>]*(搜索|[Ss]earch)/.test(tag)) {
				report('raw-search', file, lineOf(markup, m.index), '搜索语义控件必须用 SearchInput');
			}
		}
	}

	// ── SearchInput 几何覆盖 / 缺 ariaLabel（R2/R3） ──
	if (file.endsWith('.svelte')) {
		const re = /<SearchInput\b/g;
		let m;
		while ((m = re.exec(markup))) {
			const tag = readTag(markup, m.index);
			if (/(^|\s)(px-|py-|pt-|pb-|pl-|pr-|text-(xs|sm|md|lg|xl)|rounded-|shadow-)/.test(tag)) {
				report('search-override', file, lineOf(markup, m.index), tag.replace(/\s+/g, ' ').slice(0, 160));
			}
			if (!/\bariaLabel=/.test(tag)) {
				report('search-aria', file, lineOf(markup, m.index), '缺少 ariaLabel');
			}
		}
	}

	// ── 裸 textarea/select/checkbox/file（R4–R8） ──
	if (file.endsWith('.svelte')) {
		const checkTag = (pattern, primitive, rule, extra) => {
			const re = new RegExp(pattern, 'g');
			let m;
			while ((m = re.exec(markup))) {
				if (file === primitive) continue;
				if (isAllowed(rule, file)) continue;
				const tag = readTag(markup, m.index);
				report(rule, file, lineOf(markup, m.index), extra ? extra + ': ' + tag.replace(/\s+/g, ' ').slice(0, 140) : tag.replace(/\s+/g, ' ').slice(0, 140));
			}
		};
		checkTag('<textarea\\b', PRIMITIVES.textarea, 'raw-textarea');
		checkTag('<select\\b', PRIMITIVES.select, 'raw-select');
		checkTag('<input\\b[^>]*\\btype="checkbox"', PRIMITIVES.checkbox, 'raw-checkbox');
		checkTag('<input\\b[^>]*\\btype="file"', PRIMITIVES.imagePicker, 'raw-file-input');

		// 裸 input（排除 checkbox/file 已由上面规则覆盖；primitive 内部放行）
		const inputAllowed = new Set([PRIMITIVES.search, PRIMITIVES.input, PRIMITIVES.checkbox, PRIMITIVES.imagePicker]);
		const ire = /<input\b/g;
		let im;
		while ((im = ire.exec(markup))) {
			if (inputAllowed.has(file) || isAllowed('raw-input', file)) continue;
			const tag = readTag(markup, im.index);
			if (/\btype="(checkbox|file)"/.test(tag)) continue;
			report('raw-input', file, lineOf(markup, im.index), tag.replace(/\s+/g, ' ').slice(0, 140));
		}
	}

	// ── .input-surface 只能由输入族原语消费（R9；脚本里的类常量也算消费） ──
	if (!Object.values(PRIMITIVES).includes(file) && !isAllowed('input-surface-outside', file)) {
		const re = /(^|[^.\w-])input-surface\b/g;
		let m;
		while ((m = re.exec(text))) {
			report('input-surface-outside', file, lineOf(text, m.index), '页面不得手写输入表面类');
		}
	}

	// ── role=tab/tablist 只能由 TabBar 提供（R10） ──
	if (file !== PRIMITIVES.tabBar && file.endsWith('.svelte') && !isAllowed('tab-role-outside', file)) {
		const re = /<[a-zA-Z][^>]*\brole="tab(list)?"/g;
		let m;
		while ((m = re.exec(markup))) report('tab-role-outside', file, lineOf(markup, m.index), '页面自绘 Tab');
	}

	// ── 非输入族同类型规则（R11–R16）：徽章/按钮圆角/未注册圆角/手写 CTA/裸头像/弹层圆角 ──
	if (file.endsWith('.svelte')) {
		const BADGE_PRIMITIVES = new Set([
			'src/components/data-display/Badge.svelte',
			'src/components/shared/UnreadBadge.svelte',
			'src/components/forms/TagPill.svelte',
			PRIMITIVES.tabBar
		]);

		// R11 徽章：旧 .badge 类或手写「inline-flex + h-5/6 + rounded-full + px-」小胶囊
		if (!BADGE_PRIMITIVES.has(file) && !isAllowed('raw-badge', file)) {
			const re = /<(span|div|p)\b/g;
			let m;
			while ((m = re.exec(markup))) {
				const tag = readTag(markup, m.index);
				const tokens = classTokens(tag);
				if (tokens.includes('badge')) {
					report('raw-badge', file, lineOf(markup, m.index), '旧 .badge 类 → data-display/Badge');
					continue;
				}
				if (tokens.includes('media-badge') || tokens.includes('media-badge--icon')) continue;
				const pill =
					tokens.includes('inline-flex') &&
					tokens.includes('rounded-full') &&
					(tokens.includes('h-5') || tokens.includes('h-6')) &&
					tokens.some((t) => /^px-/.test(t));
				if (pill) {
					report('raw-badge', file, lineOf(markup, m.index), '手写徽章 → data-display/Badge（压图用 .media-badge）');
				}
			}
		}

		// R12 Button 圆角覆盖（派 A 固定 rounded-md）
		if (!isAllowed('button-radius-override', file)) {
			const re = /<Button\b/g;
			let m;
			while ((m = re.exec(markup))) {
				const tag = readTag(markup, m.index);
				const overridden = classTokens(tag).some((t) => /^rounded-(full|sm|lg|xl)$/.test(t));
				if (overridden) report('button-radius-override', file, lineOf(markup, m.index), tag.replace(/\s+/g, ' ').slice(0, 120));
			}
		}

		// R13 未注册圆角档（rounded-2xl/3xl）
		if (!isAllowed('untiered-radius', file)) {
			const re = /rounded-(2xl|3xl)\b/g;
			let m;
			while ((m = re.exec(text))) {
				const start = text.lastIndexOf('\n', m.index) + 1;
				const lineText = text.slice(start, text.indexOf('\n', m.index) === -1 ? text.length : text.indexOf('\n', m.index)).trim();
				if (lineText.startsWith('//') || lineText.startsWith('*') || lineText.startsWith('/*') || lineText.startsWith('<!--')) continue;
				report('untiered-radius', file, lineOf(text, m.index), lineText.slice(0, 120));
			}
		}

		// R14 手写实底 accent 胶囊 CTA（应走 Button；派 B chip 请用 FollowButton/TagPill）
		if (!isAllowed('handwritten-cta', file)) {
			const re = /<button\b/g;
			let m;
			while ((m = re.exec(markup))) {
				const tag = readTag(markup, m.index);
				const tokens = classTokens(tag);
				const cta =
					tokens.includes('bg-accent-default') &&
					tokens.includes('rounded-full') &&
					tokens.some((t) => /^px-/.test(t));
				if (cta) report('handwritten-cta', file, lineOf(markup, m.index), tag.replace(/\s+/g, ' ').slice(0, 120));
			}
		}

		// R15 裸头像 img（必须在 Avatar 内）
		if (file !== 'src/components/shared/Avatar.svelte' && !isAllowed('raw-avatar-img', file)) {
			const re = /<img\b/g;
			let m;
			while ((m = re.exec(markup))) {
				const tag = readTag(markup, m.index);
				const tokens = classTokens(tag);
				// 品牌 logo（alt 含 HotSpot/Logo）是品牌标记，不是用户头像，排除
				if (/\bHotSpot\b|\bLogo\b/.test(tag)) continue;
				const avatarish =
					tokens.includes('rounded-full') &&
					tokens.some((t) => /^(size-|w-\d|h-\d)/.test(t));
				if (avatarish) report('raw-avatar-img', file, lineOf(markup, m.index), tag.replace(/\s+/g, ' ').slice(0, 120));
			}
		}

		// R16 Dialog 圆角覆盖（浮层卡片用 floating，不手写 rounded-*）
		if (!isAllowed('sheet-radius-override', file)) {
			const re = /<Dialog\.Content\b/g;
			let m;
			while ((m = re.exec(markup))) {
				const tag = readTag(markup, m.index);
				const overridden = classTokens(tag).some((t) => /^rounded-(sm|md|lg|xl|2xl)$/.test(t));
				if (overridden) report('sheet-radius-override', file, lineOf(markup, m.index), tag.replace(/\s+/g, ' ').slice(0, 120));
			}
		}
	}
}

if (violations.length === 0) {
	console.log('UI 一致性 OK：输入族 + 徽章/按钮圆角/头像/弹层等 16 条同类型原语约束全部满足（扫描 ' + files.length + ' 文件）。');
	process.exit(0);
}

const byRule = new Map();
for (const v of violations) {
	if (!byRule.has(v.rule)) byRule.set(v.rule, []);
	byRule.get(v.rule).push(v);
}
console.error('UI 一致性检查失败：同类型组件必须走同一原语（详见 docs/ui-components.md「同类型一致性契约」）。');
for (const [rule, items] of byRule) {
	console.error('');
	console.error('[' + rule + '] ' + RULES[rule] + ' —— ' + items.length + ' 处');
	for (const v of items.slice(0, 40)) console.error('  ' + v.file + ':' + v.line + '  ' + v.detail);
	if (items.length > 40) console.error('  ... 其余 ' + (items.length - 40) + ' 处省略');
}
console.error('');
console.error('修复：迁移到对应 forms 原语；合理例外登记 scripts/ui-consistency-allowlist.json（rule+file+reason）。');
process.exit(1);

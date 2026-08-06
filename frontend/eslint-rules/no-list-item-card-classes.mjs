/**
 * 自定义 ESLint 规则：ListItem 调用方禁止叠加行级卡片类（多边框防线）。
 *
 * 背景：ListItem 的 joined 行（默认 tone）由 ListContainer 的 divide 线提供行间
 * 分隔、由 hover-fill 提供激活反馈，圆角外框只允许出现在容器级
 * （ListContainer bordered 唯一外框）或 tone="card"（独立卡片容器 + gap 分隔）。
 * 调用方在 class/style 里叠加 rounded/border/shadow/ring 会使每个 item 呈现
 * 独立圆角矩形边缘（历史 bug：/map 6 处、ChatListItem 1 处）。
 *
 * 基于 svelte-eslint-parser 的模板 AST 精确分析 <ListItem> 元素（Svelte 5 结构）：
 *   - 组件元素 node.name 为 Identifier 节点，属性位于 startTag.attributes
 *   - class/style 属性值由 SvelteLiteral（字面量段）与 SvelteMustacheTag
 *     （动态表达式段）交替组成；字面量段合并后扫描卡片类，表达式段跳过
 *     （真实调用点静态段已全覆盖）
 *
 * 配套防线：ListItem.svelte dev 模式运行时警告 + AGENTS.md 列表骨架契约。
 * 注册见 eslint.config.mjs；运行：bun run lint（已挂入 bun run check 与 pre-commit）。
 */
const CARD_CLASS_RE = /\b(?:rounded|border|shadow|ring)[\w-]*/g;
const CARD_STYLE_RE = /(?:^|[;"'}\s])(?:border|box-shadow|outline)\s*:/g;

/** @type {import('eslint').Rule.RuleModule} */
export const noListItemCardClasses = {
	meta: {
		type: 'problem',
		docs: {
			description: 'ListItem 调用方禁止叠加行级卡片类（边框/圆角/阴影）',
			recommended: true
		},
		messages: {
			cardClass:
				'ListItem class 含行级卡片类「{{cls}}」——joined 行禁止叠加边框/圆角/阴影，卡片化请用 tone="card"（独立卡片容器）或 ListContainer bordered（容器级唯一外框）',
			cardStyle: 'ListItem style 含行级边框声明「{{prop}}」'
		},
		schema: []
	},
	create(context) {
		return {
			/** @param {any} node svelte-eslint-parser 的 SvelteElement 节点 */
			SvelteElement(node) {
				// Svelte 5：组件元素 name 为 Identifier 节点
				if (node.name?.type !== 'Identifier' || node.name.name !== 'ListItem') return;
				for (const attr of node.startTag?.attributes ?? []) {
					if (attr.type !== 'SvelteAttribute') continue;
					// 属性值 = SvelteLiteral 字面量段 + SvelteMustacheTag 表达式段；只取字面量段
					const text = (attr.value ?? []).map((v) => v.value ?? '').join('');
					// Svelte 5：attr.key 为 SvelteName 节点（name 属性为属性名）
					const attrName = attr.key?.name ?? attr.key;
					if (attrName === 'class') {
						for (const m of text.matchAll(CARD_CLASS_RE)) {
							context.report({ node: attr, messageId: 'cardClass', data: { cls: m[0] } });
						}
					} else if (attrName === 'style') {
						for (const m of text.matchAll(CARD_STYLE_RE)) {
							context.report({ node: attr, messageId: 'cardStyle', data: { prop: m[0].trim() } });
						}
					}
				}
			}
		};
	}
};

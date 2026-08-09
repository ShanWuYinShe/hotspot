/**
 * 自定义 ESLint 规则：纯图标按钮必须有可访问名称（aria-label / title / 可见文本）。
 *
 * 背景：icon-only `<button>` 若没有 aria-label/title，屏幕阅读器读不到任何内容；
 * 同时纯图标按钮触控面积通常偏小，也容易漏掉 title 提示。
 * 本规则基于 svelte-eslint-parser 模板 AST：
 *   - 只检查原生 <button>（组件按钮由组件自身契约负责）
 *   - 存在 aria-label / title 属性即视为有名称
 *   - 子节点含可见文本（含 mustache 表达式）视为有名称
 *   - 子节点含 snippet 渲染（{@render children()}）或 <slot> 时跳过——
 *     实际名称由调用方 snippet 决定，无法静态判定
 *
 * 注册见 eslint.config.mjs；运行：bun run lint（已挂入 bun run check 与 pre-commit）。
 */

/** @type {import('eslint').Rule.RuleModule} */
export const iconButtonHasLabel = {
	meta: {
		type: 'problem',
		docs: {
			description: '纯图标按钮必须提供 aria-label / title 或可见文本',
			recommended: true
		},
		messages: {
			missingLabel:
				'图标按钮缺少可访问名称：请添加 aria-label（或 title）与可见文本，例如 aria-label="关闭"'
		},
		schema: []
	},
	create(context) {
		function hasLabelAttr(node) {
			for (const attr of node.startTag?.attributes ?? []) {
				if (attr.type !== 'SvelteAttribute' && attr.type !== 'SvelteShorthandAttribute') continue;
				const name = attr.type === 'SvelteAttribute' ? (attr.key?.name ?? attr.key) : attr.key?.name;
				if (name === 'aria-label' || name === 'title') return true;
			}
			return false;
		}

		function hasVisibleText(node) {
			for (const child of node.children ?? []) {
				if (child.type === 'SvelteRenderTag' || child.type === 'SvelteSlot') return true;
				if (child.type === 'SvelteText' && child.value.trim()) return true;
				if (child.type === 'SvelteMustacheTag') return true;
				// 递归检查内嵌元素（如 <span> 包裹的图标或文字）
				if (child.type === 'SvelteElement' && hasVisibleText(child)) return true;
			}
			return false;
		}

		return {
			/** @param {any} node svelte-eslint-parser 的 SvelteElement 节点 */
			SvelteElement(node) {
				if (node.name?.type !== 'Identifier' || node.name.name !== 'button') return;
				if (hasLabelAttr(node) || hasVisibleText(node)) return;
				context.report({ node, messageId: 'missingLabel' });
			}
		};
	}
};

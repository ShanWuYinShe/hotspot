/**
 * 自定义 ESLint 规则：禁止裸 `outline-none`（无焦点环替代）。
 *
 * 背景：Tailwind v4 的 `outline-none`（utilities 层）会压掉 base.css 的全局
 * `button:focus-visible` 轮廓，导致键盘用户看不到焦点。正确做法是使用
 * `focus-ring`（focus-visible 时 accent outline）或
 * `focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-default`。
 * 本规则只拦截 class 属性中独立的 `outline-none` token，不拦截
 * `focus-visible:outline-none`（带 focus-visible 变体时说明有配套焦点样式），
 * 或同 class 内已含 `focus-visible:ring-*` / `focus-ring` / `focus-within`
 * （说明有替代焦点环，裸 outline-none 只是压掉默认轮廓）。
 *
 * 注册见 eslint.config.mjs；运行：bun run lint（已挂入 bun run check 与 pre-commit）。
 */

/** @type {import('eslint').Rule.RuleModule} */
export const noPlainOutlineNone = {
	meta: {
		type: 'problem',
		docs: {
			description: '禁止裸 outline-none，必须提供可见焦点环',
			recommended: true
		},
		messages: {
			plainOutline:
				'裸 outline-none 会压掉全局焦点轮廓：请改用 focus-ring，或补充 focus-visible:ring-2 focus-visible:ring-accent-default'
		},
		schema: []
	},
	create(context) {
		return {
			/** @param {any} node svelte-eslint-parser 的 SvelteElement 节点 */
			SvelteElement(node) {
				for (const attr of node.startTag?.attributes ?? []) {
					if (attr.type !== 'SvelteAttribute') continue;
					const attrName = attr.key?.name ?? attr.key;
					if (attrName !== 'class') continue;
					const text = (attr.value ?? []).map((v) => v.value ?? '').join('');
					if (/focus-visible:ring|focus-visible:outline|focus-ring|focus-within/.test(text)) continue;
					for (const token of text.split(/\s+/)) {
						if (token === 'outline-none') {
							context.report({ node: attr, messageId: 'plainOutline' });
						}
					}
				}
			}
		};
	}
};

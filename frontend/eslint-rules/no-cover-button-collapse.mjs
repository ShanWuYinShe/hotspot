/**
 * 自定义 ESLint 规则:行内按钮「自动抬升」类必须排除覆盖按钮自身(:not([data-cover]))。
 *
 * 背景:带整行/整卡覆盖按钮的组件(ListItem/NearbyImageCard)用
 * [&_button]:relative [&_button]:z-N 把行内既有按钮抬到覆盖按钮之上
 * (axe nested-interactive 防线)。但 [&_button]:relative 与覆盖按钮自身的
 * absolute inset-0 同 specificity,且 Tailwind 工具类生成顺序 .relative 在
 * .absolute 之后——后者胜出,覆盖按钮塌缩 0×0,真实点击全部落在内容上,
 * 整行/整卡「点了没反应」(线上实测 P0:ListItem 2026-09、NearbyImageCard
 * 2026-09;jsdom 不应用样式表,测试只能靠类串断言,见 NearbyImageCard.test.ts)。
 *
 * 覆盖按钮统一带 data-cover 属性;本规则禁止出现未排除的
 * [&_button]:relative / [&_button]:z-N 形态(a/input/select 抬升无需排除,
 * 不在检测范围)。解析方式与 no-list-item-card-classes 相同:
 * svelte-eslint-parser 模板 AST 的 class 属性字面量段。
 */
const FORBIDDEN_RE = /\[&_button\]:(?:relative|z-\d+)\b/g;

/** @type {import('eslint').Rule.RuleModule} */
export const noCoverButtonCollapse = {
	meta: {
		type: 'problem',
		docs: {
			description: '按钮抬升类必须用 :not([data-cover]) 排除覆盖按钮自身,防 relative 覆盖 absolute 塌缩',
			recommended: true
		},
		messages: {
			forbidden:
				'class 含未排除覆盖按钮的抬升类「{{cls}}」——它会覆盖覆盖按钮的 absolute(塌缩 0×0,点击失效),必须写 [&_button:not([data-cover])]:…'
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
					for (const m of text.matchAll(FORBIDDEN_RE)) {
						context.report({ node: attr, messageId: 'forbidden', data: { cls: m[0] } });
					}
				}
			}
		};
	}
};

/**
 * ESLint flat config（v10）——最小化接入：只注册项目自定义规则，不启用任何
 * 默认规则集，存量代码零噪音。后续要引入通用 JS/TS 规范时在此追加。
 *
 * 当前规则：
 *   hotspot/no-list-item-card-classes —— ListItem 调用方禁止行级卡片类（多边框防线）
 *
 * 运行：bun run lint（已挂入 bun run check；check 经仓库根 .husky/pre-commit
 * 在每次提交前自动执行，见 frontend/scripts/setup-git-hooks.sh）。
 */
import svelteParser from 'svelte-eslint-parser';
import { parser as tsParser } from 'typescript-eslint';
import { noListItemCardClasses } from './eslint-rules/no-list-item-card-classes.mjs';
import { iconButtonHasLabel } from './eslint-rules/icon-button-has-label.mjs';
import { noPlainOutlineNone } from './eslint-rules/no-plain-outline-none.mjs';
import { noCoverButtonCollapse } from './eslint-rules/no-cover-button-collapse.mjs';

export default [
	{
		ignores: ['node_modules/**', 'build/**', '.svelte-kit/**', 'static/**']
	},
	{
		files: ['src/**/*.svelte'],
		languageOptions: {
			parser: svelteParser,
			parserOptions: {
				// <script lang="ts"> 段交给 typescript-eslint parser 解析
				parser: tsParser,
				extraFileExtensions: ['.svelte']
			}
		},
		plugins: {
			hotspot: {
				rules: {
					'no-list-item-card-classes': noListItemCardClasses,
					'icon-button-has-label': iconButtonHasLabel,
					'no-plain-outline-none': noPlainOutlineNone,
					'no-cover-button-collapse': noCoverButtonCollapse
				}
			}
		},
		rules: {
			'hotspot/no-list-item-card-classes': 'error',
			'hotspot/icon-button-has-label': 'error',
			'hotspot/no-plain-outline-none': 'error',
			'hotspot/no-cover-button-collapse': 'error'
		}
	}
];

// ListItem 契约测试：role/aria、点击回调、选中态。
import { describe, it, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ListItem from '../ListItem.svelte';

describe('ListItem', () => {
	it('可点击时提供 role=button 与 aria-label', () => {
		render(ListItem, {
			props: { children: (() => '内容') as never, onClick: () => {}, ariaLabel: '查看鹿鸣' }
		});
		expect(screen.getByRole('button', { name: '查看鹿鸣' })).toBeTruthy();
	});

	it('点击触发 onClick', async () => {
		const onClick = vi.fn();
		render(ListItem, {
			props: { children: (() => '内容') as never, onClick, ariaLabel: '行' }
		});
		await fireEvent.click(screen.getByRole('button'));
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	it('pressed 时 aria-pressed=true', () => {
		render(ListItem, {
			props: { children: (() => '内容') as never, onClick: () => {}, ariaLabel: '行', pressed: true }
		});
		expect(screen.getByRole('button').getAttribute('aria-pressed')).toBe('true');
	});

	it('根节点带稳定 data-list-item 标记（hover 次级文字对比度规则的作用域锚点）', () => {
		const { container } = render(ListItem, {
			props: { children: (() => '内容') as never, onClick: () => {}, ariaLabel: '行' }
		});
		const row = container.querySelector('[data-list-item]');
		expect(row).not.toBeNull();
		expect(row?.getAttribute('role')).toBe('group');
	});

	it('hover/focus 次级文字升到 text-primary 的规则存在且排除选中态（源码契约）', () => {
		// jsdom 不应用 hover 伪类 / 组件样式表，无法做渲染断言；按 perf-contract.test.ts
		// 同口径做源码级契约：规则一旦被删（对比度回退到 3.2–4.43）即失败。
		const src = readFileSync('src/components/list/ListItem.svelte', 'utf8');
		expect(src).toMatch(
			/\[data-list-item\]:not\(\[data-active='true'\]\):hover \.text-text-secondary/
		);
		expect(src).toMatch(
			/\[data-list-item\]:not\(\[data-active='true'\]\):focus-within \.text-text-secondary/
		);
		expect(src).toMatch(/color: var\(--cst-text-primary\)/);
	});

	it('tone=card 生成独立卡片行类', () => {
		const { container } = render(ListItem, {
			props: { children: (() => '内容') as never, tone: 'card', ariaLabel: '行' }
		});
		const row = container.querySelector('[aria-label="行"]') as HTMLElement;
		expect(row).not.toBeNull();
		expect(row.className).toContain('rounded-md');
	});
});

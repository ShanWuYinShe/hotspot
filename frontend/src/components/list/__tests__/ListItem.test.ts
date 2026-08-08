// ListItem 契约测试：role/aria、点击回调、选中态。
import { describe, it, expect, vi } from 'vitest';
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

	it('tone=card 生成独立卡片行类', () => {
		const { container } = render(ListItem, {
			props: { children: (() => '内容') as never, tone: 'card', ariaLabel: '行' }
		});
		const row = container.querySelector('[aria-label="行"]') as HTMLElement;
		expect(row).not.toBeNull();
		expect(row.className).toContain('rounded-md');
	});
});

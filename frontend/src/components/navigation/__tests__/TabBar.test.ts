// TabBar 契约测试：下划线分类 Tab 的 aria-pressed、指示条与切换回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import TabBar, { type TabItem } from '../TabBar.svelte';

const items: TabItem[] = [
	{ id: 'moments', label: '动态', count: 3 },
	{ id: 'groups', label: '群组' }
];

describe('TabBar', () => {
	it('渲染全部 tab 与计数', () => {
		render(TabBar, { props: { items, value: 'moments', onChange: () => {} } });
		expect(screen.getByRole('button', { name: /动态/ })).toBeTruthy();
		expect(screen.getByRole('button', { name: /群组/ })).toBeTruthy();
		expect(screen.getByText('3')).toBeTruthy();
	});

	it('激活项 aria-pressed=true，非激活为 false', () => {
		render(TabBar, { props: { items, value: 'moments', onChange: () => {} } });
		expect(screen.getByRole('button', { name: /动态/ }).getAttribute('aria-pressed')).toBe('true');
		expect(screen.getByRole('button', { name: /群组/ }).getAttribute('aria-pressed')).toBe('false');
	});

	it('点击 tab 触发 onChange 并传入 id', async () => {
		const onChange = vi.fn();
		render(TabBar, { props: { items, value: 'moments', onChange } });
		await fireEvent.click(screen.getByRole('button', { name: /群组/ }));
		expect(onChange).toHaveBeenCalledWith('groups');
	});
});

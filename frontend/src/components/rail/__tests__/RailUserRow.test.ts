// RailUserRow 契约测试：头像/名称/副标渲染与点击回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import RailUserRow from '../RailUserRow.svelte';

describe('RailUserRow', () => {
	it('渲染头像 alt、名称与副标', () => {
		render(RailUserRow, {
			props: { avatar: 'https://example.com/a.jpg', name: '鹿鸣', sub: '0.4km' }
		});
		expect(screen.getByAltText('鹿鸣')).toBeTruthy();
		expect(screen.getByText('鹿鸣')).toBeTruthy();
		expect(screen.getByText('0.4km')).toBeTruthy();
	});

	it('无 sub 时不渲染副标', () => {
		render(RailUserRow, { props: { avatar: 'x', name: '鹿鸣' } });
		expect(screen.queryByText('0.4km')).toBeNull();
	});

	it('点击触发 onClick', async () => {
		const onClick = vi.fn();
		render(RailUserRow, {
			props: { avatar: 'x', name: '鹿鸣', onClick }
		});
		await fireEvent.click(screen.getByRole('button'));
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	// 回归守卫：可点行统一走 ListItem，行交互（hover 填充/圆角/覆盖按钮）由行原语承载；
	// trailing 动作在行外，不会被高亮吞掉。
	it('可点行由 ListItem 承载交互（data-list-item + 覆盖按钮）', () => {
		render(RailUserRow, { props: { avatar: 'x', name: '鹿鸣', onClick: vi.fn() } });
		const btn = screen.getByRole('button');
		expect(btn.getAttribute('data-cover')).not.toBeNull();
		expect(btn.closest('[data-list-item]')).toBeTruthy();
	});

	it('不可点击的行不渲染覆盖按钮', () => {
		render(RailUserRow, { props: { avatar: 'x', name: '鹿鸣' } });
		expect(screen.queryByRole('button')).toBeNull();
	});
});

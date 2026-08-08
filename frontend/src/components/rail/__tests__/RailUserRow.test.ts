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
});

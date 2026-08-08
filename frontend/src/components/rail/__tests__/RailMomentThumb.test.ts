// RailMomentThumb 契约测试：缩略图/标题/元信息渲染与点击回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import RailMomentThumb from '../RailMomentThumb.svelte';

describe('RailMomentThumb', () => {
	it('渲染图片 alt、标题与元信息', () => {
		render(RailMomentThumb, {
			props: { image: 'https://example.com/m.jpg', title: '清晨公园', meta: '0.4km · 1分钟前' }
		});
		expect(screen.getByAltText('清晨公园')).toBeTruthy();
		expect(screen.getByText('清晨公园')).toBeTruthy();
		expect(screen.getByText('0.4km · 1分钟前')).toBeTruthy();
	});

	it('点击触发 onClick', async () => {
		const onClick = vi.fn();
		render(RailMomentThumb, {
			props: { image: 'x', title: '清晨公园', meta: '0.4km', onClick }
		});
		await fireEvent.click(screen.getByRole('button'));
		expect(onClick).toHaveBeenCalledTimes(1);
	});
});

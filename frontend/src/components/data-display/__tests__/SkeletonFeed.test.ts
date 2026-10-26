// SkeletonFeed 契约测试：三种形态、栅格类透传、role=status 播报。
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import SkeletonFeed from '../SkeletonFeed.svelte';

describe('SkeletonFeed', () => {
	it('feed 形态渲染作者行 + 媒体块 + 操作行占位', () => {
		const { container } = render(SkeletonFeed, {
			props: { variant: 'feed', count: 2, class: 'grid grid-cols-2 gap-4', label: '加载动态…' }
		});
		const status = screen.getByRole('status');
		expect(status.getAttribute('aria-label')).toBe('加载动态…');
		expect(status.className).toContain('grid');
		expect(container.querySelectorAll('article').length).toBe(2);
	});

	it('card 形态支持水平紧凑卡（h-32）', () => {
		const { container } = render(SkeletonFeed, { props: { variant: 'card', count: 1, horizontal: true } });
		const card = container.querySelector('article');
		expect(card?.className).toContain('h-32');
		expect(card?.className).toContain('flex');
	});

	it('portrait 形态为 3:4 竖卡', () => {
		const { container } = render(SkeletonFeed, { props: { variant: 'portrait', count: 1 } });
		expect(container.querySelector('article')?.className).toContain('aspect-[3/4]');
	});
});

// NearbyImageCard 契约测试：图片/占位渲染、点击与键盘触发。
import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import NearbyImageCard from '../NearbyImageCard.svelte';

describe('NearbyImageCard', () => {
	it('有图片时渲染 img 并带 alt', () => {
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: 'https://example.com/a.jpg', imageAlt: '群组封面' }
		});
		expect(container.querySelector('img')?.getAttribute('alt')).toBe('群组封面');
	});

	it('无图片时渲染渐变占位层', () => {
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: '', placeholder: (() => null) as never }
		});
		expect(container.querySelector('.surface-accent-gradient')).not.toBeNull();
		expect(container.querySelector('img')).toBeNull();
	});

	it('onClick 提供时 role=button 并点击触发', async () => {
		const onClick = vi.fn();
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: 'x', onClick }
		});
		const card = container.querySelector('article') as HTMLElement;
		expect(card.getAttribute('role')).toBe('button');
		await fireEvent.click(card);
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	it('键盘 Enter 触发 onClick', async () => {
		const onClick = vi.fn();
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: 'x', onClick }
		});
		const card = container.querySelector('article') as HTMLElement;
		await fireEvent.keyDown(card, { key: 'Enter' });
		expect(onClick).toHaveBeenCalledTimes(1);
	});
});

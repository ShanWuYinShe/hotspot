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

	it('onClick 提供时由覆盖按钮承载点击（根不再持 button 语义）', async () => {
		const onClick = vi.fn();
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: 'x', onClick }
		});
		const card = container.querySelector('article') as HTMLElement;
		expect(card.getAttribute('role')).toBeNull();
		const cover = card.querySelector('[data-cover]') as HTMLElement;
		expect(cover.tagName).toBe('BUTTON');
		await fireEvent.click(cover);
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	it('覆盖按钮为原生 button（键盘 Enter/Space 原生激活）', async () => {
		const onClick = vi.fn();
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: 'x', onClick }
		});
		const cover = container.querySelector('[data-cover]') as HTMLElement;
		expect(cover.tagName).toBe('BUTTON');
		expect((cover as HTMLButtonElement).type).toBe('button');
		await fireEvent.click(cover);
		expect(onClick).toHaveBeenCalledTimes(1);
	});
});

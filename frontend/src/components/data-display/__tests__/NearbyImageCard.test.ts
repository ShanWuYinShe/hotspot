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

	it('按钮抬升类必须排除覆盖按钮自身（:not([data-cover])）', () => {
		// 回归防线:jsdom 不应用 Tailwind 样式表,塌缩(0×0)只能靠类串断言拦截——
		// [&_button]:relative 与覆盖按钮 absolute 同 specificity 且 .relative 后生成,
		// 一旦漏排除,真实浏览器整卡点击失效(线上实测 P0)
		const { container } = render(NearbyImageCard, {
			props: { imageSrc: 'x', onClick: () => {} }
		});
		const root = container.querySelector('article') as HTMLElement;
		expect(root.className).toContain('[&_button:not([data-cover])]:relative');
		expect(root.className).toContain('[&_button:not([data-cover])]:z-20');
		expect(root.className).not.toMatch(/\[&_button\]:(?:relative|z-\d+)/);
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

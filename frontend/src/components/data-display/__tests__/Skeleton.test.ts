// Skeleton 基元契约测试：恒 aria-hidden（不进入读屏/焦点），几何类由调用方提供。
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import Skeleton from '../Skeleton.svelte';

describe('Skeleton', () => {
	it('渲染装饰块且对读屏隐藏', () => {
		const { container } = render(Skeleton, { props: { class: 'size-10 rounded-full' } });
		const el = container.querySelector('[data-skeleton]');
		expect(el).not.toBeNull();
		expect(el?.getAttribute('aria-hidden')).toBe('true');
		expect(el?.className).toContain('skeleton-block');
		expect(el?.className).toContain('rounded-full');
	});

	it('支持内联布局几何（文本条高度/宽度）', () => {
		const { container } = render(Skeleton, {
			props: { class: 'rounded-sm', style: 'height: 14px; width: 55%;' }
		});
		const el = container.querySelector('[data-skeleton]') as HTMLElement;
		expect(el.style.height).toBe('14px');
		expect(el.style.width).toBe('55%');
	});
});

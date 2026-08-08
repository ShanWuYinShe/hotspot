// OnlineDot 契约测试：尺寸档位与颜色 token 覆盖。
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import OnlineDot from '../OnlineDot.svelte';

describe('OnlineDot', () => {
	it('默认 md 尺寸（10px）并使用在线状态 token', () => {
		const { container } = render(OnlineDot, { props: {} });
		const dot = container.firstElementChild as HTMLElement;
		expect(dot.style.width).toBe('10px');
		expect(dot.style.height).toBe('10px');
		expect(dot.style.background).toContain('var(--cst-status-online)');
	});

	it('size=lg 应用 12px', () => {
		const { container } = render(OnlineDot, { props: { size: 'lg' } });
		const dot = container.firstElementChild as HTMLElement;
		expect(dot.style.width).toBe('12px');
	});

	it('color 覆盖默认背景', () => {
		const { container } = render(OnlineDot, {
			props: { color: 'var(--cst-accent-default)' }
		});
		const dot = container.firstElementChild as HTMLElement;
		expect(dot.style.background).toContain('var(--cst-accent-default)');
	});
});

// SkeletonList 契约测试：role=status 播报、行数与前导圆形档、ListContainer 内自动补 listitem。
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import SkeletonList from '../SkeletonList.svelte';

describe('SkeletonList', () => {
	it('role=status + aria-busy 播报一次，行数与 rows 一致', () => {
		const { container } = render(SkeletonList, { props: { rows: 3, label: '加载会话…' } });
		const status = screen.getByRole('status');
		expect(status.getAttribute('aria-label')).toBe('加载会话…');
		expect(status.getAttribute('aria-busy')).toBe('true');
		expect(container.querySelectorAll('[data-skeleton-row]').length).toBe(3);
	});

	it('前导圆形直径走 36/40/44/48 档位', () => {
		const { container } = render(SkeletonList, { props: { rows: 1, leading: 44 } });
		const circle = container.querySelector('[data-skeleton-row] [data-skeleton]');
		expect(circle?.className).toContain('size-11');
	});

	it('裸用不产生 listitem（列表上下文语义由 ListContainer 子树自动补足，浏览器实测覆盖）', () => {
		const { container } = render(SkeletonList, { props: { rows: 2 } });
		expect(container.querySelector('[role="listitem"]')).toBeNull();
	});
});

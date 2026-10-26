// ListContainer 契约测试：bordered 形态差异与 aria-label。
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import ListContainer from '../ListContainer.svelte';

describe('ListContainer', () => {
	it('bordered=true 生成内容软岛（Round 3：无描边 + 表面 + 投影 + 圆角 20）', () => {
		const { container } = render(ListContainer, {
			props: { children: (() => '行') as never, bordered: true, ariaLabel: '好友列表' }
		});
		const box = container.querySelector('[aria-label="好友列表"]') as HTMLElement;
		expect(box.className).toContain('rounded-lg');
		expect(box.className).toContain('bg-surface-island');
		expect(box.className).toContain('shadow-xs');
		// 去明显边界：软岛不允许 1px 全框描边
		expect(/(^|\s)border(\s|$)/.test(box.className)).toBe(false);
	});

	it('bordered=false 无外框类', () => {
		const { container } = render(ListContainer, {
			props: { children: (() => '行') as never, ariaLabel: '会话列表' }
		});
		const box = container.querySelector('[aria-label="会话列表"]') as HTMLElement;
		expect(box.className).not.toContain('rounded-md');
		expect(/(^|\s)border(\s|$)/.test(box.className)).toBe(false);
	});
});

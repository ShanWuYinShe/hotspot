// ListContainer 契约测试：bordered 形态差异与 aria-label。
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import ListContainer from '../ListContainer.svelte';

describe('ListContainer', () => {
	it('bordered=true 生成唯一外框类', () => {
		const { container } = render(ListContainer, {
			props: { children: (() => '行') as never, bordered: true, ariaLabel: '好友列表' }
		});
		const box = container.querySelector('[aria-label="好友列表"]') as HTMLElement;
		expect(box.className).toContain('rounded-md');
		expect(box.className).toContain('border');
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

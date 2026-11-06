// NavTab 契约测试：aria 属性、点击回调、高质感 Content-First 选中态与徽标
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { Home } from '@lucide/svelte';
import NavTab from '../NavTab.svelte';

describe('NavTab', () => {
	it('渲染 aria-label 与文字内容', () => {
		render(NavTab, {
			props: {
				label: '首页',
				isActive: false,
				onClick: vi.fn(),
				Icon: Home
			}
		});

		const btn = screen.getByRole('button', { name: '首页' });
		expect(btn).toBeTruthy();
		expect(btn.getAttribute('aria-current')).toBeNull();
		expect(btn.textContent).toContain('首页');
	});

	it('点击触发 onClick 回调', async () => {
		const onClick = vi.fn();
		render(NavTab, {
			props: {
				label: '首页',
				isActive: false,
				onClick,
				Icon: Home
			}
		});

		await fireEvent.click(screen.getByRole('button', { name: '首页' }));
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	it('选中态：aria-current 为 page，废除笨重方框，图标与文字统一步入 text-accent-default', () => {
		const { container } = render(NavTab, {
			props: {
				label: '首页',
				isActive: true,
				onClick: vi.fn(),
				Icon: Home
			}
		});

		const btn = screen.getByRole('button', { name: '首页' });
		expect(btn.getAttribute('aria-current')).toBe('page');

		// 绝不包含笨重的方块色块与描边
		expect(container.innerHTML).not.toContain('bg-accent-soft');
		expect(container.innerHTML).not.toContain('ring-accent-border');

		// 统一的品牌色高亮
		const text = container.querySelector('span.font-semibold');
		expect(text).toBeTruthy();
		expect(text?.className).toContain('text-accent-default');

		// 底部精致微指示条展开
		const indicator = container.querySelector('[data-nav-indicator]');
		expect(indicator).toBeTruthy();
		expect(indicator?.className).toContain('opacity-100');
		expect(indicator?.className).toContain('scale-100');
		expect(indicator?.className).toContain('bg-accent-default');
	});

	it('未选中态：底部微指示条占位隐退，文字与图标呈现次级中性色', () => {
		const { container } = render(NavTab, {
			props: {
				label: '附近',
				isActive: false,
				onClick: vi.fn(),
				Icon: Home
			}
		});

		const indicator = container.querySelector('[data-nav-indicator]');
		expect(indicator).toBeTruthy();
		expect(indicator?.className).toContain('opacity-0');
		expect(indicator?.className).toContain('scale-50');

		const text = container.querySelector('span.font-medium');
		expect(text).toBeTruthy();
		expect(text?.className).toContain('text-text-secondary');
	});

	it('显示未读消息气泡', () => {
		const { container } = render(NavTab, {
			props: {
				label: '消息',
				isActive: false,
				onClick: vi.fn(),
				Icon: Home,
				count: 5
			}
		});

		expect(container.textContent).toContain('5');
	});
});

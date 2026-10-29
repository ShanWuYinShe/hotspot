// PageHeader 布局契约测试：标题渲染、返回按钮显隐、弹性溢出契约。
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import PageHeader from '../PageHeader.svelte';

describe('PageHeader', () => {
	it('渲染标题', () => {
		render(PageHeader, { props: { title: '附近动态' } });
		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('附近动态');
	});

	it('onBack 存在时（非桌面）显示返回按钮', () => {
		render(PageHeader, { props: { title: '设置', onBack: () => {} } });
		expect(screen.getByLabelText('返回')).toBeTruthy();
	});

	it('未传 onBack 时不显示返回按钮', () => {
		render(PageHeader, { props: { title: '首页' } });
		expect(screen.queryByLabelText('返回')).toBeNull();
	});

	it('标题列恒为弹性列（窄屏下先收缩让位，trailing 不被压缩）', () => {
		render(PageHeader, { props: { title: '首页', subtitle: '发现周边有趣的人和事' } });
		const titleCol = screen.getByRole('heading', { level: 1 }).parentElement;
		expect(titleCol?.classList.contains('flex-1')).toBe(true);
		expect(titleCol?.classList.contains('min-w-0')).toBe(true);
	});

	it('副标题文本单行截断（窄屏下尾字省略，不折行撑高）', () => {
		render(PageHeader, { props: { title: '首页', subtitle: '发现周边有趣的人和事' } });
		const subtitleText = screen.getByText('发现周边有趣的人和事');
		// 文本包在 truncate span 内，图标与文字行可独立收缩
		expect(subtitleText.tagName).toBe('SPAN');
		expect(subtitleText.classList.contains('truncate')).toBe(true);
		expect(subtitleText.parentElement?.classList.contains('min-w-0')).toBe(true);
	});
});

describe('PageHeader floating 悬浮工具条（2026-10-29 详情页）', () => {
	it('整行是一块悬浮玻璃面板：返回 / 标题 / 更多同处一个容器，而不是孤立圆钮', () => {
		const { container } = render(PageHeader, {
			props: { title: '瞬间详情', variant: 'floating', onBack: () => {}, showBackOnDesktop: false }
		});
		const header = container.querySelector('header') as HTMLElement;
		expect(header.className).not.toContain('page-header-scroll');
		expect(header.className).toContain('pointer-events-none');
		expect(header.className).toContain('sticky');
		const bar = container.querySelector('[data-floating-toolbar]') as HTMLElement;
		expect(bar).toBeTruthy();
		// 整行共享同一表面：圆角 + 半透明底 + 毛玻璃 + 细边 + 轻阴影
		expect(bar.className).toContain('rounded-md');
		expect(bar.className).toContain('bg-surface-primary/85');
		expect(bar.className).toContain('backdrop-blur-md');
		expect(bar.className).toContain('border-border-default/60');
		expect(bar.className).toContain('shadow-raised-sm');
		// 标题与返回都在同一面板内，且标题可见（不再 sr-only）
		const title = screen.getByRole('heading', { level: 1 });
		expect(title.className).not.toContain('sr-only');
		expect(bar.contains(title)).toBe(true);
		expect(bar.contains(screen.getByLabelText('返回'))).toBe(true);
	});

	it('默认变体仍是可见标题 + 可滚动 glass 页头', () => {
		const { container } = render(PageHeader, { props: { title: '附近动态' } });
		const header = container.querySelector('header') as HTMLElement;
		expect(header.className).toContain('page-header-scroll');
		expect(container.querySelector('[data-floating-toolbar]')).toBeNull();
		expect(screen.getByRole('heading', { level: 1 }).className).not.toContain('sr-only');
	});
});

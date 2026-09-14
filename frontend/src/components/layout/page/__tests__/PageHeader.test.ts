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

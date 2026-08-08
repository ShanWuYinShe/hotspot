// PageHeader 布局契约测试：标题渲染、返回按钮显隐、below 附属区。
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
});

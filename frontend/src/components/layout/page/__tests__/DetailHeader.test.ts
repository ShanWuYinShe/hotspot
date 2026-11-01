import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import DetailHeader from '../DetailHeader.svelte';

describe('DetailHeader', () => {
	it('默认初始态：渲染返回胶囊和桌面实体面包屑', () => {
		const { container } = render(DetailHeader, {
			props: {
				category: '发现动态',
				entityName: '@Mika 的瞬间',
				author: {
					name: 'Mika',
					avatar: '/avatars/mika.jpg',
					location: '回龙窝街区',
					time: '10分钟前'
				}
			}
		});

		// 返回按钮存在
		const backBtn = screen.getByLabelText('返回');
		expect(backBtn).toBeTruthy();

		// 面包屑存在且包含分类与实体名
		const breadcrumb = container.querySelector('[data-breadcrumb]');
		expect(breadcrumb).toBeTruthy();
		expect(breadcrumb?.textContent).toContain('发现动态');
		expect(breadcrumb?.textContent).toContain('@Mika 的瞬间');

		// 顶端初始态：中间微名片不显示（留白给大图）
		expect(container.querySelector('[data-detail-header-center]')).toBeNull();
	});

	it('点击返回按钮触发 onBack 回调', async () => {
		const onBack = vi.fn();
		render(DetailHeader, { props: { onBack } });

		const backBtn = screen.getByLabelText('返回');
		await fireEvent.click(backBtn);
		expect(onBack).toHaveBeenCalledTimes(1);
	});

	it('支持自定义返回文案', () => {
		render(DetailHeader, { props: { backLabel: '返回上一页' } });
		expect(screen.getByLabelText('返回上一页')).toBeTruthy();
	});

	it('结构契约：外层 pointer-events-none，胶囊条 pointer-events-auto', () => {
		const { container } = render(DetailHeader, {
			props: {
				category: '发现动态',
				entityName: '瞬间详情'
			}
		});

		const header = container.querySelector('header');
		expect(header?.classList.contains('pointer-events-none')).toBe(true);

		const leftPill = container.querySelector('[data-detail-header-left]');
		expect(leftPill?.classList.contains('pointer-events-auto')).toBe(true);
		expect(leftPill?.classList.contains('rounded-full')).toBe(true);
		expect(leftPill?.classList.contains('backdrop-blur-md')).toBe(true);
	});

	it('无封面时（纯文字/资料页）：顶部呈现语义标题并挂载毛玻璃底色', () => {
		const { container } = render(DetailHeader, {
			props: {
				category: '发现动态',
				entityName: '纯文字瞬间',
				hasCover: false
			}
		});

		const header = container.querySelector('header');
		expect(header?.className).toContain('bg-surface-page/85');
		expect(header?.className).toContain('backdrop-blur-md');
		expect(container.textContent).toContain('纯文字瞬间');
	});
});

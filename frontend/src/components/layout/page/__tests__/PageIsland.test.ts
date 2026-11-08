import { createRawSnippet } from 'svelte';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import PageIsland from '../PageIsland.svelte';

// PageIsland 组件契约（docs/design-system.md §22）：
//   - 紧凑胶囊：返回 · 标题/作者身份 + meta · 行动区（≤2）· 最右侧阅读进度圆环
//   - 点击标题回锚点/进入作者主页；左侧返回仅在注册了 onBack 时渲染
//   - 圆环弧长 = progress（0–100，越界收敛），纯展示（aria-hidden）

describe('PageIsland（桌面端页面锚点岛）', () => {
	it('渲染标题、meta 与阅读进度圆环，点击标题触发回锚点', async () => {
		const onTitleClick = vi.fn();
		const { container } = render(PageIsland, {
			props: { title: '附近的人', meta: '20 人 · 5km 内', onTitleClick, progress: 42 }
		});
		const island = container.querySelector('[data-desktop-dynamic-island]') as HTMLElement;
		expect(island).not.toBeNull();
		expect(island.getAttribute('role')).toBe('group');
		expect(island.textContent).toContain('附近的人');
		expect(island.textContent).toContain('20 人 · 5km 内');

		const ring = island.querySelector('[data-island-progress-ring]') as HTMLElement;
		expect(ring).not.toBeNull();
		expect(ring.dataset.islandProgress).toBe('42');
		// 圆环是 SVG 双圈：底轨 + accent 弧；弧的 dashoffset 随进度减小
		expect(ring.querySelectorAll('svg circle').length).toBe(2);

		await fireEvent.click(screen.getByRole('button', { name: '回到「附近的人」' }));
		expect(onTitleClick).toHaveBeenCalledOnce();
	});

	it('进度圆环：100% 弧长归零，0% 为整圈空环', () => {
		const { container: full } = render(PageIsland, { props: { title: 'A', progress: 100 } });
		const arcOf = (root: HTMLElement) =>
			Number(
				(root.querySelector('[data-island-progress-ring] svg circle:last-of-type') as SVGCircleElement).getAttribute(
					'stroke-dashoffset'
				)
			);
		expect(arcOf(full)).toBeCloseTo(0, 3);

		const { container: empty } = render(PageIsland, { props: { title: 'B', progress: 0 } });
		const circumference = 2 * Math.PI * 7;
		expect(arcOf(empty)).toBeCloseTo(circumference, 3);
	});

	it('详情页身份模式：显示作者名，点击进入作者主页（由调用方语义决定）', async () => {
		const onTitleClick = vi.fn();
		const { container } = render(PageIsland, {
			props: {
				title: '瞬间',
				author: { name: '鹿鸣', location: '云龙湖东岸', time: '10月29日 21:59' },
				onTitleClick,
				progress: 10
			}
		});
		const island = container.querySelector('[data-desktop-dynamic-island]') as HTMLElement;
		expect(island.textContent).toContain('鹿鸣');
		// meta 缺省回退作者位置 · 时间
		expect(island.textContent).toContain('云龙湖东岸 · 10月29日 21:59');

		await fireEvent.click(screen.getByRole('button', { name: '查看 鹿鸣 的资料' }));
		expect(onTitleClick).toHaveBeenCalledOnce();
	});

	it('详情页身份模式：支持自定义 role（如社群）与 onAuthorClick 点击回调', async () => {
		const onAuthorClick = vi.fn();
		render(PageIsland, {
			props: {
				title: '群组',
				author: { name: '骑行俱乐部', location: '徐州市', time: '12 位成员', role: '社群' },
				onAuthorClick
			}
		});

		// 展开面板
		await fireEvent.click(screen.getByRole('button', { name: '展开控制面板' }));
		expect(screen.getByText('社群')).not.toBeNull();
		const groupBtn = screen.getByRole('button', { name: '查看群组' });
		expect(groupBtn).not.toBeNull();

		await fireEvent.click(groupBtn);
		expect(onAuthorClick).toHaveBeenCalledOnce();
	});

	it('返回钮仅在传入 onBack 时渲染；点击回调', async () => {
		const onBack = vi.fn();
		const { container } = render(PageIsland, {
			props: { title: '搜索', onBack, backLabel: '返回上一页' }
		});
		const island = container.querySelector('[data-desktop-dynamic-island]') as HTMLElement;
		expect(island.textContent).toContain('搜索');
		await fireEvent.click(screen.getByRole('button', { name: '返回上一页' }));
		expect(onBack).toHaveBeenCalledOnce();
	});

	it('进度越界收敛到 0–100', () => {
		const { container: over } = render(PageIsland, { props: { title: 'A', progress: 180 } });
		expect(
			(over.querySelector('[data-island-progress-ring]') as HTMLElement).dataset.islandProgress
		).toBe('100');
		const { container: under } = render(PageIsland, { props: { title: 'B', progress: -5 } });
		expect(
			(under.querySelector('[data-island-progress-ring]') as HTMLElement).dataset.islandProgress
		).toBe('0');
	});

	it('普通页面缺省渲染 leading 视觉微锚点（accent 圆点）保持胶囊平衡', () => {
		const { container } = render(PageIsland, { props: { title: '附近瞬间' } });
		const dot = container.querySelector('.bg-accent-default\\/70');
		expect(dot).not.toBeNull();
		expect(dot?.getAttribute('aria-hidden')).toBe('true');
	});

	it('点击展开面板后，点击外部或按 Escape 键自动收起', async () => {
		const { container } = render(PageIsland, { props: { title: '附近的人' } });
		const expandBtn = screen.getByRole('button', { name: '展开控制面板' });
		expect(expandBtn).not.toBeNull();

		// 点击展开
		await fireEvent.click(expandBtn);
		expect(screen.getByText('页面控制与导航')).not.toBeNull();
		expect(screen.getByRole('button', { name: '收起控制面板' })).not.toBeNull();

		// 点击外部空白区域（触发 window pointerdown）
		const outside = document.createElement('div');
		document.body.appendChild(outside);
		await fireEvent.pointerDown(outside);
		outside.remove();

		// 面板已收起
		expect(screen.queryByText('页面控制与导航')).toBeNull();
		expect(screen.getByRole('button', { name: '展开控制面板' })).not.toBeNull();

		// 再次展开并测试 Escape 键收起
		await fireEvent.click(screen.getByRole('button', { name: '展开控制面板' }));
		expect(screen.getByText('页面控制与导航')).not.toBeNull();

		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(screen.queryByText('页面控制与导航')).toBeNull();
	});

	it('支持传入 center 自定义交互区（如搜索输入框）替代默认标题文本按钮', () => {
		const { container } = render(PageIsland, {
			props: {
				title: '搜索',
				center: createRawSnippet(() => ({
					render: () => '<input data-test-island-input placeholder="搜索测试..." />'
				}))
			}
		});
		const input = container.querySelector('input[data-test-island-input]');
		expect(input).not.toBeNull();
		// 存在 center 时，默认的主标题按钮不应被重复渲染
		expect(container.querySelector('button[aria-label="回到「搜索」"]')).toBeNull();
	});
});

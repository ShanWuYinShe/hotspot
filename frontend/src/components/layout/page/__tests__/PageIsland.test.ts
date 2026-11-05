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
});

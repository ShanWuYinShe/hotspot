// MapHeatDot 契约测试：数目标签、定位、直径分档与 token 样式。
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import MapHeatDot from '../MapHeatDot.svelte';

const bounds = { minLat: 30, maxLat: 31, minLng: 117, maxLng: 118 };

function bucketWith(total: number, breakdown: Record<string, number | null> | null = null) {
	return {
		cell_token: 't',
		level: 12,
		location: { latitude: 30.5, longitude: 117.5 },
		total_count: total,
		breakdown
	} as never;
}

function renderDot(total: number, breakdown: Record<string, number | null> | null = null) {
	const { container } = render(MapHeatDot, { props: { bucket: bucketWith(total, breakdown), bounds } });
	return container.firstElementChild as HTMLElement;
}

describe('MapHeatDot', () => {
	it('直接把该网格的对象总数显示出来（用户一眼可知聚合了多少内容）', () => {
		expect(renderDot(54).textContent?.trim()).toBe('54');
		expect(renderDot(7).textContent?.trim()).toBe('7');
	});

	it('超过 999 收敛为 999+，避免数字撑破圆点', () => {
		expect(renderDot(12345).textContent?.trim()).toBe('999+');
	});

	it('按桶坐标换算视口百分比并居中', () => {
		const dot = renderDot(5);
		expect(dot.style.left).toBe('50%');
		expect(dot.style.top).toBe('50%');
		expect(dot.style.transform).toContain('translate(-50%,-50%)');
	});

	it('直径按数字位数分档，保证数字完整显示', () => {
		const cases: Array<[number, string]> = [
			[7, '28px'],
			[54, '28px'],
			[540, '34px'],
			[12345, '40px']
		];
		for (const [total, size] of cases) {
			const dot = renderDot(total);
			expect(dot.style.width, `total=${total}`).toBe(size);
			expect(dot.style.height, `total=${total}`).toBe(size);
		}
	});

	it('中性卡面 + accent 描边：既醒目又不与「实心蓝」的位置点/图钉同形同色', () => {
		const dot = renderDot(54);
		// jsdom 会把 style 属性规范化（冒号后补空格），比较前压掉空白
		const style = (dot.getAttribute('style') ?? '').replace(/\s+/g, '');
		// 卡面用 surface、文字用常规文字色：不占用任何「实心彩色」语义
		expect(style).toContain('background:var(--cst-surface-primary)');
		expect(style).toContain('color:var(--cst-text-primary)');
		// accent 只用于描边（保留品牌色醒目度），不得再作填充——
		// 填充成实心蓝圆点时，会与 --cst-marker-user 图钉和「当前位置点」混为一谈
		expect(style).toContain('border:var(--cst-border-width-strong)solidvar(--cst-accent-default)');
		expect(style).not.toContain('background:var(--cst-accent-default)');
		expect(dot.className).toContain('shadow-raised-md');
		// 旧实现靠 opacity 插值弱化，现为实色，不应再写入 opacity
		expect(style).not.toContain('opacity:');
	});

	it('用徽标排版契约（居中 + 不继承全局行高）', () => {
		const dot = renderDot(54);
		const cls = dot.className;
		for (const required of ['inline-flex', 'items-center', 'justify-center', 'leading-none', 'rounded-full']) {
			expect(cls, required).toContain(required);
		}
	});

	it('层级高于当前位置点（z-10 > z-index:5），数字不会被定位点盖住', () => {
		// 当前位置点固定在视口正中心且自带 z-index:5；地图中心常与数据密集区重合，
		// 热力徽标必须抬到它之上，否则该处数字整块不可见（实测 2026-10-03）。
		expect(renderDot(54).className).toContain('z-10');
	});

	it('提供含分类明细的无障碍名称', () => {
		const withBreakdown = renderDot(34, { user: 22, group: 4, moment: 8 });
		expect(withBreakdown.getAttribute('aria-label')).toBe('此处聚合 34 项：22 位用户 · 4 个群组 · 8 条动态');
		expect(withBreakdown.getAttribute('role')).toBe('img');
	});

	it('breakdown 为 null 时无障碍名称退化为总数（不出现 undefined）', () => {
		const dot = renderDot(12, null);
		expect(dot.getAttribute('aria-label')).toBe('此处聚合 12 项');
	});
});

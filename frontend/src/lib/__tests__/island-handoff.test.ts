import { describe, it, expect } from 'vitest';
import {
	anchorHandoffProgress,
	handoffCss,
	islandHandoff,
	publishHandoff,
	scrollHandoffProgress
} from '../island-handoff';
import type { IslandRect } from '../island-morph';

const rect = (x: number, y: number, width: number, height: number): IslandRect => ({
	x,
	y,
	width,
	height
});

describe('island-handoff 纯函数', () => {
	it('滚动窗口进度：窗口前为 0、窗口后为 1、窗口内线性', () => {
		expect(scrollHandoffProgress(0, 120, 260)).toBe(0);
		expect(scrollHandoffProgress(119, 120, 260)).toBe(0);
		expect(scrollHandoffProgress(120, 120, 260)).toBe(0);
		expect(scrollHandoffProgress(190, 120, 260)).toBeCloseTo(0.5, 5);
		expect(scrollHandoffProgress(260, 120, 260)).toBe(1);
		expect(scrollHandoffProgress(999, 120, 260)).toBe(1);
	});

	it('锚点位置进度：标题接近顶栏时开始、完全被顶栏裁掉后落位', () => {
		const barBottom = 52;
		const height = 28;
		// 标题完整可见（top=100，远低于顶栏下沿）→ 0
		expect(anchorHandoffProgress(100, height, barBottom)).toBe(0);
		// 刚好要点到顶栏（top = barBottom + approach = 76）→ 0
		expect(anchorHandoffProgress(76, height, barBottom)).toBe(0);
		// 完全没入顶栏（top = barBottom - height - tail = 14）→ 1
		expect(anchorHandoffProgress(14, height, barBottom)).toBe(1);
		expect(anchorHandoffProgress(-40, height, barBottom)).toBe(1);
		// 中段单调、且落在 (0,1)
		const mid = anchorHandoffProgress(40, height, barBottom);
		expect(mid).toBeGreaterThan(0);
		expect(mid).toBeLessThan(1);
		expect(anchorHandoffProgress(50, height, barBottom)).toBeLessThan(mid);
	});

	it('交接样式：p=0 收缩在锚点、p=1 回到 CSS 接管（transform 清空、内容全显）', () => {
		const target = rect(600, 8, 200, 36);
		const source = rect(100, 90, 80, 28);

		const start = handoffCss(target, source, 0);
		expect(start.opacity).toBe(0);
		expect(start.content).toBe(0);
		expect(start.transform).toContain('translate3d');
		expect(start.transform).toContain('scale(');

		const end = handoffCss(target, source, 1);
		expect(end.transform).toBe('');
		expect(end.opacity).toBe(1);
		expect(end.content).toBe(1);
	});

	it('交接样式：中段单调（壳透明度只增、内容只增、位移量只减）', () => {
		const target = rect(600, 8, 200, 36);
		const source = rect(100, 90, 80, 28);
		const samples = [0.2, 0.4, 0.6, 0.8].map((p) => handoffCss(target, source, p));
		for (let i = 1; i < samples.length; i++) {
			expect(samples[i].opacity).toBeGreaterThanOrEqual(samples[i - 1].opacity);
			expect(samples[i].content).toBeGreaterThanOrEqual(samples[i - 1].content);
		}
		expect(samples[0].content).toBe(0);
		expect(samples[samples.length - 1].content).toBeGreaterThan(0.5);
	});

	it('islandHandoff action 默认启用：不传 params 也会消费交接帧（防「岛不跟手」回归）', () => {
		const node = document.createElement('div');
		node.getBoundingClientRect = () =>
			({
				x: 600,
				y: 8,
				left: 600,
				top: 8,
				width: 200,
				height: 36,
				right: 800,
				bottom: 44,
				toJSON: () => ({})
			}) as DOMRect;
		document.body.appendChild(node);
		const cleanup = islandHandoff(node, {});
		try {
			// 0.5 处于交接中段：应写入锚点位移/缩放与半透明，而不是「停在泊位、完全不透明」
			publishHandoff({ p: 0.5, source: { x: 100, y: 76, width: 80, height: 28 } });
			expect(node.style.transform).toContain('translate3d');
			// 壳在交接前段就凝实（显影快），但内容层必须还没显字
			expect(Number(node.style.opacity)).toBeGreaterThan(0);
			expect(Number(node.style.getPropertyValue('--cst-island-content'))).toBeLessThan(1);
			// 落位后清空内联 transform，把 hover 的 1px 上移交还给 CSS
			publishHandoff({ p: 1, source: { x: 100, y: 76, width: 80, height: 28 } });
			expect(node.style.transform).toBe('');
			expect(node.style.opacity).toBe('1');
			expect(node.classList.contains('island-land')).toBe(true);
		} finally {
			cleanup?.destroy();
			node.remove();
			publishHandoff({ p: 0, source: null });
		}
	});

	it('岛落地微动效：抵达终态 (p=1) 时触发 .island-land，反向滚动 (p<1) 立即清空', () => {
		const node = document.createElement('div');
		node.getBoundingClientRect = () =>
			({
				x: 600,
				y: 8,
				left: 600,
				top: 8,
				width: 200,
				height: 36,
				right: 800,
				bottom: 44,
				toJSON: () => ({})
			}) as DOMRect;
		document.body.appendChild(node);
		publishHandoff({ p: 0, source: null });
		const cleanup = islandHandoff(node, {});
		try {
			// 初始 mount 不触发
			expect(node.classList.contains('island-land')).toBe(false);

			// 飞行中 p=0.6 不触发
			publishHandoff({ p: 0.6, source: { x: 100, y: 76, width: 80, height: 28 } });
			expect(node.classList.contains('island-land')).toBe(false);

			// 抵达 p=1 触发落地类
			publishHandoff({ p: 1, source: { x: 100, y: 76, width: 80, height: 28 } });
			expect(node.classList.contains('island-land')).toBe(true);

			// 反向滚动 p=0.8，立即清空落地类保持滚动严格跟手
			publishHandoff({ p: 0.8, source: { x: 100, y: 76, width: 80, height: 28 } });
			expect(node.classList.contains('island-land')).toBe(false);
		} finally {
			cleanup?.destroy();
			node.remove();
			publishHandoff({ p: 0, source: null });
		}
	});

	it('reduced-motion：中途不插值（过闸即切换），不做连续位移', () => {
		const target = rect(600, 8, 200, 36);
		const source = rect(100, 90, 80, 28);
		const before = handoffCss(target, source, 0.4, true);
		expect(before.opacity).toBe(0);
		expect(before.content).toBe(0);
		const after = handoffCss(target, source, 0.6, true);
		expect(after.transform).toBe('');
		expect(after.opacity).toBe(1);
		expect(after.content).toBe(1);
	});
});
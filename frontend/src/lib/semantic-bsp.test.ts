import { describe, it, expect } from 'vitest';
import { getCardBspLayout, computeBspLayoutSequence, type CardBspMeta } from './semantic-bsp';

describe('semantic-bsp', () => {
	it('桌面端：首项默认展开 2x2 巨幕精选', () => {
		const meta: CardBspMeta = { aspectRatio: 1.0, isFeatured: true };
		const cell = getCardBspLayout(meta, 0, { device: 'desktop', total: 6 });
		expect(cell.tier).toBe('2x2');
		expect(cell.cls).toBe('col-span-2 row-span-2');
		expect(cell.featured).toBe(true);
	});

	it('移动端：严禁 2x2 霸屏，首项降级为 2x1 满宽焦点', () => {
		const meta: CardBspMeta = { aspectRatio: 1.0, isFeatured: true };
		const cell = getCardBspLayout(meta, 0, { device: 'mobile', total: 6 });
		expect(cell.tier).toBe('2x1');
		expect(cell.cls).toBe('col-span-2 row-span-1');
		expect(cell.horizontal).toBe(true);
		expect(cell.featured).toBe(false);
	});

	it('比例感知：极竖长图 (ratio < 0.75) 识别为 1x2 竖高卡', () => {
		const meta: CardBspMeta = { width: 900, height: 1600 }; // ratio 0.5625
		const cell = getCardBspLayout(meta, 1, { device: 'desktop', total: 10 });
		expect(cell.tier).toBe('1x2');
		expect(cell.cls).toBe('col-span-1 row-span-2');
	});

	it('比例感知：开阔横图 (ratio > 1.35) 识别为 2x1 宽横卡', () => {
		const meta: CardBspMeta = { width: 1600, height: 900 }; // ratio 1.77
		const cell = getCardBspLayout(meta, 2, { device: 'desktop', total: 10 });
		expect(cell.tier).toBe('2x1');
		expect(cell.cls).toBe('col-span-2 row-span-1');
		expect(cell.horizontal).toBe(true);
	});

	it('防粘连机制：连续竖图时，第二张降级为 1x1，防止局部堆叠过长', () => {
		const items: CardBspMeta[] = [
			{ width: 1000, height: 1000 }, // index 0 hero -> 2x2
			{ width: 900, height: 1600 },  // index 1 竖图 -> 1x2
			{ width: 900, height: 1600 }   // index 2 竖图 -> 1x1 (防粘连降级)
		];
		const seq = computeBspLayoutSequence(items, { device: 'desktop' });
		expect(seq[0].tier).toBe('2x2'); // index 0 hero
		expect(seq[1].tier).toBe('1x2'); // index 1 竖图
		expect(seq[2].tier).toBe('1x1'); // index 2 防粘连降级
	});

	it('防粘连机制：连续横图时，第二张降级为 1x1', () => {
		const items: CardBspMeta[] = [
			{ width: 1000, height: 1000 }, // index 0 -> 2x2
			{ width: 1600, height: 900 },  // index 1 -> 2x1
			{ width: 1600, height: 900 }   // index 2 -> 1x1 (防粘连)
		];
		const seq = computeBspLayoutSequence(items, { device: 'desktop' });
		expect(seq[1].tier).toBe('2x1');
		expect(seq[2].tier).toBe('1x1');
	});

	it('纯文字卡（hasImage=false）在所有设备都不放大（避免空画布/空图片区）', () => {
		const items: CardBspMeta[] = Array.from({ length: 20 }, () => ({
			hasImage: false,
			likes: 99,
			isFeatured: true
		}));
		for (const device of ['mobile', 'tablet', 'desktop']) {
			const seq = computeBspLayoutSequence(items, { device });
			expect(seq.every((s) => s.tier === '1x1')).toBe(true);
		}
	});

	it('长流周期性节奏：长列表中周期性穿插 2x1 与 2x2 锚点', () => {
		const items: CardBspMeta[] = Array.from({ length: 20 }, () => ({
			width: 1000,
			height: 1000
		}));
		const seq = computeBspLayoutSequence(items, { device: 'desktop' });
		const tiers = seq.map((s) => s.tier);
		expect(tiers[0]).toBe('2x2');
		expect(tiers[3]).toBe('2x1'); // index 3 周期性穿插
		expect(tiers[8]).toBe('2x2'); // index 8 周期性 2x2 视觉锚点
	});
});

/**
 * 语义化 BSP 卡片网格布局引擎（Semantic BSP Grid Layout Engine）。
 *
 * 结合二叉空间分割（BSP）的离散多档位美学（1x1, 1x2, 2x1, 2x2）
 * 与现代化 CSS Grid Dense 密排引擎。
 *
 * 核心特性：
 * 1. 内容感知（Content-Aware）：根据图片长宽比（宽高比）、内容热度、推荐权重动态匹配档位；
 * 2. 节奏守卫（Rhythm & Anti-Clustering）：防止相同异形卡（如连续 1x2 或连续 2x1）扎堆，保持视线平滑流动；
 * 3. 跨端自适应（Cross-Device Adaptation）：移动端（双列）禁 2x2 霸屏降为 2x1，桌面端放开完整杂志画册组合；
 * 4. 零空洞满铺（Zero-Hole dense packing）：由原生 CSS grid-auto-flow: dense 负责无缝嵌合，零 CLS，无补位骨架。
 */
import type { Tier } from './bsp-layout';

export type CardTier = Tier;

export interface CardBspMeta {
	aspectRatio?: number | null;
	width?: number | null;
	height?: number | null;
	likes?: number;
	members?: number;
	isFeatured?: boolean;
	hasImage?: boolean;
	isOnline?: boolean;
}

export interface BspLayoutCell {
	tier: CardTier;
	cls: string;
	horizontal: boolean;
	featured: boolean;
}

export interface BspLayoutOptions {
	device?: string; // 'mobile' | 'tablet' | 'desktop'
	total?: number;
	prevTiers?: CardTier[];
	kind?: 'moment' | 'group' | 'user';
}

/**
 * 计算单张卡片在语义化 BSP 网格中的布局规格
 */
export function getCardBspLayout(
	item: CardBspMeta,
	index: number,
	options: BspLayoutOptions = {}
): BspLayoutCell {
	const { device = 'desktop', total = 1, prevTiers = [], kind = 'moment' } = options;
	const isMobile = device === 'mobile';
	const isTablet = device === 'tablet';

	// 计算图片长宽比 (width / height)
	let ratio: number | null = item.aspectRatio ?? null;
	if (ratio === null && item.width && item.height && item.height > 0) {
		ratio = item.width / item.height;
	}

	const prevTier = prevTiers[index - 1] ?? null;
	// 纯文字卡（hasImage === false）不参与任何跨度档位：2x2/1x2 会把无图内容
	// 拉成整块空画布，2x1 横排也会留下 40% 的空图片区（2K 下尤其明显）。
	// 所有大卡只留给真的有图的卡片。
	// 用户卡（kind === 'user'）同理：只有小头像、没有封面媒体，跨档后卡内会剩下
	// 一半以上空白（1440 实测 2x2 = 734×573，内容仅约 300px 高），一律 1x1。
	const canSpan = item.hasImage !== false && kind !== 'user';

	// 数据量 ≤3 时不做任何跨度：稀疏网格需要「整行居中」的稳定轨道，
	// 一张 2x2/2x1 会撑破居中行；少数据下巨卡也会放大空洞感。
	if (total <= 3) {
		return { tier: '1x1', cls: 'col-span-1 row-span-1', horizontal: false, featured: false };
	}

	let tier: CardTier = '1x1';

	if (isMobile) {
		// ════ 移动端（双列 2-Cols 约束）════
		// 严禁 2x2（会吃掉整屏高度，造成严重压迫感）。
		// 首项推荐：首张横画幅卡片或特别推荐卡展开 2x1 满宽焦点
		if (canSpan && index === 0 && (item.isFeatured || (ratio !== null && ratio >= 1.2) || Boolean(item.likes && item.likes > 5) || Boolean(item.members && item.members > 10))) {
			tier = '2x1';
		} else if (ratio !== null && ratio < 0.75) {
			// 竖长图（< 0.75，如 9:16、3:4）：采用 1x2 竖高卡（小红书式优雅纵深）
			// 节奏防粘连：若前一张刚是竖卡，降为 1x1 避免并排过长
			tier = prevTier === '1x2' ? '1x1' : '1x2';
		} else if (canSpan && ratio !== null && ratio > 1.35) {
			// 宽横图（> 1.35，如 16:9）：采用 2x1 满宽横向卡
			// 节奏防粘连：若前一张刚是横卡，降为 1x1
			tier = prevTier === '2x1' ? '1x1' : '2x1';
		} else {
			// 无明显极端比例时，通过周期律动（每 7-9 张）插入一次 2x1 满宽横条破局
			if (canSpan && index > 0 && index % 8 === 4 && prevTier !== '2x1') {
				tier = '2x1';
			} else {
				tier = '1x1';
			}
		}
	} else if (isTablet) {
		// ════ 平板端（三列 3-Cols 约束）════
		if (canSpan && index === 0 && (item.isFeatured || (ratio !== null && ratio >= 1.0) || total >= 6)) {
			tier = '2x2';
		} else if (canSpan && ratio !== null && ratio < 0.75) {
			tier = prevTier === '1x2' ? '1x1' : '1x2';
		} else if (canSpan && ratio !== null && ratio > 1.35) {
			tier = prevTier === '2x1' ? '1x1' : '2x1';
		} else if (canSpan && index % 7 === 3 && prevTier !== '2x1') {
			tier = '2x1';
		} else {
			tier = '1x1';
		}
	} else {
		// ════ 桌面端（4 ~ 6 列宽幅画册）════
		// 1. 首屏视线黄金位：首项优先展开 2x2 巨幕精选
		if (canSpan && index === 0 && (total >= 4 || item.isFeatured || (ratio !== null && ratio >= 1.0))) {
			tier = '2x2';
		} else if (canSpan && ratio !== null && ratio < 0.75) {
			// 竖图优先 1x2
			tier = prevTier === '1x2' ? '1x1' : '1x2';
		} else if (canSpan && ratio !== null && ratio > 1.35) {
			// 横图优先 2x1
			tier = prevTier === '2x1' ? '1x1' : '2x1';
		} else if (canSpan && total >= 12 && index > 0 && index % 14 === 8 && prevTier !== '2x2') {
			// 长瀑布流周期性 2x2 视觉锚点（每 14 张左右穿插 1 张巨幕）
			tier = '2x2';
		} else if (canSpan && index > 0 && (index % 6 === 3 || item.isFeatured || Boolean(item.members && item.members > 30)) && prevTier !== '2x1') {
			// 周期性横向穿插（2x1）
			tier = '2x1';
		} else {
			tier = '1x1';
		}
	}

	// 映射 CSS Grid 类名
	let cls = 'col-span-1 row-span-1';
	if (tier === '2x2') {
		cls = 'col-span-2 row-span-2';
	} else if (tier === '2x1') {
		cls = 'col-span-2 row-span-1';
	} else if (tier === '1x2') {
		cls = 'col-span-1 row-span-2';
	}

	return {
		tier,
		cls,
		horizontal: tier === '2x1',
		featured: tier === '2x2'
	};
}

/**
 * 批量计算整个列表的卡片布局序列（自动维系上下文与防粘连链条）
 */
export function computeBspLayoutSequence(
	items: CardBspMeta[],
	options: Omit<BspLayoutOptions, 'prevTiers'> = {}
): BspLayoutCell[] {
	const tiers: CardTier[] = [];
	const results: BspLayoutCell[] = [];
	const total = items.length;

	for (let i = 0; i < total; i++) {
		const cell = getCardBspLayout(items[i], i, {
			...options,
			total,
			prevTiers: tiers
		});
		tiers.push(cell.tier);
		results.push(cell);
	}

	return results;
}

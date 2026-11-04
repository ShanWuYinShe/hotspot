// 数据量自适应网格：0/1/2/少量/大量都保持同一套卡片尺寸与节奏。
//
// 规则（不引发布局跳变）：
// - 列数按断点：手机 2 / 平板 3 / 桌面按最小轨 auto-fill（与改造前 CSS 一致）；
// - 数量 < 可见列数 → 整行居中，但卡片宽度 = 密排时的轨道宽度（不放大、不缩小）；
// - 数量 ≥ 列数 → 显式网格铺开（最后一行不满保持列对齐，不居中、不拉伸）。
// 组件通过 clientWidth 测量容器宽度，用同一公式反推 auto-fill 的列数与轨道宽度，
// 保证稀疏/密集切换时卡片尺寸完全一致。

export interface AdaptiveFixedCols {
	/** 手机/平板（<1024）列数 */
	base: number;
	/** lg（≥1024）列数 */
	lg?: number;
	/** xl（≥1280）列数 */
	xl?: number;
}

export interface AdaptiveGridMetrics {
	columns: number;
	trackPx: number;
	centered: boolean;
}

// 断点与 platform store（mobile<600 / tablet 600–1023 / desktop≥1024）
// 及 Tailwind sm=600 保持同一口径：此前 mobile 用到 640，600–639 区间两端会打架。
const MOBILE_MAX = 600;
const TABLET_MAX = 1024;
const WIDE_MIN = 1280;

/** 根据容器宽度/数量算出列数、轨道宽度与是否居中（纯函数，便于单测）。 */
export function computeAdaptiveGrid(options: {
	width: number;
	/** 视口宽度：断点判定必须与 Tailwind 媒体查询一致（不是内容容器宽度）。 */
	viewportWidth?: number;
	count: number;
	rootPx: number;
	gapRem: readonly [number, number, number];
	/** auto 模式的最小轨宽（rem）；提供 fixedCols 时忽略。 */
	minTrackRem?: number;
	fixedCols?: AdaptiveFixedCols;
	/** 是否允许稀疏居中（BSP 大跨度卡片不可居中，否则会撑破轨道）。 */
	allowCenter?: boolean;
}): AdaptiveGridMetrics {
	const { width, viewportWidth, count, rootPx, gapRem, minTrackRem = 18.75, fixedCols, allowCenter = true } = options;
	const bp = viewportWidth ?? width;
	let band: 0 | 1 | 2 = 0;
	if (bp >= TABLET_MAX) band = 2;
	else if (bp >= MOBILE_MAX) band = 1;
	const gapPx = gapRem[band] * rootPx;

	let columns: number;
	if (fixedCols) {
		columns =
			bp >= WIDE_MIN
				? (fixedCols.xl ?? fixedCols.lg ?? fixedCols.base)
				: bp >= TABLET_MAX
					? (fixedCols.lg ?? fixedCols.base)
					: fixedCols.base;
	} else if (band === 2) {
		// 与 repeat(auto-fill, minmax(minTrack, 1fr)) 同公式反推列数。
		columns = Math.max(1, Math.floor((width + gapPx) / (minTrackRem * rootPx + gapPx)));
	} else {
		columns = band === 1 ? 3 : 2;
	}

	const trackPx = Math.max(0, (width - (columns - 1) * gapPx) / columns);
	const centered = allowCenter && count > 0 && count < columns;
	return { columns, trackPx, centered };
}

/** 生成网格行内样式：稀疏时固定 count 个轨道并居中，密集时显式列数（与 auto-fill 等宽）。 */
export function adaptiveGridStyle(options: {
	width: number;
	viewportWidth?: number;
	count: number;
	rootPx: number;
	gapRem: readonly [number, number, number];
	minTrackRem?: number;
	fixedCols?: AdaptiveFixedCols;
	allowCenter?: boolean;
}): string {
	const bp = options.viewportWidth ?? options.width;
	const band = bp >= TABLET_MAX ? 2 : bp >= MOBILE_MAX ? 1 : 0;
	const gapVar = options.gapRem[band] * options.rootPx;
	if (options.width <= 0) {
		if (options.fixedCols) {
			const cols =
				bp >= WIDE_MIN
					? (options.fixedCols.xl ?? options.fixedCols.lg ?? options.fixedCols.base)
					: bp >= TABLET_MAX
						? (options.fixedCols.lg ?? options.fixedCols.base)
						: options.fixedCols.base;
			return 'grid-template-columns:repeat(' + cols + ',minmax(0,1fr));gap:' + gapVar + 'px;';
		}
		// 首帧未测量前：手机/平板直接用固定列数（避免 auto-fill 先塌成 1 列的闪动）。
		if (bp > 0 && bp < TABLET_MAX) {
			const cols = bp >= MOBILE_MAX ? 3 : 2;
			return 'grid-template-columns:repeat(' + cols + ',minmax(0,1fr));gap:' + gapVar + 'px;';
		}
		return 'grid-template-columns:repeat(auto-fill,minmax(' + (options.minTrackRem ?? 18.75) + 'rem,1fr));gap:' + gapVar + 'px;';
	}
	const { columns, trackPx, centered } = computeAdaptiveGrid(options);
	if (centered) {
		return 'grid-template-columns:repeat(' + options.count + ', ' + trackPx + 'px);justify-content:center;gap:' + gapVar + 'px;';
	}
	return 'grid-template-columns:repeat(' + columns + ', minmax(0, 1fr));gap:' + gapVar + 'px;';
}

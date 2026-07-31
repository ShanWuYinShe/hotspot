<script lang="ts" generics="T">
	import type { Snippet } from 'svelte';
	import {
		generateBspCardLayout,
		deriveBspGridGeometry,
		appendPlacements,
		type BspRect
	} from '@/lib/bsp-layout';

	// 响应式 BSP 渲染器（三个桌面发现目录共用）。
	// 迁移自 React BspCardCanvas.tsx：卡片多为 1×1，穿插少量 1×2 / 2×1，
	// 偶尔 2×2 hero。包含 ResizeObserver 重排、IntersectionObserver 虚拟化
	// 与滚动加载哨兵。

	let {
		items,
		renderItem,
		className = '',
		gap = 24,
		minCardSize = 160,
		minCardWidth,
		minCardHeight,
		minRatio = 0.68,
		maxRatio = 1.48,
		minCanvasHeight = 460,
		ariaLabel = 'Card layout',
		onLoadMore,
		hasMore = false
	}: {
		items: T[];
		renderItem: Snippet<[T, number]>;
		className?: string;
		/** Clear space between cards. 24px protects the shared card hover lift and scale. */
		gap?: number;
		/** Minimum visible edge length after the inter-card gutter has been removed. */
		minCardSize?: number;
		/** Minimum visible card width after the gutter is removed. Defaults to `minCardSize`. */
		minCardWidth?: number;
		/** Minimum visible card height after the gutter is removed. Defaults to `minCardSize`. */
		minCardHeight?: number;
		minRatio?: number;
		maxRatio?: number;
		minCanvasHeight?: number;
		ariaLabel?: string;
		/** 哨兵接近视口时触发一次（需配合 hasMore 防重复触发）。 */
		onLoadMore?: () => void;
		hasMore?: boolean;
	} = $props();

	// 方向性最小值折叠到对称回退（与 React 版一致）
	const resolvedMinWidth = minCardWidth ?? minCardSize;
	const resolvedMinHeight = minCardHeight ?? minCardSize;

	let containerEl = $state<HTMLElement | null>(null);
	let width = $state(0);
	const seed = Math.floor(Math.random() * 2 ** 32);
	let solve = $state<{ canvasHeight: number; layout: BspRect[] }>({
		canvasHeight: minCanvasHeight,
		layout: []
	});

	// 求解缓存（跨 items 增长事件 APPEND 而非全量重解）；宽度变化时重置
	let prevSolve: { width: number; count: number; height: number; layout: BspRect[] } | null = null;
	/** 首次成功求解后固定的列数（网格几何冻结，加载更多只向下扩展）。 */
	let pinnedCols: number | null = null;

	/** 求解：append 路径（同宽 + items 增长）或全新 solve；语义与 React useMemo 版一致。 */
	function recompute(w: number, n: number) {
		if (!w || !n) {
			prevSolve = null;
			pinnedCols = null;
			return { canvasHeight: minCanvasHeight, layout: [] as BspRect[] };
		}

		// generateBspCardLayout 在 gap 内缩前度量源叶，把 gutter 加回去
		const sourceMinWidth = resolvedMinWidth + gap;
		const sourceMinHeight = resolvedMinHeight + gap;

		// 宽度变化时清除固定列数，下次 solve 按新宽度重新定列
		if (pinnedCols !== null && prevSolve?.width !== w) {
			pinnedCols = null;
		}

		// Append 路径：同宽、items 只增。冻结已有卡片，仅把新卡放入剩余空位，
		// 再向下扩展画布足够行数。
		if (prevSolve && prevSolve.width === w && n > prevSolve.count) {
			const appendGeo = deriveBspGridGeometry({
				canvasWidth: w, canvasHeight: w * 3, targetCount: n,
				gap, minWidth: sourceMinWidth, minHeight: sourceMinHeight, minRatio, maxRatio,
				cols: pinnedCols ?? undefined
			});
			const estimatedRows = Math.max(2, Math.ceil((n * 1.6) / appendGeo.cols));
			const expandedHeight = Math.max(prevSolve.height, Math.round(estimatedRows * appendGeo.unitH));

			const appended = appendPlacements({
				existing: prevSolve.layout,
				canvasWidth: w,
				canvasHeight: expandedHeight,
				newCount: n,
				gap,
				minWidth: sourceMinWidth,
				minHeight: sourceMinHeight,
				minRatio,
				maxRatio,
				cols: pinnedCols ?? undefined,
				seed
			});
			if (appended.length >= n) {
				const maxY = appended.reduce((m, r) => Math.max(m, r.y + r.height), 0);
				const newHeight = Math.max(expandedHeight, Math.ceil(maxY / appendGeo.unitH) * appendGeo.unitH);
				prevSolve = { width: w, count: n, height: newHeight, layout: appended };
				return { canvasHeight: newHeight, layout: appended };
			}
			// Append 未能覆盖（罕见：网格几何不匹配）→ 落到全量重解
		}

		// 全新 solve：从最小可行行数开始探测候选高度，取第一个覆盖全部 items 的最小高度。
		const geometry = deriveBspGridGeometry({
			canvasWidth: w,
			canvasHeight: w * 3,
			targetCount: n,
			gap,
			minWidth: sourceMinWidth,
			minHeight: sourceMinHeight,
			minRatio,
			maxRatio,
			cols: pinnedCols ?? undefined
		});
		const cols = geometry.cols;
		if (pinnedCols === null) pinnedCols = cols;

		// 小集合做丰富搜索（多 seed），大集合单次确定性求解
		const richSearchThreshold = 24;
		const rich = n <= richSearchThreshold;
		const avgCellsPerCard = 1.6;
		const estimatedRows = Math.max(2, Math.ceil((n * avgCellsPerCard) / cols));
		const minRowsForCount = rich ? Math.max(2, Math.ceil(n / cols)) : estimatedRows;

		let bestResult: BspRect[] = [];
		let chosenHeight = Math.max(minCanvasHeight, Math.round(minRowsForCount * geometry.unitH));

		const maxExtraRows = rich ? 4 : 0;
		const seedsPerHeight = rich ? 64 : 1;
		for (let extraRows = 0; extraRows <= maxExtraRows; extraRows++) {
			const rows = minRowsForCount + extraRows;
			const candidateHeight = Math.round(rows * geometry.unitH);
			if (candidateHeight < minCanvasHeight && extraRows === 0) continue;

			for (let seedOffset = 0; seedOffset < seedsPerHeight; seedOffset++) {
				const result = generateBspCardLayout({
					canvasWidth: w,
					canvasHeight: candidateHeight,
					targetCount: n,
					gap,
					minWidth: sourceMinWidth,
					minHeight: sourceMinHeight,
					minRatio,
					maxRatio,
					cols: pinnedCols ?? undefined,
					seed: seed + seedOffset * 0x9e3779b9
				});
				const covers = result.length >= n;
				const better = covers && (
					bestResult.length < n ||
					result.length < bestResult.length ||
					(result.length === bestResult.length && bestResult.length === 0)
				);
				if (better) {
					bestResult = result;
					chosenHeight = candidateHeight;
				}
				if (covers) {
					prevSolve = { width: w, count: n, height: candidateHeight, layout: result };
					return { canvasHeight: candidateHeight, layout: result };
				}
			}
			if (bestResult.length >= n) break;
		}

		prevSolve = { width: w, count: n, height: chosenHeight, layout: bestResult };
		return { canvasHeight: chosenHeight, layout: bestResult };
	}

	// 宽度 + items 变化驱动重解
	$effect(() => {
		solve = recompute(width, items.length);
	});

	// 容器宽度测量
	$effect(() => {
		const element = containerEl;
		if (!element) return;
		const update = () => {
			width = Math.round(element.getBoundingClientRect().width);
		};
		update();
		const observer = new ResizeObserver(update);
		observer.observe(element);
		return () => observer.disconnect();
	});

	// 虚拟化：仅渲染视口附近（根 null + 400px overscan）的卡片占位
	const slotEls: (HTMLElement | null)[] = [];
	let visible = $state<Set<number>>(new Set());

	$effect(() => {
		// 依赖 layout：布局变化时重建观察
		const layout = solve.layout;
		const visibleSet = new Set<number>();
		const observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					const idx = Number((entry.target as HTMLElement).dataset.bspIndex);
					if (entry.isIntersecting) visibleSet.add(idx);
					else visibleSet.delete(idx);
				}
				// 顶部前 6 张常驻，初始绘制不空白
				for (let i = 0; i < Math.min(layout.length, 6); i++) visibleSet.add(i);
				visible = new Set(visibleSet);
			},
			{ root: null, rootMargin: '400px 0px 400px 0px' }
		);
		for (const el of slotEls) if (el) observer.observe(el);
		return () => observer.disconnect();
	});

	// 加载更多哨兵
	let sentinelEl = $state<HTMLElement | null>(null);

	$effect(() => {
		if (!onLoadMore || !hasMore) return;
		const sentinel = sentinelEl;
		if (!sentinel) return;
		const observer = new IntersectionObserver(
			(entries) => {
				if (entries.some((e) => e.isIntersecting)) onLoadMore();
			},
			{ root: null, rootMargin: '600px 0px 0px 0px' }
		);
		observer.observe(sentinel);
		return () => observer.disconnect();
	});
</script>

<section
	bind:this={containerEl}
	aria-label={ariaLabel}
	class="relative w-full {className}"
	style="height:{solve.canvasHeight}px;"
>
	{#each solve.layout as rect, index (index)}
		{@const item = items.length > 0 ? items[index % items.length] : undefined}
		{@const isVisible = visible.has(index) || index < 6}
		<div
			bind:this={slotEls[index]}
			data-bsp-index={index}
			class="absolute"
			style="left:{rect.x}px;top:{rect.y}px;width:{rect.width}px;height:{rect.height}px;"
		>
			{#if isVisible && item !== undefined}
				{@render renderItem(item, index)}
			{/if}
		</div>
	{/each}
	{#if hasMore}
		<div bind:this={sentinelEl} aria-hidden="true" style="position:absolute;bottom:0;left:0;right:0;height:1px;"></div>
	{/if}
</section>

<script lang="ts" generics="T">
	import type { Snippet } from 'svelte';
	import {
		generateBspCardLayout,
		deriveBspGridGeometry,
		planBspLayout,
		appendPlacements,
		type BspRect
	} from '@/lib/bsp-layout';

	// 响应式 BSP 渲染器（三个桌面附近目录共用）。
	// 卡片多为 1×1，穿插少量 1×2 / 2×1，偶尔 2×2 hero。包含 ResizeObserver
	// 重排、IntersectionObserver 虚拟化与滚动加载哨兵。

	let {
		items,
		renderItem,
		class: className = '',
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
		class?: string;
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

	// 方向性最小值折叠到对称回退
	const resolvedMinWidth = $derived(minCardWidth ?? minCardSize);
	const resolvedMinHeight = $derived(minCardHeight ?? minCardSize);

	let containerEl = $state<HTMLElement | null>(null);
	let width = $state(0);
	const seed = Math.floor(Math.random() * 2 ** 32);
	// minCanvasHeight 仅作为首帧高度（此后由 recompute 驱动），有意只读取一次初始值。
	function initialSolve(): { canvasHeight: number; layout: BspRect[] } {
		return { canvasHeight: minCanvasHeight, layout: [] };
	}
	let solve = $state<{ canvasHeight: number; layout: BspRect[] }>(initialSolve());

	// 求解缓存（跨 items 增长事件 APPEND 而非全量重解）；宽度变化时重置
	let prevSolve: { width: number; count: number; height: number; layout: BspRect[] } | null = null;
	/** 首次成功求解后固定的列数（网格几何冻结，加载更多只向下扩展）。 */
	let pinnedCols: number | null = null;

	/** 真实卡片数：fillToFull 的补位格（placeholder）不算卡片。 */
	function realCount(rects: BspRect[]): number {
		return rects.reduce((n, r) => n + (r.placeholder ? 0 : 1), 0);
	}

	/** 求解：优先 plan 精确满铺（基础宽列数 + 恰好行数 + 配额，无补位骨架）；
	 *  append 路径冻结已放置卡片仅扩展新区；plan 失败回退探测循环。 */
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

		// Append 路径：同宽、items 只增。冻结已有卡片，新卡按 plan 的精确配额
		// 放入扩展行（初次布局满铺时没有剩余空位可填）。
		if (prevSolve && prevSolve.width === w && n > prevSolve.count) {
			const appendGeo = deriveBspGridGeometry({
				canvasWidth: w, canvasHeight: w * 3, targetCount: n,
				gap, minWidth: sourceMinWidth, minHeight: sourceMinHeight, minRatio, maxRatio,
				cols: pinnedCols ?? undefined
			});
			const frozenReal = realCount(prevSolve.layout);
			const newCards = n - frozenReal;
			if (newCards > 0) {
				const plan = planBspLayout({
					canvasWidth: w, targetCount: newCards,
					gap, minWidth: sourceMinWidth, minHeight: sourceMinHeight, minRatio, maxRatio,
					cols: pinnedCols ?? undefined
				});
				if (plan) {
					const oldRows = Math.max(1, Math.round(prevSolve.height / appendGeo.unitH));
					const expandedHeight = Math.round((oldRows + plan.rows) * appendGeo.unitH);
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
						quotas: plan.quotas,
						// 用 plan 验证成功的 seed 放置，保证实际布局与验证一致（无补位骨架）
						seed: plan.seed
					});
					if (realCount(appended) >= n && appended.every((r) => !r.placeholder)) {
						const maxY = appended.reduce((m, r) => Math.max(m, r.y + r.height), 0);
						const newHeight = Math.max(expandedHeight, Math.ceil(maxY / appendGeo.unitH) * appendGeo.unitH);
						prevSolve = { width: w, count: n, height: newHeight, layout: appended };
						return { canvasHeight: newHeight, layout: appended };
					}
				}
			}
			// plan 失败（罕见）→ 回退估算路径：按行数估算扩展高度，软配额
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
			if (realCount(appended) >= n && appended.every((r) => !r.placeholder)) {
				const maxY = appended.reduce((m, r) => Math.max(m, r.y + r.height), 0);
				const newHeight = Math.max(expandedHeight, Math.ceil(maxY / appendGeo.unitH) * appendGeo.unitH);
				prevSolve = { width: w, count: n, height: newHeight, layout: appended };
				return { canvasHeight: newHeight, layout: appended };
			}
			// Append 未能覆盖（罕见：网格几何不匹配）→ 落到全量重解
		}

		// 全新 solve：plan 精确满铺。列数由基础宽度决定，行数与配额由
		// planBspLayout 计算，保证 n 张卡恰好铺满网格（无骨架、无洞）。
		const plan = planBspLayout({
			canvasWidth: w, targetCount: n,
			gap, minWidth: sourceMinWidth, minHeight: sourceMinHeight, minRatio, maxRatio,
			cols: pinnedCols ?? undefined, minCanvasHeight
		});
		if (plan) {
			if (pinnedCols === null) pinnedCols = plan.geometry.cols;
			const result = generateBspCardLayout({
				canvasWidth: w,
				canvasHeight: plan.canvasHeight,
				targetCount: n,
				gap,
				minWidth: sourceMinWidth,
				minHeight: sourceMinHeight,
				minRatio,
				maxRatio,
				cols: plan.geometry.cols,
				quotas: plan.quotas,
				// 用 plan 验证成功的 seed 放置，保证实际布局与验证一致（无补位骨架）
				seed: plan.seed
			});
			if (realCount(result) >= n) {
				prevSolve = { width: w, count: n, height: plan.canvasHeight, layout: result };
				return { canvasHeight: plan.canvasHeight, layout: result };
			}
		}

		// plan 失败（罕见：超额不可分解）→ 探测循环兜底：从最小可行行数开始
		// 探测候选高度，取第一个覆盖全部 items 的最小高度。
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
				// covers 只看真实卡片数：fillToFull 的补位格不能替代真实卡片，
				// 否则行数不足时也会“覆盖”，底部会全是补位槽。
				const covers = realCount(result) >= n;
				const better = covers && realCount(bestResult) < n;
				if (better) {
					bestResult = result;
					chosenHeight = candidateHeight;
				}
				if (covers) {
					prevSolve = { width: w, count: n, height: candidateHeight, layout: result };
					return { canvasHeight: candidateHeight, layout: result };
				}
			}
			if (realCount(bestResult) >= n) break;
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

	// 虚拟化：仅渲染视口附近（根 null + 400px overscan）的卡片占位。
	// slotEls 用 $state：bind:this 赋值后触发观察 effect 重跑，避免
	// effect 先于 DOM 绑定执行导致所有槽位都观察不到。
	let slotEls = $state<(HTMLElement | null)[]>([]);
	let visible = $state<Set<number>>(new Set());

	// 渲染映射：占位格（fillToFull 补位）不消耗 items，真实卡片按真实序号取 item，
	// 避免布局格子数 > items 时出现重复卡片。
	const mappedLayout = $derived.by(() => {
		let real = 0;
		return solve.layout.map((rect) => {
			const isPlaceholder = rect.placeholder === true;
			const entry = { rect, item: real < items.length ? items[real] : undefined, isPlaceholder };
			if (!isPlaceholder) real++;
			return entry;
		});
	});

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
	{#each mappedLayout as { rect, item, isPlaceholder }, index (index)}
		{@const isVisible = visible.has(index) || index < 6}
		<div
			bind:this={slotEls[index]}
			data-bsp-index={index}
			class="absolute"
			style="left:{rect.x}px;top:{rect.y}px;width:{rect.width}px;height:{rect.height}px;"
		>
			{#if isVisible}
				{#if !isPlaceholder && item !== undefined}
					{@render renderItem(item, index)}
				{:else if isPlaceholder}
					<!-- 补位格：无对应卡片，渲染空骨架保持满铺轮廓（不重复用户） -->
					<div
						aria-hidden="true"
						class="h-full w-full"
						style="border-radius:20px;background:var(--cst-surface-primary);box-shadow:0 0 0 1px var(--cst-border-default);opacity:.55;"
					></div>
				{/if}
			{/if}
		</div>
	{/each}
	{#if hasMore}
		<div class="h-0.25" bind:this={sentinelEl} aria-hidden="true"  style="position:absolute;bottom:0;left:0;right:0"></div>
	{/if}
</section>

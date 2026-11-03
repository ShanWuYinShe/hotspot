<script lang="ts">
	import { onMount, type Snippet } from 'svelte';
	import { cn } from '@/lib/utils';
	import { platform } from '@/stores/platform.svelte';
	import { adaptiveGridStyle, type AdaptiveFixedCols } from '@/lib/adaptive-grid';

	// 数据量自适应卡片网格：统一「少量居中 / 大量铺开 / 卡片尺寸恒定」的排列规则。
	// 用法：<AdaptiveCardGrid count={items.length} minTrackRem={18.75} gapRem={[0.75, 0.875, 0.875]}>...</AdaptiveCardGrid>
	// - auto 模式（默认）：手机 2 列 / 平板 3 列 / 桌面 minmax(auto-fill)；
	// - fixed 模式：传 fixedCols 时按 base(<1024)/lg(≥1024)/xl(≥1280) 固定列数。
	let {
		count,
		minTrackRem = 18.75,
		gapRem = [0.75, 0.875, 1] as const,
		fixedCols = undefined,
		centerSparse = true,
		class: className = '',
		style = '',
		children
	}: {
		count: number;
		/** auto 模式：桌面最小轨道宽（rem，16px 基准）。 */
		minTrackRem?: number;
		/** 手机 / 平板 / 桌面三档 gap（rem）。 */
		gapRem?: readonly [number, number, number];
		/** fixed 模式列数档；提供后忽略 minTrackRem。 */
		fixedCols?: AdaptiveFixedCols;
		/** BSP 大跨度网格传 false：有跨列卡片时不做稀疏居中，避免撑破轨道。 */
		centerSparse?: boolean;
		class?: string;
		style?: string;
		children: Snippet;
	} = $props();

	let width = $state(0);
	// 断点判定必须用视口宽度（与 Tailwind 媒体查询一致）；容器宽度只用于 auto-fill 列数与轨道宽度。
	let viewportWidth = $state(typeof window === 'undefined' ? 0 : window.innerWidth);
	onMount(() => {
		const update = () => (viewportWidth = window.innerWidth);
		update();
		window.addEventListener('resize', update);
		return () => window.removeEventListener('resize', update);
	});
	const rootPx = $derived(16 * platform.uiScale);
	const gridStyle = $derived(
		adaptiveGridStyle({
			width,
			viewportWidth: viewportWidth || undefined,
			count,
			rootPx,
			gapRem,
			minTrackRem,
			fixedCols,
			allowCenter: centerSparse
		})
	);
</script>

<div bind:clientWidth={width} class={cn('grid', className)} style="{gridStyle}{style}">
	{@render children()}
</div>

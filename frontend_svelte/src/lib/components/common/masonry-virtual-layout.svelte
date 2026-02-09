<script lang="ts" generics="T">
	/**
	 * 虚拟滚动 Masonry 瀑布流布局组件
	 * 
	 * 适用于大量数据（1000+ 项）的场景
	 * 只渲染可见区域的项目，大幅提升性能
	 * 
	 * 特点：
	 * 1. 虚拟滚动（只渲染可见项）
	 * 2. 高性能（处理 10000+ 项无压力）
	 * 3. 真正的瀑布流布局
	 * 4. 支持动态加载
	 * 5. 平滑滚动体验
	 */
	import { onMount, tick } from 'svelte';
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	interface Props {
		/**
		 * 数据列表
		 */
		items: T[];
		
		/**
		 * 渲染每个项目的 snippet
		 */
		children: Snippet<[T, number]>;
		
		/**
		 * 项目的唯一 key 函数
		 */
		keyFn: (item: T, index: number) => string | number;
		
		/**
		 * 列数配置
		 */
		columns?: {
			mobile?: number;
			tablet?: number;
			desktop?: number;
			wide?: number;
		};
		
		/**
		 * 列间距（px）
		 */
		gap?: number;
		
		/**
		 * 估计的项目平均高度（px）
		 * 用于初始化布局，越准确性能越好
		 */
		estimatedItemHeight?: number;
		
		/**
		 * 额外渲染的项目数量（缓冲区）
		 * 越大滚动越流畅，但内存占用越高
		 */
		overscan?: number;
		
		/**
		 * 加载更多的回调
		 */
		onLoadMore?: () => void;
		
		/**
		 * 是否正在加载
		 */
		isLoading?: boolean;
		
		/**
		 * 是否还有更多数据
		 */
		hasMore?: boolean;
		
		/**
		 * 额外的 CSS class
		 */
		class?: string;
	}

	let {
		items = $bindable(),
		children,
		keyFn,
		columns = {},
		gap = 16,
		estimatedItemHeight = 300,
		overscan = 3,
		onLoadMore,
		isLoading = false,
		hasMore = true,
		class: className
	}: Props = $props();

	// 列数配置（$derived 确保 columns 变化时更新）
	const columnConfig = $derived({
		mobile: columns.mobile ?? 1,
		tablet: columns.tablet ?? 2,
		desktop: columns.desktop ?? 3,
		wide: columns.wide ?? 4
	});

	let container: HTMLDivElement;
	let scrollContainer: HTMLDivElement;
	let containerWidth = $state(0);
	let scrollTop = $state(0);
	let viewportHeight = $state(0);
	let currentColumns = $state(1);

	// 记录每个项目的位置和尺寸
	interface ItemRect {
		index: number;
		x: number;
		y: number;
		width: number;
		height: number;
		key: string | number;
	}

	let itemRects = $state.raw<ItemRect[]>([]);
	let totalHeight = $state(0);
	let visibleItems = $state.raw<ItemRect[]>([]);

	// 已测量高度的项目缓存（纯缓存，不需要响应式）
	const measuredHeights = new Map<string | number, number>();

	/**
	 * 根据窗口宽度计算列数
	 */
	function calculateColumns(width: number): number {
		if (width >= 1536) return columnConfig.wide;
		if (width >= 1024) return columnConfig.desktop;
		if (width >= 640) return columnConfig.tablet;
		return columnConfig.mobile;
	}

	/**
	 * 计算每列的宽度
	 */
	function calculateColumnWidth(totalWidth: number, cols: number, gapSize: number): number {
		const totalGap = (cols - 1) * gapSize;
		return (totalWidth - totalGap) / cols;
	}

	/**
	 * 计算所有项目的位置
	 */
	function calculateLayout() {
		if (!container || items.length === 0) return;

		const cols = currentColumns;
		const colWidth = calculateColumnWidth(containerWidth, cols, gap);

		// 初始化列高度（使用 let 避免响应式问题）
		const tempColumnHeights = new Array(cols).fill(0);
		const tempItemRects: ItemRect[] = [];

		items.forEach((item, index) => {
			const key = keyFn(item, index);
			
			// 找到最短的列
			const columnIndex = tempColumnHeights.indexOf(Math.min(...tempColumnHeights));
			
			// 计算位置
			const x = columnIndex * (colWidth + gap);
			const y = tempColumnHeights[columnIndex];
			
			// 使用缓存的高度或估计高度
			const height = measuredHeights.get(key) ?? estimatedItemHeight;
			
			// 创建 rect
			tempItemRects.push({
				index,
				x,
				y,
				width: colWidth,
				height,
				key
			});
			
			// 更新列高度
			tempColumnHeights[columnIndex] += height + gap;
		});

		// 一次性更新所有响应式状态
		itemRects = tempItemRects;
		totalHeight = Math.max(...tempColumnHeights);
	}

	/**
	 * 计算可见的项目
	 */
	function calculateVisibleItems() {
		const startY = scrollTop - overscan * estimatedItemHeight;
		const endY = scrollTop + viewportHeight + overscan * estimatedItemHeight;

		visibleItems = itemRects.filter(rect => {
			return rect.y + rect.height >= startY && rect.y <= endY;
		});
	}

	/**
	 * 测量项目的实际高度（Svelte 5 Attachment 工厂）
	 */
	function measureItem(key: string | number) {
		return (element: HTMLElement) => {
			const height = element.offsetHeight;
			const cachedHeight = measuredHeights.get(key);

			// 如果高度变化，更新缓存并重新计算布局
			if (cachedHeight !== height) {
				measuredHeights.set(key, height);
				tick().then(() => {
					calculateLayout();
					calculateVisibleItems();
				});
			}
		};
	}

	/**
	 * 处理滚动事件
	 */
	function handleScroll() {
		if (!scrollContainer) return;

		scrollTop = scrollContainer.scrollTop;
		calculateVisibleItems();

		// 检查是否需要加载更多
		if (
			onLoadMore &&
			hasMore &&
			!isLoading &&
			scrollTop + viewportHeight >= totalHeight - viewportHeight
		) {
			onLoadMore();
		}
	}

	/**
	 * 更新布局
	 */
	function updateLayout() {
		if (!container) return;

		const newWidth = container.offsetWidth;
		const newColumns = calculateColumns(newWidth);

		if (newWidth !== containerWidth || newColumns !== currentColumns) {
			containerWidth = newWidth;
			currentColumns = newColumns;
			calculateLayout();
			calculateVisibleItems();
		}
	}

	// 响应数据变化
	$effect(() => {
		if (items.length > 0) {
			calculateLayout();
			calculateVisibleItems();
		}
	});

	onMount(() => {
		// 初始化尺寸
		containerWidth = container.offsetWidth;
		viewportHeight = scrollContainer.offsetHeight;
		currentColumns = calculateColumns(containerWidth);

		// 初始化布局
		calculateLayout();
		calculateVisibleItems();

		// 监听滚动
		scrollContainer.addEventListener('scroll', handleScroll);

		// 监听窗口大小变化
		window.addEventListener('resize', updateLayout);

		// 使用 ResizeObserver
		const resizeObserver = new ResizeObserver(() => {
			updateLayout();
		});
		resizeObserver.observe(container);

		return () => {
			scrollContainer?.removeEventListener('scroll', handleScroll);
			window.removeEventListener('resize', updateLayout);
			resizeObserver?.disconnect();
		};
	});

	/**
	 * 公开方法：刷新布局
	 */
	export function refresh() {
		calculateLayout();
		calculateVisibleItems();
	}

	/**
	 * 公开方法：滚动到顶部
	 */
	export function scrollToTop() {
		if (scrollContainer) {
			scrollContainer.scrollTop = 0;
		}
	}

	/**
	 * 公开方法：滚动到指定项目
	 */
	export function scrollToItem(index: number) {
		const rect = itemRects[index];
		if (rect && scrollContainer) {
			scrollContainer.scrollTop = rect.y;
		}
	}
</script>

<div
	bind:this={scrollContainer}
	class={cn('masonry-virtual-scroll h-full overflow-y-auto custom-scrollbar', className)}
>
	<div
		bind:this={container}
		class="masonry-virtual-container relative w-full"
		style:height={`${totalHeight}px`}
	>
		{#each visibleItems as rect (rect.key)}
			<div
				class="masonry-virtual-item absolute"
				style:left={`${rect.x}px`}
				style:top={`${rect.y}px`}
				style:width={`${rect.width}px`}
				{@attach measureItem(rect.key)}
			>
				{@render children(items[rect.index]!, rect.index)}
			</div>
		{/each}
	</div>
	
	{#if isLoading}
		<div class="flex items-center justify-center py-8">
			<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
			<span class="ml-3 text-muted-foreground">加载中...</span>
		</div>
	{/if}
	
	{#if !hasMore && items.length > 0}
		<div class="flex items-center justify-center py-8 text-muted-foreground text-sm">
			没有更多了
		</div>
	{/if}
</div>

<style>
	.masonry-virtual-scroll {
		/* 确保滚动容器有固定高度 */
		position: relative;
	}

	.masonry-virtual-container {
		/* 容器高度由 JS 动态设置 */
		position: relative;
	}

	.masonry-virtual-item {
		/* 项目使用绝对定位 */
		position: absolute;
		box-sizing: border-box;
		
		/* 入场动画 */
		animation: fade-in-up 0.3s ease-out;
	}

	@keyframes fade-in-up {
		from {
			opacity: 0;
			transform: translateY(20px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	/* 禁用动画 */
	@media (prefers-reduced-motion: reduce) {
		.masonry-virtual-item {
			animation: none;
		}
	}
</style>

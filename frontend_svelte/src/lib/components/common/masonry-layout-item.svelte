<script lang="ts">
	/**
	 * Masonry 布局项组件
	 * 
	 * 配合 MasonryLayout 使用，自动处理项目的定位
	 */
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils';

	interface Props {
		/**
		 * 子元素 snippet
		 */
		children?: Snippet;
		
		/**
		 * 额外的 CSS class
		 */
		class?: string;
	}

	let {
		children,
		class: className
	}: Props = $props();
</script>

<div class={cn('masonry-item relative z-0 w-full', className)}>
	{#if children}
		{@render children()}
	{/if}
</div>

<style>
	.masonry-item {
		/* 位置和宽度由父组件的 JS 设置 */
		/* 这里只定义内部样式 */
		box-sizing: border-box;
	}
</style>

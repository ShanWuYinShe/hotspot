<script lang="ts">
	/**
	 * 通用无限滚动加载指示器组件
	 * 
	 * 参考 Instagram/Twitter/小红书 的设计：轻量、微妙、不打扰
	 */
	import Loader2 from 'lucide-svelte/icons/loader-2';

	interface Props {
		/** 是否正在加载 */
		loading: boolean;
		/** 是否还有更多数据 */
		hasMore: boolean;
		/** 点击加载更多的回调 */
		onLoadMore: () => void;
		/** 自定义CSS类 */
		class?: string;
		/** 是否开启无限滚动（滚动到底部自动加载） */
		infinite?: boolean;
	}

	let {
		loading,
		hasMore,
		onLoadMore,
		class: className = '',
		infinite = true
	}: Props = $props();

	let viewport: HTMLDivElement | undefined = $state();
	let wasIntersecting = $state(false);

	$effect(() => {
		if (!infinite || !hasMore || !viewport) return;

		const observer = new IntersectionObserver(
			(entries) => {
				const isIntersecting = entries[0].isIntersecting;

				// 关键逻辑：只有当按钮从"不可见"变为"可见"时才触发加载
				// 这样可以避免加载完成后如果按钮还在视口中导致的无限循环
				if (isIntersecting && !wasIntersecting && !loading) {
					onLoadMore();
				}

				// 更新可见状态
				wasIntersecting = isIntersecting;
			},
			{
				rootMargin: '400px' // 提前 400px 开始加载
			}
		);

		observer.observe(viewport);

		return () => {
			observer.disconnect();
		};
	});
</script>

{#if hasMore || loading}
	<div bind:this={viewport} class="flex justify-center py-8 {className}">
		{#if loading}
			<!-- 加载中：轻量的 spinner -->
			<div class="flex items-center gap-2 text-muted-foreground">
				<Loader2 class="h-5 w-5 animate-spin" />
				<span class="text-sm">加载中...</span>
			</div>
		{:else}
			<!-- 未加载：微妙的向下箭头提示 -->
			<button
				type="button"
				onclick={onLoadMore}
				class="group flex flex-col items-center gap-1 text-muted-foreground/50 transition-colors hover:text-muted-foreground"
				aria-label="加载更多"
			>
				<svg
					class="h-6 w-6 animate-bounce"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M19 14l-7 7m0 0l-7-7m7 7V3"
					/>
				</svg>
				<span class="text-xs opacity-0 transition-opacity group-hover:opacity-100"
					>点击加载</span
				>
			</button>
		{/if}
	</div>
{/if}

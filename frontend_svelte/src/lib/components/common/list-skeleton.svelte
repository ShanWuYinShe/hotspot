<script lang="ts">
	/**
	 * 通用列表加载骨架屏组件
	 * 
	 * 用于显示列表加载状态的统一UI组件
	 * 支持多种类型的骨架屏样式
	 */
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';

	interface Props {
		/** 骨架屏类型 */
		type: 'user' | 'group' | 'moment' | 'chat' | 'notification';
		/** 显示数量 */
		count?: number;
		/** 自定义CSS类 */
		class?: string;
		/** 布局方式 */
		layout?: 'grid' | 'list';
	}

	let { type, count = 6, class: className = '', layout = 'grid' }: Props = $props();

	// 根据布局决定容器类
	const containerClass = $derived(
		layout === 'grid'
			? 'grid gap-4 sm:grid-cols-2 lg:grid-cols-3'
			: 'space-y-4'
	);
</script>

<div class="{containerClass} {className}">
	{#each Array(count) as _, i (i)}
		<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
			<CardContent class="p-6">
				{#if type === 'user'}
					<!-- 用户骨架屏 -->
					<div class="flex items-center gap-4">
						<Skeleton class="h-12 w-12 rounded-full" />
						<div class="flex-1 space-y-2">
							<Skeleton class="h-4 w-24" />
							<Skeleton class="h-3 w-16" />
						</div>
					</div>
					<Skeleton class="mt-3 h-4 w-full" />
					<Skeleton class="mt-1.5 h-4 w-3/4" />
				{:else if type === 'group'}
					<!-- 群组骨架屏 -->
					<div class="flex items-center gap-3">
						<Skeleton class="h-12 w-12 rounded-lg" />
						<div class="flex-1 space-y-2">
							<Skeleton class="h-4 w-24" />
							<Skeleton class="h-3 w-full" />
						</div>
					</div>
					<Skeleton class="mt-3 h-4 w-full" />
					<Skeleton class="mt-1.5 h-4 w-2/3" />
				{:else if type === 'moment'}
					<!-- 动态骨架屏 -->
					<div class="space-y-3">
						<div class="flex items-center gap-3">
							<Skeleton class="h-10 w-10 rounded-full" />
							<div class="space-y-1.5">
								<Skeleton class="h-4 w-24" />
								<Skeleton class="h-3 w-16" />
							</div>
						</div>
						<Skeleton class="h-32 w-full rounded-lg" />
						<Skeleton class="h-4 w-full" />
						<Skeleton class="h-4 w-2/3" />
					</div>
				{:else if type === 'chat'}
					<!-- 聊天骨架屏 -->
					<div class="flex items-center gap-3">
						<div class="relative">
							<Skeleton class="h-12 w-12 rounded-full" />
						</div>
						<div class="flex-1 space-y-2">
							<div class="flex items-center justify-between">
								<Skeleton class="h-4 w-24" />
								<Skeleton class="h-3 w-12" />
							</div>
							<Skeleton class="h-3 w-full" />
						</div>
					</div>
				{:else if type === 'notification'}
					<!-- 通知骨架屏 -->
					<div class="flex gap-3">
						<Skeleton class="h-10 w-10 rounded-lg" />
						<div class="flex-1 space-y-2">
							<Skeleton class="h-4 w-32" />
							<Skeleton class="h-3 w-full" />
							<Skeleton class="h-3 w-20" />
						</div>
					</div>
				{/if}
			</CardContent>
		</Card>
	{/each}
</div>

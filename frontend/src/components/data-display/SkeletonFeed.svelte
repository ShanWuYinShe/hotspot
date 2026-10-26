<script lang="ts">
	import { cn } from '@/lib/utils';
	import Skeleton from './Skeleton.svelte';

	// SkeletonFeed — 卡片/信息流首屏骨架屏（首页 feed、附近三叶、我的瞬间/群组等）。
	// 三种形态与真实卡片一一对应，保证「骨架 → 内容」零位移：
	//   feed     首页动态卡（作者行 + 正文两行 + 4:3 媒体 + 操作行）
	//   card     图片优先内容卡（4:3 媒体 + 信息托盘）；horizontal 用于移动端 h-32 横排卡
	//   portrait 人物/瞬间竖版卡（3:4 图片 + 底部信息条）
	// 几何口径对齐鸿蒙 MomentPage 骨架（SkeletonShimmer + 占位卡）：媒体块高度节奏、
	// 圆角档位（卡片 20 / 文本条 12）与行内间距（16 / 12 / 8）同口径；
	// 栅格列数由调用方 class 传入与真实内容完全相同的 grid 类。
	let {
		variant = 'feed',
		count = 6,
		horizontal = false,
		label = '加载中…',
		class: className = '',
		cardClass = ''
	}: {
		variant?: 'feed' | 'card' | 'portrait';
		count?: number;
		/** card 形态：移动端横排卡（图片左 40% + 信息右），对齐 GroupCard/NearbyImageCard horizontal。 */
		horizontal?: boolean;
		/** 读屏播报文案（role=status 的 aria-label）。 */
		label?: string;
		/** 与真实内容完全一致的栅格类（列数/间距），保证骨架→内容不跳动。 */
		class?: string;
		/** 单卡补充类（如 aspect-[3/4] / h-32 / rounded-md）。 */
		cardClass?: string;
	} = $props();
</script>

<div data-skeleton-root class={cn(className)} role="status" aria-busy="true" aria-label={label}>
	{#each Array(count) as _, i (i)}
		{#if variant === 'feed'}
			<!-- 首页动态卡：外壳与 HomeFeedItem 一致（rounded-2xl + 发丝描边 + 白卡 + p-4/p-5） -->
			<article class={cn('rounded-2xl border border-border-subtle/50 bg-surface-primary p-4 sm:p-5', cardClass)}>
				<div class="flex items-center gap-3">
					<Skeleton class="size-9 shrink-0 rounded-full" />
					<div class="flex min-w-0 flex-1 flex-col gap-1.5">
						<Skeleton class="w-32 rounded-sm" style="height: 14px;" />
						<Skeleton class="w-20 rounded-sm" style="height: 11px;" />
					</div>
					<Skeleton class="h-6 w-20 shrink-0 rounded-full" />
				</div>
				<div class="mt-3 flex flex-col gap-2">
					<Skeleton class="w-full rounded-sm" style="height: 14px;" />
					<Skeleton class="w-4/5 rounded-sm" style="height: 14px;" />
				</div>
				<Skeleton class="mt-3 aspect-[4/3] w-full rounded-xl" />
				<div class="mt-4 flex items-center gap-6">
					<Skeleton class="h-4 w-10 rounded-sm" />
					<Skeleton class="h-4 w-10 rounded-sm" />
					<Skeleton class="h-4 w-10 rounded-sm" />
				</div>
			</article>
		{:else if variant === 'card'}
			<!-- 图片优先内容卡：与 NearbyImageCard/GroupCard 同构（图片 + 信息托盘） -->
			<article
				class={cn(
					'overflow-hidden rounded-md border border-border-subtle/60 bg-surface-primary',
					horizontal && 'flex h-32',
					cardClass
				)}
			>
				<Skeleton class={cn(horizontal ? 'h-full w-2/5' : 'aspect-[4/3] w-full', 'rounded-none')} />
				<div class={cn('flex flex-col gap-2 p-3.5', horizontal && 'min-w-0 flex-1 justify-center')}>
					<Skeleton class="w-3/4 rounded-sm" style="height: 14px;" />
					<Skeleton class="w-1/2 rounded-sm" style="height: 11px;" />
					{#if !horizontal}
						<Skeleton class="mt-1 h-7 w-2/3 rounded-full" />
					{/if}
				</div>
			</article>
		{:else}
			<!-- 竖版人物/瞬间卡：与 UserCard / MomentCard 3:4 竖卡同构 -->
			<article class={cn('relative aspect-[3/4] overflow-hidden rounded-2xl border border-border-subtle/60', cardClass)}>
				<Skeleton class="absolute inset-0 h-full w-full rounded-none" />
				<div class="absolute inset-x-0 bottom-0 flex flex-col gap-1.5 bg-surface-primary/90 p-3.5">
					<Skeleton class="w-24 rounded-sm" style="height: 14px;" />
					<Skeleton class="w-16 rounded-sm" style="height: 11px;" />
				</div>
			</article>
		{/if}
	{/each}
</div>

<script lang="ts">
	import { cn } from '@/lib/utils';

	// SkeletonFeed — 卡片/信息流首屏骨架屏（Round 2：对齐整卡图片卡形状）。
	// 骨架与真实卡同外壳尺寸/圆角/比例，保证「骨架 → 内容」零位移：
	//   feed     首页动态卡：aspect-[4/5]（整卡图片位 + 顶部作者行 + 底部两行文本条）
	//   card     内容卡：aspect-[4/3]（图片位 + 底部两行文本条）
	//   portrait 竖版卡（附近的人/我的瞬间/群组）：aspect-[3/4]
	// horizontal card 为横排紧凑例外（图左 40% + 右信息），沿用旧结构。
	// 动效：外层 .skeleton-block 脉冲；内部信息条用主题表面色（非骨架色）保持层级。
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
		/** card 形态：移动端横排卡（图片左 40% + 信息右）。 */
		horizontal?: boolean;
		/** 读屏播报文案（role=status 的 aria-label）。 */
		label?: string;
		/** 与真实内容完全一致的栅格类（列数/间距），保证骨架→内容不跳动。 */
		class?: string;
		/** 单卡补充类（比例/圆角覆盖）。 */
		cardClass?: string;
	} = $props();
</script>

<div data-skeleton-root class={cn(className)} role="status" aria-busy="true" aria-label={label}>
	{#each Array(count) as _, i (i)}
		{#if variant === 'feed'}
			<!-- 首页动态卡：整卡图片位（4/5）+ 顶部作者行 + 底部两行文本条 -->
			<article
				data-skeleton-card
				class={cn('skeleton-block relative aspect-[4/5] w-full overflow-hidden rounded-lg', cardClass)}
				aria-hidden="true"
			>
				<div class="absolute inset-x-0 top-0 flex items-center gap-3 p-4">
					<div class="size-8 shrink-0 rounded-full bg-surface-primary/60"></div>
					<div class="flex flex-1 flex-col gap-1.5">
						<div class="h-3.5 w-28 rounded-sm bg-surface-primary/70"></div>
						<div class="h-2.5 w-16 rounded-sm bg-surface-primary/55"></div>
					</div>
				</div>
				<div class="absolute inset-x-0 bottom-0 flex flex-col gap-2 p-4">
					<div class="h-4 w-4/5 rounded-sm bg-surface-primary/75"></div>
					<div class="h-3 w-1/3 rounded-sm bg-surface-primary/60"></div>
				</div>
			</article>
		{:else if variant === 'card' && horizontal}
			<article
				data-skeleton-card
				class={cn('relative flex h-32 overflow-hidden rounded-md bg-surface-subtle', cardClass)}
				aria-hidden="true"
			>
				<div class="skeleton-block h-full w-2/5 rounded-none"></div>
				<div class="flex flex-1 flex-col justify-center gap-2 p-3.5">
					<div class="h-3.5 w-3/4 rounded-sm bg-surface-primary/70"></div>
					<div class="h-3 w-1/2 rounded-sm bg-surface-primary/55"></div>
				</div>
			</article>
		{:else}
			<!-- 内容卡（4:3）/ 竖版卡（3:4）：整卡图片位 + 底部两行文本条 -->
			{@const ratioClass = variant === 'portrait' ? 'aspect-[3/4]' : 'aspect-[4/3]'}
			<article
				data-skeleton-card
				class={cn('skeleton-block relative w-full overflow-hidden rounded-lg', ratioClass, cardClass)}
				aria-hidden="true"
			>
				<div class="absolute inset-x-0 bottom-0 flex flex-col gap-1.5 p-3.5">
					<div class="h-3.5 w-3/4 rounded-sm bg-surface-primary/70"></div>
					<div class="h-3 w-1/2 rounded-sm bg-surface-primary/55"></div>
				</div>
			</article>
		{/if}
	{/each}
</div>

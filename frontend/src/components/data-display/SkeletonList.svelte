<script lang="ts">
	import { cn } from '@/lib/utils';
	import Skeleton from './Skeleton.svelte';
	import { isInListContext } from '@/components/list/context';

	// SkeletonList — 列表型首屏加载骨架屏（消息 / 通知 / 搜索 / 关系 / 评论等）。
	// 几何节奏对齐鸿蒙 MessagePage 会话骨架（SkeletonShimmer + Skeleton）：
	//   行内上下 12px；前导圆形 40；文本列行间距 8px、两行间距 8px；
	//   主行高 14 / 宽 55%，次行高 11 / 宽 35%（圆角 12 = RADIUS.SM）。
	// 移动端两端逐像素同节奏；桌面端只按容器宽度适配，行结构不变。
	// 行尾 trailing 用于关系列表的「发消息/关注」动作 chip 占位（h-7 = 28px）。
	let {
		rows = 5,
		leading = 40,
		trailing = false,
		label = '加载中…',
		class: className = '',
		rowClass = '',
		decorative = false
	}: {
		rows?: number;
		/** 前导圆形直径：40 = 鸿蒙会话骨架口径；44 = web 会话/通知行；36 = 搜索结果行；48 = 卡片行。 */
		leading?: 36 | 40 | 44 | 48;
		/** 行尾动作 chip 占位（关系列表的「发消息」按钮）。 */
		trailing?: boolean;
		/** 读屏播报文案（role=status 的 aria-label）。 */
		label?: string;
		/** 外层布局类（与真实列表的 padding/margin 对齐，避免骨架→内容位移）。 */
		class?: string;
		/** 单行覆盖类（如分栏内紧凑行 px-3.5 py-2.5）。 */
		rowClass?: string;
		/** 嵌入其他 role=status 骨架（如详情页）时置 true：不再播报，整块 aria-hidden。 */
		decorative?: boolean;
	} = $props();

	const inList = isInListContext();
	const leadingClass = $derived(
		leading === 36 ? 'size-9' : leading === 40 ? 'size-10' : leading === 44 ? 'size-11' : 'size-12'
	);
</script>

{#if inList}
	<!-- ListContainer 内：整块骨架作为一个 listitem 承载（对齐 InlineLoading 的
	     aria-required-children 处理），status 语义保留在内层。 -->
	<div role="listitem">
		{@render body()}
	</div>
{:else}
	{@render body()}
{/if}

{#snippet body()}
	<div
		data-skeleton-root
		class={cn('flex flex-col', className)}
		role={decorative ? undefined : 'status'}
		aria-busy={decorative ? undefined : 'true'}
		aria-label={decorative ? undefined : label}
		aria-hidden={decorative ? 'true' : undefined}
	>
		{#each Array(rows) as _, i (i)}
			<div data-skeleton-row class={cn('flex items-center gap-3 px-4 py-3', rowClass)}>
				<Skeleton class={cn('shrink-0 rounded-full', leadingClass)} />
				<div class="flex min-w-0 flex-1 flex-col gap-2">
					<Skeleton class="rounded-sm" style="height: 14px; width: 55%;" />
					<Skeleton class="rounded-sm" style="height: 11px; width: 35%;" />
				</div>
				{#if trailing}
					<Skeleton class="h-7 w-16 shrink-0 rounded-full" />
				{/if}
			</div>
		{/each}
	</div>
{/snippet}

<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Inbox } from '@lucide/svelte';
	import { cn } from '@/lib/utils';

	// EmptyState — 空列表唯一状态原语。bare / 非 bare 两态契约保持不变。
	//
	// R13 视觉升级（契约不变）：
	// - 非 bare：图标砖改为 72px 圆形软底（accent 淡底 + accent 图标），标题升为 14px semibold，
	//   描述 12px 次级、限宽可读，动作按钮与上方内容建立 4px 级差，整块 py-20 保持页面级呼吸；
	//   观感从「线框圆圈 + 灰字」提升为「有重心的空态」，不再像加载失败。
	// - bare：列表内嵌空态保持纯图标 + 文案，标题提升到 text-primary 以脱离"灰字失效感"。
	// - 新增 extra snippet（标题/描述之后、动作之前）：承载价值点列表、建议 chips 等内容。
	// - spotlight：超大空态（整页/整块内容区、访客引导）可选装饰层——中心光晕 +
	//   内外双环 + 四向星点；恒 aria-hidden / pointer-events-none / 无动画，
	//   色值全部走 --cst-ambient-* token（三主题 × 明暗自适应），不用 CSS filter /
	//   SVG-only 技巧，鸿蒙端可用径向渐变 + 圆描边 + 圆点原样镜像（几何参数见 report）。
	//   小面板空态不要开 spotlight（留白本身已足够，堆装饰反而吵）。
	let {
		icon,
		title,
		description,
		action,
		extra,
		bare = false,
		iconSize = 64,
		iconTone = 'neutral',
		spotlight = false,
		class: className = ''
	}: {
		icon?: Snippet;
		title: string;
		description?: string;
		action?: Snippet;
		/** 标题/描述之后、动作之前的补充内容（价值点、建议 chips 等）。 */
		extra?: Snippet;
		/** bare：对齐 Figma 列表内空态（无圆形容器，图标直接渲染 + muted 文案）。 */
		bare?: boolean;
		/** 非 bare 图标圆形软底尺寸（px）。 */
		iconSize?: number;
		/** 图标砖色调：neutral（中性软底）/ accent（accent 淡底，用于整页级空态）。 */
		iconTone?: 'neutral' | 'accent';
		/** 场景化空态装饰层：仅超大空态使用（跨端 spec 见 report「跨端同步说明」）。 */
		spotlight?: boolean;
		class?: string;
	} = $props();

	/** 四向星点：外环半径 144px 的 45° 等分位（跨端可镜像的固定几何）。 */
	const SPOTLIGHT_DOTS: ReadonlyArray<readonly [number, number]> = [
		[102, 102],
		[-102, 102],
		[-102, -102],
		[102, -102]
	];

	const iconTileCls = $derived(
		iconTone === 'accent'
			? 'bg-accent-soft text-text-accent-on-soft'
			: 'bg-surface-subtle text-text-secondary'
	);
</script>

{#snippet spotlightLayer()}
	<div aria-hidden="true" data-empty-spotlight class="empty-spotlight">
		<div class="empty-spotlight__glow"></div>
		<div class="empty-spotlight__ring empty-spotlight__ring--outer"></div>
		<div class="empty-spotlight__ring empty-spotlight__ring--inner"></div>
		{#each SPOTLIGHT_DOTS as dot, i (i)}
			<span class="empty-spotlight__dot" style="transform: translate({dot[0]}px, {dot[1]}px)"></span>
		{/each}
	</div>
{/snippet}

<div class={cn('flex flex-col items-center justify-center gap-3 py-20', spotlight && 'relative isolate', className)}>
	{#if spotlight}
		{@render spotlightLayer()}
	{/if}
	{#if bare}
		{#if icon}
			{@render icon()}
		{/if}
		<p class="text-sm font-medium text-text-primary">{title}</p>
		{#if description}
			<p class="max-w-xs text-center text-sm leading-relaxed text-text-secondary">{description}</p>
		{/if}
		{#if extra}
			{@render extra()}
		{/if}
		{#if action}
			{@render action()}
		{/if}
	{:else}
		<div
			class={cn('flex items-center justify-center rounded-full shadow-raised-sm', iconTileCls)}
			style="width:{iconSize}px;height:{iconSize}px;"
		>
			{#if icon}
				{@render icon()}
			{:else}
				<!-- 默认图标：多数调用方只传标题，此前会退化成纯文字空态；
				     有图标的空态更容易被识别为"这里本来就没有内容"而不是"加载失败"。 -->
				<Inbox class="size-7" strokeWidth={1.5} />
			{/if}
		</div>
		<p class="text-md font-semibold text-text-primary">{title}</p>
		{#if description}
			<p class="max-w-xs text-center text-sm leading-relaxed text-text-secondary">{description}</p>
		{/if}
		{#if extra}
			{@render extra()}
		{/if}
		{#if action}
			{@render action()}
		{/if}
	{/if}
</div>

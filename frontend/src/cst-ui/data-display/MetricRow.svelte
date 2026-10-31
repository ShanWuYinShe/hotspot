<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 指标行原语：value 接受 Snippet 或 数字/字符串（自动渲染为纯文本）。
	//
	// 三种形态（一个交互意图只允许一种实现，新增形态走 variant 扩展）：
	// - strip（R13 新增，推荐）：大数字 + 小标签 + 发丝分隔，无灰底块。用于资料页/枢纽 hero/
	//   群组详情等「数字是节奏锚点」的场合；数字 24px（md）/20px（sm），发丝竖线分隔。
	// - capsule：Bento 微岛胶囊（淡底 + 圆角），保留给需要强分组感的既有场景。
	// - default：等分网格 + 1px 描边分隔的密集指标行。
	//
	// 可点项（item.onClick）一律渲染为 button（键盘可达 + focus-ring），整格可点。

	type Tone = 'default' | 'accent' | 'positive' | 'warning';

	export type MetricItem = {
		label: string;
		value: Snippet | number | string;
		tone?: Tone;
		onClick?: () => void;
	};

	const tones: Record<Tone, string> = {
		default: 'text-text-primary',
		accent: 'text-accent-default',
		positive: 'text-status-online',
		warning: 'text-status-warning'
	};

	let {
		items,
		variant = 'default',
		align = 'center',
		size = 'md',
		class: className = ''
	}: {
		items: MetricItem[];
		variant?: 'default' | 'capsule' | 'strip';
		/** strip 形态的对齐：center（枢纽/统计条）/ start（资料卡左对齐列表） */
		align?: 'center' | 'start';
		/** strip 形态数字档位：md=24px / sm=20px */
		size?: 'md' | 'sm';
		class?: string;
	} = $props();

	const stripValueCls = $derived(
		size === 'sm'
			? 'text-xl font-semibold leading-none tracking-tight tabular-nums'
			: 'text-2xl font-semibold leading-none tracking-tight tabular-nums'
	);
	const stripAlignCls = $derived(align === 'start' ? 'items-start text-left' : 'items-center text-center');
</script>

{#snippet metricValue(item: MetricItem, valueCls: string)}
	<span class={cn('whitespace-nowrap', valueCls, tones[item.tone ?? 'default'])}>
		{#if typeof item.value === 'function'}
			{@render item.value()}
		{:else}
			{item.value}
		{/if}
	</span>
{/snippet}

{#if variant === 'strip'}
	<!-- R13：数字 + 发丝分隔。视觉层级：数字（24/semibold/tabular）> 标签（12/secondary），
	     无底色块，靠 8px 竖距与 1px 发丝竖线建立秩序，避免「一排灰药丸」的惰性感。 -->
	<div
		role="group"
		class={cn('flex items-stretch divide-x divide-border-subtle', className)}
	>
		{#each items as item (item.label)}
			{#if item.onClick}
				<button
					type="button"
					onclick={item.onClick}
					class={cn(
						'flex min-w-0 flex-1 cursor-pointer flex-col gap-2 px-4 py-0.5 transition-colors first:pl-0 last:pr-0',
						stripAlignCls,
						'hover-bg rounded-sm focus-ring'
					)}
				>
					{@render metricValue(item, stripValueCls)}
					<span class="w-full truncate text-xs font-medium text-text-secondary">{item.label}</span>
				</button>
			{:else}
				<div class={cn('flex min-w-0 flex-1 flex-col gap-2 px-4 py-0.5 first:pl-0 last:pr-0', stripAlignCls)}>
					{@render metricValue(item, stripValueCls)}
					<span class="w-full truncate text-xs font-medium text-text-secondary">{item.label}</span>
				</div>
			{/if}
		{/each}
	</div>
{:else if variant === 'capsule'}
	<!-- Capsule variant: modern Bento frosted capsules with rhythmic spacing -->
	<div role="group"
		class="grid gap-2.5 {className}"
		style="grid-template-columns:repeat({items.length}, minmax(0, 1fr));"
	>
		{#each items as item (item.label)}
			{#if item.onClick}
				<div class="flex min-w-0 flex-col-reverse items-center text-center">
					<button
						type="button"
						onclick={item.onClick}
						class="hover-btn cursor-pointer flex w-full h-full min-w-0 flex-col-reverse items-center justify-center rounded-xl bg-surface-subtle/40 py-2.5 px-2 text-center transition-all hover:bg-surface-subtle/70 backdrop-blur-xs focus-ring"
					>
						<span class="truncate text-xs text-text-secondary mt-1">{item.label}</span>
						{@render metricValue(item, 'truncate text-xl font-semibold leading-none')}
					</button>
				</div>
			{:else}
				<div class="flex min-w-0 flex-col-reverse items-center justify-center rounded-xl bg-surface-subtle/40 py-2.5 px-2 text-center backdrop-blur-xs">
					<span class="truncate text-xs text-text-secondary mt-1">{item.label}</span>
					{@render metricValue(item, 'truncate text-xl font-semibold leading-none')}
				</div>
			{/if}
		{/each}
	</div>
{:else}
	<!-- Default dense grid with divider line -->
	<div role="group"
		class="grid overflow-hidden rounded-md ring-border bg-border-default gap-px {className}"
		style="grid-template-columns:repeat({items.length}, minmax(0, 1fr));"
	>
		{#each items as item (item.label)}
			<!-- DOM 保持 dt 先 dd 后（HTML 规范/读屏术语-定义顺序），视觉值在上用 col-reverse -->
			{#if item.onClick}
				<div class="flex min-w-0 flex-col-reverse items-center bg-surface-primary text-center">
					<button
						type="button"
						onclick={item.onClick}
						class="hover-btn cursor-pointer flex w-full h-full min-w-0 flex-col-reverse items-center px-2 py-2.5 text-center transition-colors hover:bg-interactive-hover focus-ring"
					>
						<span class="truncate text-xs text-text-secondary">{item.label}</span>
						{@render metricValue(item, 'mb-1 truncate text-xl font-semibold leading-none')}
					</button>
				</div>
			{:else}
				<div class="flex min-w-0 flex-col-reverse items-center bg-surface-primary px-2 py-2.5 text-center">
					<span class="truncate text-xs text-text-secondary">{item.label}</span>
					{@render metricValue(item, 'mb-1 truncate text-xl font-semibold leading-none')}
				</div>
			{/if}
		{/each}
	</div>
{/if}

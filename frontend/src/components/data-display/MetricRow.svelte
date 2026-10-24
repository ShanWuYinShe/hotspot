<script lang="ts">
	import type { Snippet } from 'svelte';

	// 指标行原语：value 接受 Snippet 或 数字/字符串（自动渲染为纯文本）。

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

	let { items, class: className = '' }: { items: MetricItem[]; class?: string } = $props();
</script>

<!-- Dense, horizontally comparable metrics: counts, ratios, and relationship states. -->
<dl
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
					<dt class="truncate text-xs text-text-secondary">{item.label}</dt>
					<dd class="mb-1 truncate text-xl font-semibold leading-none {tones[item.tone ?? 'default']}">
						{#if typeof item.value === 'function'}
							{@render item.value()}
						{:else}
							{item.value}
						{/if}
					</dd>
				</button>
			</div>
		{:else}
			<div class="flex min-w-0 flex-col-reverse items-center bg-surface-primary px-2 py-2.5 text-center">
				<dt class="truncate text-xs text-text-secondary">{item.label}</dt>
				<dd class="mb-1 truncate text-xl font-semibold leading-none {tones[item.tone ?? 'default']}">
					{#if typeof item.value === 'function'}
						{@render item.value()}
					{:else}
						{item.value}
					{/if}
				</dd>
			</div>
		{/if}
	{/each}
</dl>

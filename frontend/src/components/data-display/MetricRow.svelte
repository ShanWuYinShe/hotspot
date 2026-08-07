<script lang="ts">
	import type { Snippet } from 'svelte';

	// 指标行原语：value 接受 Snippet 或 数字/字符串（自动渲染为纯文本）。

	type Tone = 'default' | 'accent' | 'positive' | 'warning';

	export type MetricItem = {
		label: string;
		value: Snippet | number | string;
		tone?: Tone;
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
	class="grid overflow-hidden rounded-sm ring-border {className}"
	style="grid-template-columns:repeat({items.length}, minmax(0, 1fr));gap:1px;background:var(--cst-border-default);"
>
	{#each items as item (item.label)}
		<div class="flex min-w-0 flex-col items-center bg-surface-primary px-2 py-2.5 text-center">
			<dd class="truncate text-xl font-semibold leading-none {tones[item.tone ?? 'default']}">
				{#if typeof item.value === 'function'}
					{@render item.value()}
				{:else}
					{item.value}
				{/if}
			</dd>
			<dt class="mt-1 truncate text-xs text-text-secondary">{item.label}</dt>
		</div>
	{/each}
</dl>

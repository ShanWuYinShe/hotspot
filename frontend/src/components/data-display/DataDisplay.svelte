<script lang="ts">
	import type { Snippet } from 'svelte';

	// 可复用的数据展示原语。
	// 在发明一次性统计块之前先使用这些。
	// value 接受 Snippet 或 数字/字符串（自动渲染为纯文本）。

	type Tone = 'default' | 'accent' | 'positive' | 'warning';

	export type MetricItem = {
		label: string;
		value: Snippet | number | string;
		tone?: Tone;
	};

	const tones: Record<Tone, string> = {
		default: 'text-[var(--text-primary)]',
		accent: 'text-[var(--accent-default)]',
		positive: 'text-[var(--status-online)]',
		warning: 'text-[var(--status-warning)]'
	};

	let { items, class: className = '' }: { items: MetricItem[]; class?: string } = $props();
</script>

<!-- Dense, horizontally comparable metrics: counts, ratios, and relationship states. -->
<dl
	class="grid overflow-hidden rounded-[14px] ring-border {className}"
	style="grid-template-columns:repeat({items.length}, minmax(0, 1fr));gap:1px;background:var(--border-default);"
>
	{#each items as item (item.label)}
		<div class="flex min-w-0 flex-col items-center bg-[var(--surface-primary)] px-2 py-2.5 text-center">
			<dd class="truncate text-[19px] font-semibold leading-none {tones[item.tone ?? 'default']}">
				{#if typeof item.value === 'function'}
					{@render item.value()}
				{:else}
					{item.value}
				{/if}
			</dd>
			<dt class="mt-1 truncate text-[11px] text-[var(--text-secondary)]">{item.label}</dt>
		</div>
	{/each}
</dl>

<script lang="ts">
	import { Search, X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 搜索输入框（对齐 Figma Make SearchField）：
	// rounded-control + input-surface，无内联 padding（由调用方通过 class 提供）。
	// value 非空时显示清空按钮，否则渲染 trailing。

	let {
		value,
		onChange,
		placeholder = '搜索...',
		class: className = '',
		fontSize = 13,
		trailing
	}: {
		value: string;
		onChange: (value: string) => void;
		placeholder?: string;
		class?: string;
		fontSize?: number;
		trailing?: Snippet;
	} = $props();
</script>

<div class={cn('flex items-center gap-2.5 rounded-control input-surface', className)}>
	<Search size={14} color="var(--text-secondary)" />
	<input
		bind:value
		oninput={(e) => onChange(e.currentTarget.value)}
		{placeholder}
		class="min-w-0 flex-1 bg-transparent outline-none"
		style="font-size:{fontSize}px;color:var(--text-primary);"
	/>
	{#if value}
		<button
			type="button"
			class="shrink-0 outline-none"
			aria-label="清空搜索"
			onclick={() => onChange('')}
		>
			<X size={13} color="var(--text-secondary)" />
		</button>
	{:else if trailing}
		{@render trailing()}
	{/if}
</div>

<script lang="ts">
	import { Search, X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 搜索输入框 — 统一图标/清空按钮/主题表面（对齐 Figma Make SearchField：
	// rounded-control + neu-input-surface，value 非空时显示清空按钮，否则渲染 trailing）。

	let {
		value,
		onChange,
		placeholder = '搜索...',
		className = '',
		compact = false,
		trailing
	}: {
		value: string;
		onChange: (value: string) => void;
		placeholder?: string;
		className?: string;
		compact?: boolean;
		trailing?: Snippet;
	} = $props();
</script>

<div class={cn('flex items-center gap-2.5 neu-input-surface rounded-control', compact ? 'px-3 py-2' : 'px-3.5 py-2.5', className)}>
	<Search size={14} color="var(--neu-text-muted)" />
	<input
		bind:value
		oninput={(e) => onChange(e.currentTarget.value)}
		{placeholder}
		class="min-w-0 flex-1 bg-transparent outline-none text-[var(--neu-text)] placeholder:text-[var(--neu-text-muted)] text-[13px]"
	/>
	{#if value}
		<button
			type="button"
			class="flex items-center justify-center rounded-full outline-none shrink-0 w-5 h-5"
			aria-label="清空搜索"
			onclick={() => onChange('')}
		>
			<X size={13} color="var(--neu-text-muted)" strokeWidth={1.8} />
		</button>
	{:else if trailing}
		{@render trailing()}
	{/if}
</div>

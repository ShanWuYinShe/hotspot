<script lang="ts">
	import { Search, X } from '@lucide/svelte';
	import { cn } from '@/lib/utils';

	let {
		value,
		onChange,
		placeholder = '搜索...',
		className = '',
		compact = false
	}: {
		value: string;
		onChange: (value: string) => void;
		placeholder?: string;
		className?: string;
		compact?: boolean;
	} = $props();
</script>

<div class={cn('flex items-center gap-2 neu-input-surface', compact ? 'px-3 py-2 rounded-[14px]' : 'px-3.5 py-2.5 rounded-[18px]', className)}>
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
			class="flex items-center justify-center rounded-full hover:bg-[var(--neu-accent-soft)] w-5 h-5"
			aria-label="清除"
			onclick={() => onChange('')}
		>
			<X size={12} color="var(--neu-text-muted)" strokeWidth={2} />
		</button>
	{/if}
</div>

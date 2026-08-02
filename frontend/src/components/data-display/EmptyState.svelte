<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	let {
		icon,
		title,
		description,
		action,
		class: className = ''
	}: {
		icon?: Snippet;
		title: string;
		description?: string;
		action?: Snippet;
		class?: string;
	} = $props();
</script>

<div class={cn('flex flex-col items-center justify-center py-20 gap-3', className)}>
	{#if icon}
		<div
			class="flex items-center justify-center rounded-full w-16 h-16 border border-[var(--neu-border)]"
			style="box-shadow:6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light);"
		>
			{@render icon()}
		</div>
	{/if}
	<p class="text-[var(--neu-text)] text-[15px] font-medium">{title}</p>
	{#if description}
		<p class="text-[var(--neu-text-muted)] text-center max-w-xs text-[13px] leading-relaxed">{description}</p>
	{/if}
	{#if action}
		{@render action()}
	{/if}
</div>

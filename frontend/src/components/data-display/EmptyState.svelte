<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	let {
		icon,
		title,
		description,
		action,
		bare = false,
		class: className = ''
	}: {
		icon?: Snippet;
		title: string;
		description?: string;
		action?: Snippet;
		/** bare：对齐 Figma 列表内空态（无圆形容器，图标直接渲染 + muted 文案）。 */
		bare?: boolean;
		class?: string;
	} = $props();
</script>

{#if bare}
	<div class={cn('flex flex-col items-center justify-center gap-3 py-20', className)}>
		{#if icon}
			{@render icon()}
		{/if}
		<p class="text-[14px]" style="color:var(--neu-text-muted);">{title}</p>
	</div>
{:else}
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
{/if}

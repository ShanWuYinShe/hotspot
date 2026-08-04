<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	let {
		icon,
		title,
		description,
		action,
		bare = false,
		iconSize = 64,
		class: className = ''
	}: {
		icon?: Snippet;
		title: string;
		description?: string;
		action?: Snippet;
		/** bare：对齐 Figma 列表内空态（无圆形容器，图标直接渲染 + muted 文案）。 */
		bare?: boolean;
		/** 图标圆形容器尺寸（对齐 Figma 桌面消息空态 72px / 默认 64px）。 */
		iconSize?: number;
		class?: string;
	} = $props();
</script>

{#if bare}
	<div class={cn('flex flex-col items-center justify-center gap-3 py-20', className)}>
		{#if icon}
			{@render icon()}
		{/if}
		<p class="text-[14px]" style="color:var(--text-secondary);">{title}</p>
	</div>
{:else}
	<div class={cn('flex flex-col items-center justify-center py-20 gap-3', className)}>
		{#if icon}
			<div
				class="flex items-center justify-center rounded-full border border-[var(--border-default)]"
				style="width:{iconSize}px;height:{iconSize}px;box-shadow:6px 6px 14px var(--elevation-shadow-dark), -6px -6px 14px var(--elevation-shadow-light);"
			>
				{@render icon()}
			</div>
		{/if}
		<p class="text-[var(--text-primary)] text-[15px] font-medium">{title}</p>
		{#if description}
			<p class="text-[var(--text-secondary)] text-center max-w-xs text-[13px] leading-relaxed">{description}</p>
		{/if}
		{#if action}
			{@render action()}
		{/if}
	</div>
{/if}

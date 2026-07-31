<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ArrowLeft } from '@lucide/svelte';
	import ActionIconButton from '@/components/shared/ActionIconButton.svelte';
	import { cn } from '@/lib/utils';

	let {
		title,
		subtitle,
		onBack,
		actions,
		children,
		sticky = false,
		className = ''
	}: {
		title: string;
		subtitle?: string;
		onBack?: () => void;
		actions?: Snippet;
		children?: Snippet;
		sticky?: boolean;
		className?: string;
	} = $props();
</script>

<header
	class={cn('flex items-center gap-3 px-5 py-3', sticky && 'sticky top-0 z-10 bg-[var(--neu-bg)]', className)}
	style={sticky ? 'backdrop-filter:blur(16px);border-bottom:1px solid var(--neu-border);' : ''}
>
	<ActionIconButton icon={ArrowLeft} label="返回" onClick={onBack} size={34} />
	<div class={cn('min-w-0', children ? '' : 'flex-1')}>
		<h1 class="text-[var(--neu-text)] text-lg">{title}</h1>
		{#if subtitle}
			<p class="text-[var(--neu-text-muted)] text-[11px] mt-0.5">{subtitle}</p>
		{/if}
	</div>
	{#if children}
		{@render children()}
	{/if}
	{#if actions}
		{@render actions()}
	{/if}
</header>

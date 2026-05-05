<script lang="ts">
	import type { ComponentType, Snippet } from 'svelte';

	interface Props {
		icon: ComponentType;
		title: string;
		description?: string;
		actionText?: string;
		onAction?: () => void;
		class?: string;
		children?: Snippet;
	}

	let {
		icon: Icon,
		title,
		description,
		actionText,
		onAction,
		class: className = '',
		children
	}: Props = $props();
</script>

<div
	class="flex flex-col items-center justify-center rounded-[24px] py-16 text-center {className}"
	style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
>
	<div
		class="mb-4 flex items-center justify-center rounded-full"
		style="width: 64px; height: 64px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
	>
		<Icon size={24} strokeWidth={1.25} color="var(--neu-text-muted)" />
	</div>
	<p style="font-size: 16px; font-weight: 500; color: var(--neu-text);">{title}</p>
	{#if description}
		<p style="font-size: 13px; color: var(--neu-text-muted); margin-top: 6px; max-width: 260px;">{description}</p>
	{/if}
	{#if actionText && onAction}
		<button
			class="mt-5 rounded-full px-5 py-2 outline-none"
			style="font-size: 13px; font-weight: 500; background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);"
			onclick={onAction}
		>
			{actionText}
		</button>
	{/if}
	{@render children?.()}
</div>

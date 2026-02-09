<script lang="ts">
	import { Avatar as AvatarPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props extends WithoutChildrenOrChild<AvatarPrimitive.RootProps> {
		children?: Snippet;
		class?: string;
		glow?: boolean;
	}

	let { class: className, children, glow = false, ...restProps }: Props = $props();
</script>

<AvatarPrimitive.Root
	class={cn(
		'relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full',
		'ring-[3px] ring-primary/20 dark:ring-primary/50',
		'shadow-sm shadow-black/10',
		'dark:shadow-md dark:shadow-black/30',
		'transition-all duration-200',
		glow && 'hover:ring-4 hover:ring-primary/40 dark:hover:ring-primary/70',
		glow && 'hover:shadow-lg hover:shadow-primary/20',
		glow && 'dark:hover:shadow-xl dark:hover:shadow-primary/40',
		className
	)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</AvatarPrimitive.Root>

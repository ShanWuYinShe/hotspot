<script lang="ts">
	import { Avatar as AvatarPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	interface Props extends WithoutChildrenOrChild<AvatarPrimitive.FallbackProps> {
		children?: Snippet;
		class?: string;
	}

	let { class: className, children, ...restProps }: Props = $props();
</script>

<AvatarPrimitive.Fallback
	class={cn('flex h-full w-full items-center justify-center rounded-full bg-muted', className)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</AvatarPrimitive.Fallback>

<script lang="ts">
	import { Tabs as TabsPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import type { Snippet } from 'svelte';

	interface Props extends WithoutChildrenOrChild<TabsPrimitive.RootProps> {
		children?: Snippet;
		class?: string;
	}

	let { value = $bindable(''), onValueChange, children, ...restProps }: Props = $props();
</script>

<TabsPrimitive.Root 
	bind:value 
	onValueChange={onValueChange} 
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</TabsPrimitive.Root>

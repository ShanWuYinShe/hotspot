<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import type { Snippet } from 'svelte';

	interface Props {
		value?: string;
		onValueChange?: (value: string) => void;
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

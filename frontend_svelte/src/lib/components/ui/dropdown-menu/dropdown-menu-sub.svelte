<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import type { Snippet } from 'svelte';

	interface Props extends WithoutChildrenOrChild<DropdownMenuPrimitive.SubProps> {
		children?: Snippet;
	}

	let { open = $bindable(false), children, ...restProps }: Props = $props();
</script>

<DropdownMenuPrimitive.Sub bind:open {...restProps}>
	{#if children}
		{@render children()}
	{/if}
</DropdownMenuPrimitive.Sub>

<script lang="ts">
	import * as Dialog from '@/components/ui/dialog';
	import SheetShell from '../SheetShell.svelte';
	import type { Snippet } from 'svelte';

	let {
		title = '测试弹层',
		onClose = () => {},
		children
	}: {
		title?: string;
		onClose?: () => void;
		children?: Snippet;
	} = $props();
</script>

<Dialog.Root open={true} onOpenChange={() => {}}>
	<Dialog.Content side="bottom" aria-label={title}>
		<SheetShell {title} {onClose}>
			{#if children}
				{@render children()}
			{:else}
				<div>内容</div>
			{/if}
		</SheetShell>
	</Dialog.Content>
</Dialog.Root>

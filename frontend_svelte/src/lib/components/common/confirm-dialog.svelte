<script lang="ts">
	import {
		Dialog,
		DialogContent,
		DialogDescription,
		DialogFooter,
		DialogHeader,
		DialogTitle
	} from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	interface Props {
		open?: boolean;
		title?: string;
		description?: string;
		confirmText?: string;
		cancelText?: string;
		variant?: 'default' | 'destructive';
		loading?: boolean;
		onconfirm?: () => void | Promise<void>;
		oncancel?: () => void;
	}

	let {
		open = $bindable(false),
		title = '确认操作',
		description = '确定要执行此操作吗？',
		confirmText = '确认',
		cancelText = '取消',
		variant = 'default',
		loading = false,
		onconfirm,
		oncancel
	}: Props = $props();

	function handleCancel() {
		open = false;
		oncancel?.();
	}

	async function handleConfirm() {
		await onconfirm?.();
		open = false;
	}
</script>

<Dialog bind:open>
	<DialogContent class="sm:max-w-md">
		<DialogHeader>
			<DialogTitle>{title}</DialogTitle>
			<DialogDescription>{description}</DialogDescription>
		</DialogHeader>
		<DialogFooter class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
			<Button variant="outline" onclick={handleCancel} disabled={loading}>
				{cancelText}
			</Button>
			<Button
				variant={variant === 'destructive' ? 'destructive' : 'default'}
				onclick={handleConfirm}
				disabled={loading}
			>
				{#if loading}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				{confirmText}
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>

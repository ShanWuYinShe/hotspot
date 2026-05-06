<script lang="ts">
	import {
		Dialog,
		DialogContent,
		DialogDescription,
		DialogFooter,
		DialogHeader,
		DialogTitle
	} from '$lib/components/ui/dialog';
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
	<DialogContent class="sm:max-w-md rounded-[24px]" style="background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: var(--neu-shadow-lg);">
		<DialogHeader>
			<DialogTitle>{title}</DialogTitle>
			<DialogDescription>{description}</DialogDescription>
		</DialogHeader>
		<DialogFooter class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
			<button
				class="rounded-full px-5 py-2 outline-none"
				style="background: var(--neu-bg); color: var(--neu-text); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); font-size: 13px;"
				onclick={handleCancel}
				disabled={loading}
			>
				{cancelText}
			</button>
			<button
				class="rounded-full px-5 py-2 outline-none"
				style="background: {variant === 'destructive' ? 'var(--neu-error)' : 'var(--neu-accent-primary)'}; color: var(--neu-active-text); font-size: 13px; box-shadow: 0 3px 10px {variant === 'destructive' ? 'var(--neu-error-border)' : 'var(--neu-glow)'};"
				onclick={handleConfirm}
				disabled={loading}
			>
				{#if loading}
					<Loader2 size={14} strokeWidth={1.5} class="animate-spin" style="display: inline; vertical-align: middle; margin-right: 6px;" />
				{/if}
				{confirmText}
			</button>
		</DialogFooter>
	</DialogContent>
</Dialog>

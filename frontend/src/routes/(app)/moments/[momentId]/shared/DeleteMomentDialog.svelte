<script lang="ts">
	import { Trash2 } from '@lucide/svelte';
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';

	// DeleteMomentDialog —— 删除动态确认弹窗。
	let { open, deleting, onCancel, onConfirm }: { open: boolean; deleting: boolean; onCancel: () => void; onConfirm: () => void } = $props();
</script>

<Dialog.Root {open} onOpenChange={(next) => (next ? undefined : onCancel())}>
	<Dialog.Content side="center" class="max-w-sm p-0" aria-label="确认删除动态">
		<div class="px-5 pt-6 pb-4">
			<div class="mb-3.5 flex size-10 items-center justify-center rounded-full bg-status-danger/10 text-status-danger ring-1 ring-status-danger/20">
				<Trash2 class="size-5" />
			</div>
			<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">删除这条动态？</Dialog.Title>
			<Dialog.Description class="mt-1 text-sm text-text-secondary leading-relaxed">删除后无法恢复，相关评论也将一并移除。</Dialog.Description>
		</div>
		<div class="flex items-center justify-end gap-2.5 px-5 pb-5">
			<Button
				type="button"
				variant="secondary"
				size="md"
				onclick={onCancel}
			>
				取消
			</Button>
			<Button
				type="button"
				variant="danger"
				size="md"
				disabled={deleting}
				onclick={onConfirm}
			>
				{deleting ? '删除中…' : '确认删除'}
			</Button>
		</div>
	</Dialog.Content>
</Dialog.Root>

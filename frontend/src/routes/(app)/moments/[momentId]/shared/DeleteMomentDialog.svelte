<script lang="ts">
	import * as Dialog from '@/components/ui/dialog';

	// DeleteMomentDialog —— 删除动态确认弹窗。
	let { open, deleting, onCancel, onConfirm }: { open: boolean; deleting: boolean; onCancel: () => void; onConfirm: () => void } = $props();
</script>

<Dialog.Root {open} onOpenChange={(next) => (next ? undefined : onCancel())}>
	<Dialog.Content side="center" class="max-w-sm p-5" aria-label="确认删除动态">
		<p class="text-md text-text-primary">删除这条动态？</p>
		<p class="mt-1 text-sm text-text-secondary">删除后无法恢复，相关评论也将一并移除。</p>
		<div class="mt-4 flex items-center justify-end gap-3">
			<button
				type="button"
				onclick={onCancel}
				class="hover-btn rounded-full px-4 py-2 text-sm text-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
			>
				取消
			</button>
			<button
				type="button"
				disabled={deleting}
				onclick={onConfirm}
				class="hover-btn rounded-full px-5 py-2 text-sm font-medium text-text-on-accent outline-none focus-visible:ring-2 focus-visible:ring-status-danger enabled:bg-status-danger disabled:opacity-muted"
			>
				{deleting ? '删除中…' : '确认删除'}
			</button>
		</div>
	</Dialog.Content>
</Dialog.Root>

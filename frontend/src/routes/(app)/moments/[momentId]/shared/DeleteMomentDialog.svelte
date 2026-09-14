<script lang="ts">
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';

	// DeleteMomentDialog —— 删除动态确认弹窗（ConfirmDialog 原语薄封装，调用方不动）。
	let { open, deleting, onCancel, onConfirm }: { open: boolean; deleting: boolean; onCancel: () => void; onConfirm: () => void } = $props();
</script>

<ConfirmDialog
	{open}
	title="删除这条动态？"
	description="删除后无法恢复，相关评论也将一并移除。"
	confirmLabel={deleting ? '删除中…' : '确认删除'}
	loading={deleting}
	{onCancel}
	{onConfirm}
/>

<script lang="ts">
	// 群公告创建/编辑共用 Dialog（参照 MomentEditDialog 范本）。
	// R13：标题区改用「标题 + 一句说明」两级（原 eyebrow + 正文易读成一整块），
	// 底部动作收敛到 Button 原语（取消=ghost / 发布=primary），与群详情页语言一致。
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
	import Input from '@/components/forms/Input.svelte';
	import Textarea from '@/components/forms/Textarea.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import { Pin, X } from '@lucide/svelte';

	let {
		open,
		mode = 'create',
		initial,
		onClose,
		onSubmit
	}: {
		open: boolean;
		mode?: 'create' | 'edit';
		initial?: { title: string; content: string; isPinned: boolean };
		onClose: () => void;
		onSubmit: (title: string, content: string, isPinned: boolean) => Promise<boolean> | boolean;
	} = $props();

	let title = $state('');
	let content = $state('');
	let isPinned = $state(false);
	let submitting = $state(false);

	// 打开时按 initial 重置（切换创建/编辑或不同公告）
	$effect(() => {
		if (open) {
			title = initial?.title ?? '';
			content = initial?.content ?? '';
			isPinned = initial?.isPinned ?? false;
			submitting = false;
		}
	});

	const canSubmit = $derived(title.trim().length > 0 && content.trim().length > 0 && !submitting);

	const handleSubmit = async () => {
		if (!canSubmit) return;
		submitting = true;
		try {
			const ok = await onSubmit(title.trim(), content.trim(), isPinned);
			if (ok) onClose();
		} catch {
			// 提交异常由父级统一 toast；这里只保证 loading 不会永久卡住。
		} finally {
			submitting = false;
		}
	};
</script>

<Dialog.Root open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
	<Dialog.Content
		side="center"
		class="max-w-md p-5"
		aria-label={mode === 'create' ? '发布群公告' : '编辑群公告'}
	>
		<div class="flex items-start justify-between gap-3">
			<div class="min-w-0">
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">{mode === 'create' ? '发布群公告' : '编辑群公告'}</Dialog.Title>
				<Dialog.Description class="mt-1 text-sm text-text-secondary">{mode === 'create' ? '发布后全体群成员可见' : '修改后立即对全体成员生效'}</Dialog.Description>
			</div>
			<Dialog.Close
				class="hit-slop flex size-8 shrink-0 items-center justify-center rounded-full text-text-secondary focus-ring hover:bg-interactive-hover"
				aria-label="关闭"
			>
				<X size={16} strokeWidth={1.75} />
			</Dialog.Close>
		</div>

		<div class="mt-4 flex flex-col gap-3">
			<!-- 弹层内无可见字段 label：读屏名称走 ariaLabel（与原 sr-only label 等价） -->
			<Input
				id="group-announcement-title"
				value={title}
				onChange={(v) => (title = v.slice(0, 50))}
				placeholder="公告标题"
				ariaLabel="公告标题"
				maxlength={50}
			/>
			<Textarea
				id="group-announcement-content"
				value={content}
				onChange={(v) => (content = v.slice(0, 1000))}
				placeholder="公告内容…"
				ariaLabel="公告内容"
				rows={4}
				maxlength={1000}
			/>
			<TagPill
				size="md"
				class="self-start"
				active={isPinned}
				icon={Pin}
				onClick={() => (isPinned = !isPinned)}
			>
				{isPinned ? '已置顶' : '置顶'}
			</TagPill>
		</div>

		<footer class="mt-5 flex items-center justify-end gap-3">
			<Button type="button" variant="secondary" size="md" onclick={onClose}>取消</Button>
			<Button
				type="button"
				variant="primary"
				size="md"
				disabled={!canSubmit}
				loading={submitting}
				onclick={() => void handleSubmit()}
			>
				{mode === 'create' ? '发布' : '保存'}
			</Button>
		</footer>
	</Dialog.Content>
</Dialog.Root>

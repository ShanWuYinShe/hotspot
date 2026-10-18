<script lang="ts">
	// 群公告创建/编辑共用 Dialog（参照 MomentEditDialog 范本）。
	import * as Dialog from '@/components/ui/dialog';
	import { X, Pin } from '@lucide/svelte';

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
		const ok = await onSubmit(title.trim(), content.trim(), isPinned);
		submitting = false;
		if (ok) onClose();
	};
</script>

<Dialog.Root open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
	<Dialog.Content
		side="center"
		class="max-w-md p-5"
		aria-label={mode === 'create' ? '发布群公告' : '编辑群公告'}
	>
		<div class="flex items-center justify-between">
			<div>
				<p class="eyebrow">{mode === 'create' ? '发布群公告' : '编辑群公告'}</p>
				<p class="mt-1 text-md text-text-primary">{mode === 'create' ? '通知全体群成员' : '修改公告内容'}</p>
			</div>
			<button
				type="button"
				class="hit-slop flex h-8 w-8 items-center justify-center rounded-full text-text-secondary outline-none hover:bg-interactive-hover focus-visible:ring-2 focus-visible:ring-accent-default"
				onclick={onClose}
				aria-label="关闭"
			>
				<X size={16} />
			</button>
		</div>

		<div class="mt-4 flex flex-col gap-3">
			<input
				bind:value={title}
				oninput={(e) => (title = e.currentTarget.value.slice(0, 50))}
				placeholder="公告标题"
				class="w-full input-surface rounded-md px-3 py-2.5 text-sm text-text-primary outline-none focus-ring"
			/>
			<textarea
				bind:value={content}
				rows="4"
				oninput={(e) => (content = e.currentTarget.value.slice(0, 1000))}
				placeholder="公告内容…"
				class="w-full resize-none input-surface rounded-md px-3 py-2.5 text-sm leading-6 text-text-primary outline-none focus-ring"
			></textarea>
			<button
				type="button"
				onclick={() => (isPinned = !isPinned)}
				aria-pressed={isPinned}
				class="flex items-center gap-2 self-start rounded-full px-3 py-1.5 text-xs outline-none transition-colors focus-visible:ring-2 focus-visible:ring-accent-default {isPinned
					? 'bg-accent-soft text-accent-default'
					: 'text-text-secondary hover:bg-interactive-hover'}"
			>
				<Pin size={13} strokeWidth={isPinned ? 2 : 1.5} />
				{isPinned ? '已置顶' : '置顶'}
			</button>
		</div>

		<footer class="mt-5 flex items-center justify-end gap-3">
			<button
				type="button"
				onclick={onClose}
				class="hover-btn rounded-full px-4 py-2 text-sm text-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
			>取消</button>
			<button
				type="button"
				disabled={!canSubmit}
				onclick={handleSubmit}
				class="hover-btn shrink-0 rounded-full px-5 py-2 text-sm font-medium outline-none transition-all focus-visible:ring-2 focus-visible:ring-accent-default disabled:cursor-not-allowed disabled:border disabled:border-border-default disabled:bg-transparent disabled:text-text-secondary enabled:bg-accent-default enabled:text-text-on-accent enabled:shadow-raised-sm enabled:hover:shadow-raised-md"
			>
				{submitting ? '保存中…' : mode === 'create' ? '发布' : '保存'}
			</button>
		</footer>
	</Dialog.Content>
</Dialog.Root>

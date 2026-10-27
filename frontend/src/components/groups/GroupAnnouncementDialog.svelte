<script lang="ts">
	// 群公告创建/编辑共用 Dialog（参照 MomentEditDialog 范本）。
	// R13：标题区改用「标题 + 一句说明」两级（原 eyebrow + 正文易读成一整块），
	// 底部动作收敛到 Button 原语（取消=ghost / 发布=primary），与群详情页语言一致。
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
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
		<div class="flex items-start justify-between gap-3">
			<div class="min-w-0">
				<p class="text-md font-semibold text-text-primary">{mode === 'create' ? '发布群公告' : '编辑群公告'}</p>
				<p class="mt-0.5 text-xs text-text-secondary">{mode === 'create' ? '发布后全体群成员可见' : '修改后立即对全体成员生效'}</p>
			</div>
			<button
				type="button"
				class="hit-slop flex size-8 shrink-0 items-center justify-center rounded-full text-text-secondary focus-ring hover:bg-interactive-hover"
				onclick={onClose}
				aria-label="关闭"
			>
				<X size={16} strokeWidth={1.75} />
			</button>
		</div>

		<div class="mt-4 flex flex-col gap-3">
			<label class="sr-only" for="group-announcement-title">公告标题</label>
			<input
				id="group-announcement-title"
				bind:value={title}
				oninput={(e) => (title = e.currentTarget.value.slice(0, 50))}
				placeholder="公告标题"
				class="w-full input-surface rounded-md px-3 py-2.5 text-sm text-text-primary outline-none focus-ring"
			/>
			<label class="sr-only" for="group-announcement-content">公告内容</label>
			<textarea
				id="group-announcement-content"
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
				class="hit-slop inline-flex h-7 items-center gap-1.5 self-start rounded-full px-3 text-xs font-medium focus-ring transition-colors {isPinned
					? 'bg-accent-soft text-text-accent-on-soft'
					: 'text-text-secondary hover:bg-interactive-hover'}"
			>
				<Pin size={12} strokeWidth={isPinned ? 2 : 1.75} />
				{isPinned ? '已置顶' : '置顶'}
			</button>
		</div>

		<footer class="mt-5 flex items-center justify-end gap-3">
			<Button type="button" variant="soft" size="md" onclick={onClose}>取消</Button>
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

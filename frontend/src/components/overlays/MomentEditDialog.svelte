<script lang="ts">
	// 动态编辑：独立 Dialog 表单（不复用 CreateSheet——后者二合一且经 composer store 被 4 处 layout 挂载，
	// 扩展风险高）。编辑仅改 description/tags/privacy_level，不含图片增删（依赖 ImagePicker，后续批次）。
	import * as Dialog from '@/components/ui/dialog';
	import { X, Lock, MapPin } from '@lucide/svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import { updateMoment } from '@/api';
	import { showToast } from '@/stores/toast.svelte';

	let {
		open,
		momentId,
		initial,
		onClose,
		onUpdated
	}: {
		open: boolean;
		momentId: string;
		initial: { description: string; tags: string[]; privacy_level: 'public' | 'private' };
		onClose: () => void;
		onUpdated: (updated: { description: string; tags: string[]; privacy_level: 'public' | 'private' }) => void;
	} = $props();

	const MOOD_TAGS = ['松弛', '漫游', '咖啡', '展览', '自然', '夜色'];

	// 初值不引用 props（快照由下方 $effect 在打开时回填，与 GroupSettingsSheet 模式一致）
	let text = $state('');
	let selectedTags = $state<Set<string>>(new Set());
	let privacy = $state<'public' | 'private'>('public');
	let submitting = $state(false);

	// 打开时按 initial 重置表单（切换不同动态编辑）
	$effect(() => {
		if (open) {
			text = initial.description;
			selectedTags = new Set(initial.tags);
			privacy = initial.privacy_level;
			submitting = false;
		}
	});

	const toggleTag = (tag: string) => {
		const next = new Set(selectedTags);
		next.has(tag) ? next.delete(tag) : next.add(tag);
		selectedTags = next;
	};

	const canSubmit = $derived(text.trim().length > 0 && !submitting);

	const handleSubmit = async () => {
		if (!canSubmit) return;
		submitting = true;
		try {
			const res = await updateMoment({
				moment_id: momentId,
				description: text.trim(),
				tags: Array.from(selectedTags),
				privacy_level: privacy
			});
			if (res.code === 0) {
				onUpdated({ description: text.trim(), tags: Array.from(selectedTags), privacy_level: privacy });
				onClose();
				showToast('动态已更新', 'success');
			} else {
				showToast(res.message || '保存失败', 'danger');
			}
		} catch {
			showToast('保存失败，请重试', 'danger');
		} finally {
			submitting = false;
		}
	};
</script>

<Dialog.Root open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
	<Dialog.Content
		side="center"
		class="inset-x-4 mx-auto max-w-md rounded-xl bg-surface-primary p-5"
		aria-label="编辑动态"
		style="box-shadow:var(--cst-elevation-modal);"
	>
		<div class="flex items-center justify-between">
			<div>
				<p class="eyebrow">编辑动态</p>
				<p class="mt-1 text-md text-text-primary">修改内容、氛围与可见范围</p>
			</div>
			<button
				type="button"
				class="flex h-8 w-8 items-center justify-center rounded-full text-text-secondary outline-none hover:bg-interactive-hover focus-visible:ring-2 focus-visible:ring-accent-default"
				onclick={onClose}
				aria-label="关闭"
			>
				<X size={16} />
			</button>
		</div>

		<div class="mt-4 input-surface rounded-lg p-4">
			<textarea
				bind:value={text}
				oninput={(e) => (text = e.currentTarget.value.slice(0, 500))}
				placeholder="写下此刻发生的事…"
				class="min-h-24 w-full resize-none bg-transparent text-sm leading-7 text-text-primary placeholder:text-text-secondary"
				aria-label="动态内容"
			></textarea>
			<div class="mt-2 flex items-center justify-end border-t border-border-default pt-2">
				<span class="text-xs tabular-nums text-text-secondary">{text.length}/500</span>
			</div>
		</div>

		<section aria-labelledby="edit-mood-label" class="mt-4">
			<p id="edit-mood-label" class="mb-2 eyebrow">此刻氛围</p>
			<div class="flex flex-wrap gap-2">
				{#each MOOD_TAGS as tag}
					{@const active = selectedTags.has(tag)}
					<TagPill active={active} onClick={() => toggleTag(tag)}>{tag}</TagPill>
				{/each}
			</div>
		</section>

		<section aria-labelledby="edit-privacy-label" class="mt-4 rounded-md ring-1 ring-border-default surface-tinted p-2">
			<p id="edit-privacy-label" class="px-2 pb-2 pt-1 eyebrow">谁可以看到</p>
			<div class="grid grid-cols-2 gap-1">
				<button
					type="button"
					aria-pressed={privacy === 'public'}
					onclick={() => (privacy = 'public')}
					class="flex min-w-0 flex-col items-center gap-1 rounded-sm px-1 py-2 text-center outline-none transition-colors focus-visible:ring-2 focus-visible:ring-accent-default {privacy === 'public'
						? 'bg-accent-soft text-accent-default hover:bg-interactive-active'
						: 'text-text-secondary hover:bg-accent-soft/50'}"
				>
					<MapPin size={14} strokeWidth={privacy === 'public' ? 2 : 1.5} />
					<span class="truncate text-xxs font-medium">公开可见</span>
				</button>
				<button
					type="button"
					aria-pressed={privacy === 'private'}
					onclick={() => (privacy = 'private')}
					class="flex min-w-0 flex-col items-center gap-1 rounded-sm px-1 py-2 text-center outline-none transition-colors focus-visible:ring-2 focus-visible:ring-accent-default {privacy === 'private'
						? 'bg-accent-soft text-accent-default hover:bg-interactive-active'
						: 'text-text-secondary hover:bg-accent-soft/50'}"
				>
					<Lock size={14} strokeWidth={privacy === 'private' ? 2 : 1.5} />
					<span class="truncate text-xxs font-medium">仅自己</span>
				</button>
			</div>
		</section>

		<footer class="mt-5 flex items-center justify-end gap-3">
			<button
				type="button"
				onclick={onClose}
				class="hover-btn rounded-full px-4 py-2 text-sm text-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
			>
				取消
			</button>
			<button
				type="button"
				disabled={!canSubmit}
				onclick={handleSubmit}
				class="hover-btn shrink-0 rounded-full px-5 py-2 text-sm font-medium outline-none transition-all focus-visible:ring-2 focus-visible:ring-accent-default disabled:cursor-not-allowed disabled:border disabled:border-border-default disabled:bg-transparent disabled:text-text-secondary enabled:bg-accent-default enabled:text-text-on-accent enabled:shadow-glow-md"
			>
				{submitting ? '保存中…' : '保存'}
			</button>
		</footer>
	</Dialog.Content>
</Dialog.Root>

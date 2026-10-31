<script lang="ts">
	import { Trash2, TriangleAlert } from '@lucide/svelte';
	import type { Component, Snippet } from 'svelte';
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';

	// ConfirmDialog — 全站唯一的危险/确认弹窗原语（bits-ui Dialog 底座）。
	// 收敛此前 4 种手写形态（DeleteMomentDialog / 设置退出登录 / 群公告删除+退群 /
	// 群主转让）：图标头 + 标题 + 描述 + 取消/确认，一律走这里，不再手写 Dialog
	// 内容结构。图标头用主题预定义色档（danger-bg/border），不用 /10~/50 算术
	// 透明度；按钮用 ui/button 原语（secondary + danger/primary）。
	//
	// R13 视觉收敛：图标头 44px、描述升到 14px（低对比小字在弹窗里最难读）、
	// 标题/描述/按钮三段式留白拉开；行内双钮右对齐，危险动作在右（符合阅读终点）。
	let {
		open,
		title,
		description,
		tone = 'danger',
		icon,
		confirmLabel,
		confirmVariant,
		cancelLabel = '取消',
		loading = false,
		layout = 'row',
		extra,
		onCancel,
		onConfirm
	}: {
		open: boolean;
		title: string;
		description?: string;
		tone?: 'danger' | 'neutral';
		icon?: Component;
		confirmLabel: string;
		/** 确认钮形态：默认 danger 实底；转让群主这类强警示也可用 danger。 */
		confirmVariant?: 'danger' | 'primary';
		cancelLabel?: string;
		loading?: boolean;
		/** row=右对齐 md 双钮；stacked=全宽 lg 纵排（确认在上，移动端拇指友好）。 */
		layout?: 'row' | 'stacked';
		/** 描述与按钮之间的附加内容（如解散原因/封禁天数输入区）。 */
		extra?: Snippet;
		onCancel: () => void;
		onConfirm: () => void;
	} = $props();

	const Icon = $derived(icon ?? (tone === 'danger' ? Trash2 : TriangleAlert));
	const variant = $derived(confirmVariant ?? (tone === 'danger' ? 'danger' : 'primary'));
</script>

<Dialog.Root {open} onOpenChange={(next) => (next ? undefined : onCancel())}>
	<Dialog.Content side="center" class="max-w-sm p-0" aria-label={title}>
		<div class="px-6 pt-6 pb-4">
			<div
				class="mb-4 flex size-11 items-center justify-center rounded-full ring-1 {tone === 'danger'
					? 'bg-danger-bg text-text-status-danger-on-soft ring-status-danger-border'
					: 'bg-accent-soft text-text-accent-on-soft ring-accent-border'}"
			>
				<Icon class="size-5" strokeWidth={1.75} />
			</div>
			<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">{title}</Dialog.Title>
			{#if description}
				<Dialog.Description class="mt-1.5 text-md leading-relaxed text-text-secondary">{description}</Dialog.Description>
			{/if}
			{#if extra}
				<div class="mt-4">
					{@render extra()}
				</div>
			{/if}
		</div>
		{#if layout === 'row'}
			<div class="flex items-center justify-end gap-2.5 px-6 pb-5">
				<Button type="button" variant="secondary" size="md" onclick={onCancel}>
					{cancelLabel}
				</Button>
				<Button type="button" variant={variant} size="md" {loading} onclick={onConfirm}>
					{confirmLabel}
				</Button>
			</div>
		{:else}
			<div class="flex flex-col gap-2 px-6 pb-5">
				<Button type="button" variant={variant} size="lg" class="w-full" {loading} onclick={onConfirm}>
					{confirmLabel}
				</Button>
				<Button type="button" variant="secondary" size="lg" class="w-full" onclick={onCancel}>
					{cancelLabel}
				</Button>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>

<script lang="ts">
	import { CheckCheck, MoreHorizontal, Check } from '@lucide/svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';

	// MessageBubble — 聊天气泡单元（私聊 ChatView / 群聊 GroupChatView 共用）。
	// 发送方向由 sent 决定；recalled 渲染为灰色斜体「消息已撤回」；自己发出的非撤回消息
	// 可挂载 ⋮ 菜单（撤回/编辑，按 canRecall/canEdit 动态显示）；编辑为内联 textarea 态。

	let {
		text,
		sent,
		time,
		showCheck = false,
		messageId = '',
		recalled = false,
		editedAt = '',
		canRecall = false,
		canEdit = false,
		onRecall,
		onEdit
	}: {
		text: string;
		sent: boolean;
		time: string;
		showCheck?: boolean;
		messageId?: string;
		recalled?: boolean;
		editedAt?: string;
		canRecall?: boolean;
		canEdit?: boolean;
		onRecall?: (id: string) => void;
		onEdit?: (id: string, content: string) => void;
	} = $props();

	let menuEl = $state<HTMLElement | null>(null);
	let menuOpen = $state(false);
	let editing = $state(false);
	let draft = $state('');

	// $derived：keyed each 复用实例时 props（recalled/canRecall/canEdit）会变化，需响应式重算
	const hasMenu = $derived(sent && !recalled && (canRecall || canEdit));

	const startEdit = () => {
		draft = text;
		editing = true;
		menuOpen = false;
	};
	const confirmEdit = () => {
		const next = draft.trim();
		if (next && onEdit) onEdit(messageId, next);
		editing = false;
	};
	const handleRecall = () => {
		menuOpen = false;
		onRecall?.(messageId);
	};
</script>

<div class="animate-fade-in flex {sent ? 'justify-end' : 'justify-start'}">
	<div
		class="max-w-[76%] px-3.5 py-2.5 {sent
			? 'bg-accent-default shadow-glow-md'
			: 'bg-surface-primary shadow-raised-sm border border-border-default'} {sent
			? 'rounded-[var(--cst-shape-md)_var(--cst-shape-sm)_var(--cst-shape-md)_var(--cst-shape-md)]'
			: 'rounded-[var(--cst-shape-sm)_var(--cst-shape-md)_var(--cst-shape-md)_var(--cst-shape-md)]'}"
	>
		{#if editing}
			<div class="flex flex-col gap-1.5">
				<textarea
					bind:value={draft}
					rows="2"
					class="w-full resize-none rounded-sm bg-surface-primary px-2 py-1 text-sm text-text-primary outline-none focus-ring"
				></textarea>
				<div class="flex items-center justify-end gap-2">
					<button
						type="button"
						onclick={() => (editing = false)}
						class="rounded-full px-2 py-0.5 text-xxs text-text-secondary outline-none hover-btn focus-visible:ring-2 focus-visible:ring-accent-default"
					>取消</button>
					<button
						type="button"
						onclick={confirmEdit}
						disabled={!draft.trim()}
						class="flex items-center gap-0.5 rounded-full px-2 py-0.5 text-xxs text-text-on-accent enabled:bg-accent-default disabled:opacity-50 outline-none hover-btn focus-visible:ring-2 focus-visible:ring-accent-default"
					>
						<Check size={11} strokeWidth={2} />完成
					</button>
				</div>
			</div>
		{:else}
			<p
				class="text-sm leading-card {recalled ? 'italic opacity-soft' : ''} {sent
					? 'text-text-on-accent'
					: recalled
						? 'text-text-tertiary'
						: 'text-text-primary'}"
			>
				{recalled ? '消息已撤回' : text}
			</p>
			<p
				class="mt-1 flex items-center gap-0.75 text-xxs {sent
					? 'justify-end text-text-on-accent opacity-soft'
					: 'justify-start text-text-secondary'}"
			>
				{time}
				{#if editedAt && !recalled}
					<span class="opacity-soft">· 已编辑</span>
				{/if}
				{#if sent && showCheck}
					<CheckCheck class="size-3 opacity-emphasis" strokeWidth={2} />
				{/if}
			</p>
		{/if}
	</div>

	{#if hasMenu && !editing}
		<DropdownMenu.Root bind:open={menuOpen}>
			<div bind:this={menuEl} class="relative flex items-center">
				<button
					type="button"
					onclick={() => (menuOpen = !menuOpen)}
					class="ml-1 flex h-6 w-6 self-center items-center justify-center rounded-full text-text-tertiary outline-none hover:bg-interactive-hover hover-btn scale-btn focus-visible:ring-2 focus-visible:ring-accent-default"
					aria-label="消息操作"
				>
					<MoreHorizontal size={14} strokeWidth={2} />
				</button>
			</div>
			<DropdownMenu.Content customAnchor={menuEl}>
				{#if canEdit}
					<DropdownMenu.Item onSelect={startEdit}>编辑</DropdownMenu.Item>
				{/if}
				{#if canRecall}
					<DropdownMenu.Item class="text-status-danger" onSelect={handleRecall}>撤回</DropdownMenu.Item>
				{/if}
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	{/if}
</div>

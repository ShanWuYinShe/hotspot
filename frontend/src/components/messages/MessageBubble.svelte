<script lang="ts">
	import { CheckCheck, MoreHorizontal, Check, Flag } from '@lucide/svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';

	// MessageBubble — 聊天气泡单元（私聊 ChatView / 群聊 GroupChatView 共用）。
	// 发送方向由 sent 决定；recalled 渲染为灰色斜体「消息已撤回」；自己发出的非撤回消息
	// 可挂载 ⋮ 菜单（撤回/编辑，按 canRecall/canEdit 动态显示）；编辑为内联 textarea 态。
	// 收到的消息（!sent）可挂载 ⋮ 菜单「举报」（canReport，reports/create target_type='message'）。

	let {
		text,
		sent,
		time,
		senderName = '',
		showCheck = false,
		messageId = '',
		recalled = false,
		editedAt = '',
		canRecall = false,
		canEdit = false,
		canReport = false,
		onRecall,
		onEdit,
		onReport
	}: {
		text: string;
		sent: boolean;
		time: string;
		senderName?: string;
		showCheck?: boolean;
		messageId?: string;
		recalled?: boolean;
		editedAt?: string;
		canRecall?: boolean;
		canEdit?: boolean;
		/** 收到的消息可举报（own messages 服务端 10400，不展示入口） */
		canReport?: boolean;
		onRecall?: (id: string) => void;
		onEdit?: (id: string, content: string) => void;
		onReport?: (id: string) => void;
	} = $props();

	let menuEl = $state<HTMLElement | null>(null);
	let menuOpen = $state(false);
	let editing = $state(false);
	let draft = $state('');

	// $derived：keyed each 复用实例时 props（recalled/canRecall/canEdit）会变化，需响应式重算
	const hasMenu = $derived(
		!recalled && ((sent && (canRecall || canEdit)) || (!sent && canReport))
	);

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
	const handleReport = () => {
		menuOpen = false;
		onReport?.(messageId);
	};
</script>

<div class="group animate-fade-in flex flex-col {sent ? 'items-end' : 'items-start'}">
	{#if !sent && senderName}
		<p class="mb-0.5 text-xxs text-text-secondary px-1">{senderName}</p>
	{/if}
	<div
		class="max-w-chat-bubble px-3.5 py-2.5 {sent
			? 'bg-accent-default'
			: 'bg-surface-primary'} {sent
			? 'rounded-[var(--cst-shape-md)_var(--cst-shape-sm)_var(--cst-shape-md)_var(--cst-shape-md)]'
			: 'rounded-[var(--cst-shape-sm)_var(--cst-shape-md)_var(--cst-shape-md)_var(--cst-shape-md)]'}"
	>
		{#if editing}
			<div class="flex flex-col gap-1.5">
				<textarea
					bind:value={draft}
					rows="2"
					aria-label="编辑消息内容"
					onkeydown={(e) => {
						// Esc 取消编辑（与「取消」按钮等价），Enter 不提交以支持多行换行
						if (e.key === 'Escape') {
							e.preventDefault();
							editing = false;
						}
					}}
					class="w-full resize-none rounded-sm bg-surface-primary px-2 py-1 text-sm text-text-primary outline-none focus-ring"
				></textarea>
				<div class="flex items-center justify-end gap-2">
					<button
						type="button"
						onclick={() => (editing = false)}
						class="link-btn text-xxs text-text-secondary hover:text-accent-default outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
					>取消</button>
					<button
						type="button"
						onclick={confirmEdit}
						disabled={!draft.trim()}
						class="hover-btn flex items-center gap-0.5 rounded-full px-2.5 py-1 text-xxs text-text-on-accent outline-none focus-visible:ring-2 focus-visible:ring-accent-default disabled:cursor-not-allowed disabled:opacity-faint enabled:bg-accent-default enabled:shadow-raised-sm enabled:hover:shadow-raised-md"
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
						? 'text-text-secondary'
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
				<!-- 桌面（fine pointer）：仅行 hover 时显现，降噪（Telegram/Slack 模式）；
				     触屏常显（无 hover，常显是唯一入口）。菜单开启时保持可见。 -->
				<button
					type="button"
					onclick={() => (menuOpen = !menuOpen)}
					class="desktop-reveal hit-slop hover-btn scale-btn ml-1 flex h-8 w-8 self-center items-center justify-center rounded-full text-text-secondary outline-none hover:bg-interactive-hover focus-visible:ring-2 focus-visible:ring-accent-default"
					aria-label="消息操作"
					aria-expanded={menuOpen}
				>
					<MoreHorizontal size={15} strokeWidth={2} />
				</button>
			</div>
			<DropdownMenu.Content customAnchor={menuEl}>
				{#if canEdit}
					<DropdownMenu.Item onSelect={startEdit}>编辑</DropdownMenu.Item>
				{/if}
				{#if canRecall}
					<!-- 危险操作：hover/键盘高亮走 danger 淡底档（与普通项 accent-soft 区分） -->
					<DropdownMenu.Item class="text-status-danger hover:bg-danger-bg data-[highlighted]:bg-danger-bg" onSelect={handleRecall}>撤回</DropdownMenu.Item>
				{/if}
				{#if canReport}
					<DropdownMenu.Item class="text-status-danger hover:bg-danger-bg data-[highlighted]:bg-danger-bg" onSelect={handleReport}>
						<Flag class="size-3.5" strokeWidth={1.75} />
						举报
					</DropdownMenu.Item>
				{/if}
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	{/if}
</div>

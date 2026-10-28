<script lang="ts">
	import { CheckCheck, MoreHorizontal, Check, Flag } from '@lucide/svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';

	// MessageBubble — 聊天气泡单元（私聊 ChatView / 群聊 GroupChatView 共用）。
	// 发送方向由 sent 决定；recalled 渲染为中性斜体「消息已撤回」；自己发出的非撤回消息
	// 可挂载 ⋮ 菜单（撤回/编辑，按 canRecall/canEdit 动态显示）；编辑为内联 textarea 态。
	// 收到的消息（!sent）可挂载 ⋮ 菜单「举报」（canReport，reports/create target_type='message'）。
	//
	// R16 排版重做：正文 14px（text-md）+ leading-card（1.7，design-system「卡片/聊天气泡正文」档）；
	// 时间戳 11px（xs）、群聊发送者名 12px（sm）；圆角统一 rounded-md（16）——
	// 旧实现 [md sm md md] 把 12px 缺口放在上方靠发送者一侧，观感碎且方向反直觉。
	// 对比度：自己气泡上的正文/时间戳一律 full on-accent（旧 opacity-soft 0.7 实测 3.97:1 不达标）；
	// 撤回消息改用中性气泡底色，避免 accent 底色上的低对比斜体。

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

	// 气泡外皮：撤回 = 中性底（两个方向统一，低对比问题归零）；
	// 自己 = accent 实底（无投影，实底自带层级）；对方 = 白色气泡 + 极轻投影
	// （forest/inkwash 的聊天画布与白卡几乎同亮度，只靠投影区分，不能去掉）。
	const bubbleClass = $derived(
		recalled
			? 'bg-surface-subtle'
			: sent
				? 'bg-accent-default'
				: 'bg-surface-primary shadow-xs'
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
		<p class="mb-1 px-1 text-sm font-medium text-text-secondary">{senderName}</p>
	{/if}
	<div class="relative max-w-chat-bubble rounded-md px-4 py-2 {bubbleClass}">
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
					class="w-full resize-none rounded-sm bg-surface-primary px-2 py-1 text-md text-text-primary outline-none focus-ring"
				></textarea>
				<div class="flex items-center justify-end gap-2">
					<button
						type="button"
						onclick={() => (editing = false)}
						class="link-btn text-xs text-text-secondary hover:text-accent-default outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
					>取消</button>
					<button
						type="button"
						onclick={confirmEdit}
						disabled={!draft.trim()}
						class="hover-btn flex items-center gap-0.5 rounded-full px-2.5 py-1 text-xs text-text-on-accent outline-none focus-visible:ring-2 focus-visible:ring-accent-default disabled:cursor-not-allowed disabled:opacity-faint enabled:bg-accent-default enabled:shadow-raised-sm enabled:hover:shadow-raised-md"
					>
						<Check size={12} strokeWidth={2} />完成
					</button>
				</div>
			</div>
		{:else}
			<p
				class="text-md leading-card {recalled
					? 'italic text-text-secondary'
					: sent
						? 'text-text-on-accent'
						: 'text-text-primary'}"
			>
				{recalled ? '消息已撤回' : text}
			</p>
			<!-- 时间戳：气泡内右下/左下元信息，11px（xs 档）与正文拉开一档；
			     颜色不叠 opacity（实底上会跌破 4.5:1），靠字号/字重区分主次 -->
			<p
				class="mt-1 flex items-center gap-0.75 text-xs leading-none tabular-nums {recalled
					? 'justify-start text-text-secondary'
					: sent
						? 'justify-end text-text-on-accent'
						: 'justify-start text-text-secondary'}"
			>
				{time}
				{#if editedAt && !recalled}
					<span>· 已编辑</span>
				{/if}
				{#if sent && showCheck && !recalled}
					<CheckCheck class="size-3.5 opacity-emphasis" strokeWidth={2} />
				{/if}
			</p>
		{/if}
			<!-- 消息操作入口：绝对定位到气泡内侧（sent=左侧 / received=右侧），不占文档流。
			     旧实现把 ⋯ 放在气泡下方，隐藏态仍占 36px 高——短会话底对齐后会把最后一条消息
			     顶高一大截（实测底部多出 ~40px 空隙）。桌面仅 hover 显现，触屏常显。 -->
			{#if hasMenu && !editing}
				<DropdownMenu.Root bind:open={menuOpen}>
					<div bind:this={menuEl} class="absolute top-1/2 flex -translate-y-1/2 items-center {sent ? 'right-full mr-1' : 'left-full ml-1'}">
						<button
							type="button"
							onclick={() => (menuOpen = !menuOpen)}
							class="desktop-reveal hit-slop hover-btn scale-btn flex h-9 w-9 items-center justify-center rounded-full text-text-secondary outline-none hover:bg-interactive-hover focus-visible:ring-2 focus-visible:ring-accent-default"
							aria-label="消息操作"
							aria-expanded={menuOpen}
						>
							<MoreHorizontal size={16} strokeWidth={2} />
						</button>
					</div>
					<DropdownMenu.Content customAnchor={menuEl}>
						{#if canEdit}
							<DropdownMenu.Item onSelect={startEdit}>编辑</DropdownMenu.Item>
						{/if}
						{#if canRecall}
							<!-- 危险操作：hover/键盘高亮走 danger 淡底档（与普通项 accent-soft 区分） -->
							<DropdownMenu.Item class="text-status-danger hover:text-text-status-danger-on-soft data-[highlighted]:text-text-status-danger-on-soft hover:bg-danger-bg data-[highlighted]:bg-danger-bg" onSelect={handleRecall}>撤回</DropdownMenu.Item>
						{/if}
						{#if canReport}
							<DropdownMenu.Item class="text-status-danger hover:text-text-status-danger-on-soft data-[highlighted]:text-text-status-danger-on-soft hover:bg-danger-bg data-[highlighted]:bg-danger-bg" onSelect={handleReport}>
								<Flag class="size-3.5" strokeWidth={1.75} />
								举报
							</DropdownMenu.Item>
						{/if}
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			{/if}
		</div>
	</div>

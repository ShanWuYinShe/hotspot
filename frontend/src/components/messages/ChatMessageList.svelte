<script lang="ts">
	import { MessageCircle } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import MessageBubble from '@/components/messages/MessageBubble.svelte';
	import ChatDatePill from '@/components/messages/ChatDatePill.svelte';
	import { inRecallWindow, type ChatController, type ChatMessage } from '@/lib/chat.svelte';

	// ChatMessageList —— 私聊/群聊共享的消息区（加载/错误/空态 + 日期分组气泡）。
	// 视图差异仅 canRecall 判定（群聊管理员可撤回他人消息），由 props 注入。
	let {
		chat,
		canRecall,
		showSenderNames = false,
		onReport
	}: {
		chat: ChatController;
		canRecall: (msg: ChatMessage) => boolean;
		/** 群聊= true：incoming 气泡上方显示发送者昵称 */
		showSenderNames?: boolean;
		/** 收到消息的举报入口（reports/create target_type='message'）；缺省不渲染 */
		onReport?: (messageId: string) => void;
	} = $props();
</script>

<div bind:this={chat.scrollEl} onscroll={chat.handleScroll} class="mx-auto w-full max-w-3xl lg:max-w-5xl flex-1 overflow-y-auto no-scrollbar" style="overscroll-behavior:contain;">
	{#if chat.loading}
		<!-- 加载态统一 InlineLoading（全站三态标配，不再自绘 spinner） -->
		<InlineLoading label="加载消息…" class="h-full justify-center" />
	{:else if chat.loadError}
		<!-- 错误态统一 LoadError（与全站一致），不再各页手搓文案 + 按钮 -->
		<LoadError message="消息加载失败，请检查网络后重试" onRetry={() => void chat.reload()} class="h-full" />
	{:else if chat.messages.length === 0}
		<!-- 空会话：统一 EmptyState 空态（对齐全局空态契约） -->
		<EmptyState class="h-full" title="还没有消息" description="打个招呼，开始这段对话吧" bare>
			{#snippet icon()}
				<MessageCircle class="size-6 text-text-secondary" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else}
		<div class="animate-fade-in flex flex-col gap-2 px-4 pt-4 pb-3">
			{#if chat.loadingMoreFailed}
				<!-- 分页失败可见化（layout 第 6 条）：轻量重试入口，不遮蔽已加载消息 -->
				<div class="flex items-center justify-center gap-2 py-3 text-xs text-text-secondary">
					<span>更早的消息加载失败</span>
					<button type="button" class="link-btn" onclick={() => void chat.loadMore()}>重试</button>
				</div>
			{:else if chat.loadingMore}
				<InlineLoading label="加载更早消息…" class="py-3" />
			{:else if !chat.hasMore}
				<!-- 历史耗尽：ListEnd 收尾（layout 第 6 条），区分「已到顶」与「卡住」 -->
				<ListEnd />
			{/if}
			{#each chat.dateGroups as group (group.label)}
				<ChatDatePill label={group.label} />
				{#each group.messages as msg (msg.id)}
					<MessageBubble
						text={msg.text}
						sent={msg.sent}
						time={msg.time}
						senderName={showSenderNames ? msg.senderName : ''}
						showCheck={msg.id === chat.lastSentId}
						messageId={msg.id}
						recalled={msg.recalled}
						editedAt={msg.editedAt}
						canRecall={canRecall(msg)}
						canEdit={msg.sent}
						canReport={!msg.sent && Boolean(onReport)}
						onRecall={(id) => void chat.handleRecall(id)}
						onEdit={(id, content) => void chat.handleEdit(id, content)}
						onReport={(id) => onReport?.(id)}
					/>
				{/each}
			{/each}
		</div>
	{/if}
</div>

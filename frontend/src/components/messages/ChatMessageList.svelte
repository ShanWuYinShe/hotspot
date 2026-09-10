<script lang="ts">
	import { MessageCircle } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import MessageBubble from '@/components/messages/MessageBubble.svelte';
	import ChatDatePill from '@/components/messages/ChatDatePill.svelte';
	import { inRecallWindow, type ChatController, type ChatMessage } from '@/lib/chat.svelte';

	// ChatMessageList —— 私聊/群聊共享的消息区（加载/错误/空态 + 日期分组气泡）。
	// 视图差异仅 canRecall 判定（群聊管理员可撤回他人消息），由 props 注入。
	let {
		chat,
		canRecall,
		showSenderNames = false
	}: {
		chat: ChatController;
		canRecall: (msg: ChatMessage) => boolean;
		/** 群聊= true：incoming 气泡上方显示发送者昵称 */
		showSenderNames?: boolean;
	} = $props();
</script>

<div bind:this={chat.scrollEl} onscroll={chat.handleScroll} class="flex-1 overflow-y-auto no-scrollbar" style="overscroll-behavior:contain;">
	{#if chat.loading}
		<div class="flex h-full items-center justify-center" role="status" aria-label="加载中">
			<div class="spinner size-7"></div>
		</div>
	{:else if chat.loadError}
		<!-- 错误态统一 LoadError（与全站一致），不再各页手搓文案 + 按钮 -->
		<LoadError message="消息加载失败，请检查网络后重试" onRetry={() => void chat.reload()} class="h-full" />
	{:else if chat.messages.length === 0}
		<!-- 空会话：统一 EmptyState 空态（对齐全局空态契约） -->
		<EmptyState iconSize={56} class="h-full" title="还没有消息" description="打个招呼，开始这段对话吧">
			{#snippet icon()}
				<MessageCircle class="size-5.5 text-text-secondary" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else}
		<div class="animate-fade-in flex flex-col gap-2 px-4 pt-4 pb-3">
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
						onRecall={(id) => void chat.handleRecall(id)}
						onEdit={(id, content) => void chat.handleEdit(id, content)}
					/>
				{/each}
			{/each}
		</div>
	{/if}
</div>

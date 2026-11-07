<script lang="ts">
	import { goto } from '$app/navigation';
	import { Plus, MessageSquareDashed, Pin, BellOff, Trash2, X } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import ChatView from '@/components/messages/ChatView.svelte';
	import GroupChatView from '@/components/messages/GroupChatView.svelte';
	import Button from '@/components/ui/button.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import OnlineStrip from '../shared/OnlineStrip.svelte';
	import ChatListBody from '../shared/ChatListBody.svelte';
	import ChatEmptyPanel from '../shared/ChatEmptyPanel.svelte';
	import NewConversationSheet from '@/components/overlays/NewConversationSheet.svelte';
	import CreateGroupSheet from '@/components/overlays/CreateGroupSheet.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import { ROUTES } from '@/lib/routes';
	import type { ChatListData } from '../shared/useChatListData.svelte';

	let { data }: { data: ChatListData } = $props();
</script>

<div class="flex h-full flex-1 gap-3 overflow-hidden bg-surface-page p-3">
	<!-- Left: chat list column (w-80 = 320px) -->
	<div class="flex w-80 shrink-0 flex-col overflow-hidden rounded-lg bg-surface-island shadow-xs">
		<PageHeader title="消息" sticky titleSize="xl" contentClassName="px-4" island={false}>
			{#snippet trailing()}
				{#if !data.needsAuth}
					<div class="flex items-center gap-2">
						{#if data.totalUnread > 0}
							<span class="text-text-secondary text-xs">{data.totalUnread} 条未读</span>
						{:else}
							<span class="text-text-secondary text-xs">全部已读</span>
						{/if}
						<Button
							type="button"
							variant="soft"
							size="sm"
							class="hit-slop h-8 px-2.5"
							aria-label="发起新会话"
							onclick={() => {
								data.newChatTab = 'chat';
								data.showNewChat = true;
							}}
						>
							<Plus class="size-3.5 text-text-secondary" strokeWidth={1.5} aria-hidden="true" />
							新会话
						</Button>
					</div>
				{/if}
			{/snippet}

			{#snippet below()}
				{#if !data.needsAuth}
					<div class="mt-3">
						<SearchInput
							value={data.searchQuery}
							onChange={(v) => (data.searchQuery = v)}
							placeholder="搜索会话…"
							ariaLabel="搜索会话"
						/>
					</div>
				{/if}
			{/snippet}
		</PageHeader>

		<div class="flex flex-1 flex-col overflow-y-auto px-4 pb-4">
			{#if data.onlineChats.length > 0}
				<div class="pt-2">
					<OnlineStrip
						chats={data.onlineChats}
						onSelect={(c) => {
							data.selectChat(c);
						}}
					/>
				</div>
			{/if}

			{#if !data.needsAuth}
				<div class="mt-3 mb-1.5">
					<p class="eyebrow">最近</p>
				</div>
			{/if}

			<ChatListBody
				chats={data.filtered}
				loadError={data.loadError}
				needsAuth={data.needsAuth}
				loading={data.loading}
				onRetry={() => data.reload()}
				onSelect={(chat) => data.selectChat(chat)}
				onMore={(chat) => (data.moreTarget = chat)}
				selectedId={data.selectedId}
				compact
				fill
				searchQuery={data.searchQuery}
			/>
		</div>
	</div>

	<!-- Right: chat view or empty state -->
	<div class="min-w-0 flex-1 overflow-hidden rounded-lg bg-surface-island shadow-xs">
		{#if data.queryGroup || (data.selectedChat && data.selectedChat.kind === 'group')}
			<div class="h-full">
				<GroupChatView groupId={data.queryGroup || String(data.selectedChat?.id ?? '')} compact showBackButton={false} />
			</div>
		{:else if data.queryPeer || (data.selectedChat && data.selectedChat.kind !== 'group')}
			<div class="h-full">
				<ChatView
					peerUserId={data.queryPeer || data.selectedChat?.peerUserId || String(data.selectedChat?.id ?? '')}
					compact
					contact={data.selectedChat ? { name: data.selectedChat.name, avatar: data.selectedChat.avatar, online: data.selectedChat.online } : undefined}
					showBackButton={false}
				/>
			</div>
		{:else}
			<ChatEmptyPanel />
		{/if}
	</div>

	<!-- 新会话对话框 -->
	<Dialog.Root open={data.showNewChat} onOpenChange={(next) => (next ? undefined : (data.showNewChat = false))}>
		<Dialog.Content
			side="center"
			class="overflow-hidden"
			aria-label="发起新会话"
			style="width:27.5rem;height:var(--cst-sheet-height);max-height:var(--cst-sheet-max-height);"
		>
			<NewConversationSheet
				defaultTab={data.newChatTab}
				onClose={() => (data.showNewChat = false)}
				onNewChat={(id) => {
					data.selectedId = id;
					data.showNewChat = false;
				}}
				onCreateGroup={() => {
					data.showNewChat = false;
					data.showCreateGroup = true;
				}}
			/>
		</Dialog.Content>
	</Dialog.Root>

	<!-- 创建群组对话框 -->
	<Dialog.Root open={data.showCreateGroup} onOpenChange={(next) => (next ? undefined : (data.showCreateGroup = false))}>
		<Dialog.Content side="center" class="overflow-hidden w-120" aria-label="创建群组" style="height:var(--cst-sheet-height);max-height:var(--cst-sheet-max-height)">
			<CreateGroupSheet
				onClose={() => (data.showCreateGroup = false)}
				onCreated={(groupId) => {
					data.showCreateGroup = false;
					void goto(ROUTES.groupMessage(groupId));
				}}
			/>
		</Dialog.Content>
	</Dialog.Root>

	<!-- 会话操作菜单 -->
	<Dialog.Root open={!!data.moreTarget} onOpenChange={(next) => (next ? undefined : (data.moreTarget = null))}>
		<Dialog.Content side="center" class="w-80 p-5" aria-label={data.moreTarget ? `${data.moreTarget.name} 的会话操作` : '会话操作'}>
			{#if data.moreTarget}
				<div class="flex items-center justify-between">
					<div class="min-w-0">
						<p class="truncate text-md font-semibold text-text-primary">{data.moreTarget.name}</p>
						<p class="mt-0.5 text-xs text-text-secondary">会话操作</p>
					</div>
					<ActionIconButton icon={X} label="关闭" onClick={() => (data.moreTarget = null)} />
				</div>
				<div class="mt-4 flex flex-col gap-2">
					<Button
						type="button"
						variant="soft"
						size="lg"
						class="w-full justify-start"
						onclick={() => {
							const chat = data.moreTarget;
							data.moreTarget = null;
							if (chat) data.togglePin(chat);
						}}
					>
						<Pin class="size-4" strokeWidth={1.75} />
						{data.moreTarget.isPinned ? '取消置顶' : '置顶会话'}
					</Button>
					<Button
						type="button"
						variant="soft"
						size="lg"
						class="w-full justify-start"
						onclick={() => {
							const chat = data.moreTarget;
							data.moreTarget = null;
							if (chat) data.toggleMute(chat);
						}}
					>
						<BellOff class="size-4" strokeWidth={1.75} />
						{data.moreTarget.isMuted ? '关闭免打扰' : '开启免打扰'}
					</Button>
					{#if data.moreTarget.kind !== 'group'}
						<Button
							type="button"
							variant="danger-soft"
							size="lg"
							class="w-full justify-start"
							onclick={() => {
								data.deleteTarget = data.moreTarget;
								data.moreTarget = null;
							}}
						>
							<Trash2 class="size-4" strokeWidth={1.75} />
							删除会话
						</Button>
					{/if}
				</div>
			{/if}
		</Dialog.Content>
	</Dialog.Root>

	<!-- 删除会话确认 -->
	<ConfirmDialog
		open={data.deleteTarget !== null}
		title={`删除与「${data.deleteTarget?.name ?? ''}」的会话？`}
		description="删除后将清空本机的聊天记录，对方不受影响。"
		icon={MessageSquareDashed}
		confirmLabel="删除会话"
		loading={data.deletingChat}
		onCancel={() => (data.deleteTarget = null)}
		onConfirm={() => void data.handleDeleteChat()}
	/>
</div>

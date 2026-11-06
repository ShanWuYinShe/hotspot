<script lang="ts">
	import { goto } from '$app/navigation';
	import { Plus, Pin, BellOff, Trash2, X, MessageSquareDashed } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import Button from '@/components/ui/button.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import OnlineStrip from '../shared/OnlineStrip.svelte';
	import ChatListBody from '../shared/ChatListBody.svelte';
	import NewConversationSheet from '@/components/overlays/NewConversationSheet.svelte';
	import CreateGroupSheet from '@/components/overlays/CreateGroupSheet.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import { ROUTES } from '@/lib/routes';
	import type { ChatListData } from '../shared/useChatListData.svelte';

	let { data }: { data: ChatListData } = $props();
</script>

<div class="flex flex-col flex-1 min-h-full bg-surface-page">
	<PageHeader title="消息" sticky titleSize="2xl">
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
						class="hit-slop-44 h-9 px-3.5"
						aria-label="发起新会话"
						onclick={() => {
							data.newChatTab = 'chat';
							data.showNewChat = true;
						}}
					>
						<Plus class="size-4 text-text-secondary" strokeWidth={1.5} aria-hidden="true" />
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

	<div class="flex flex-1 flex-col pb-6">
		{#if data.onlineChats.length > 0}
			<div class="px-[var(--cst-page-gutter)] pt-2">
				<OnlineStrip
					chats={data.onlineChats}
					onSelect={(c) => void goto(ROUTES.message(c.peerUserId || c.id))}
				/>
			</div>
		{:else if data.loading && !data.needsAuth && !data.loadError}
			<div class="px-[var(--cst-page-gutter)] pt-2" aria-hidden="true">
				<div class="mb-3 flex h-[5.75rem] flex-col overflow-hidden">
					<div class="eyebrow mb-2.5 h-3 w-8 rounded-sm bg-surface-subtle"></div>
					<div class="flex gap-3">
						{#each Array(4) as _, i (i)}
							<div class="flex flex-col items-center gap-1">
								<div class="skeleton-block size-11 rounded-full"></div>
								<div class="h-2.5 w-8 rounded-sm bg-surface-subtle"></div>
							</div>
						{/each}
					</div>
				</div>
			</div>
		{/if}

		{#if !data.needsAuth}
			<div class="px-[var(--cst-page-gutter)] mt-3 mb-1.5">
				<p class="eyebrow">最近</p>
			</div>
		{/if}

		<ChatListBody
			chats={data.filtered}
			loadError={data.loadError}
			needsAuth={data.needsAuth}
			loading={data.loading}
			onRetry={() => data.reload()}
			onSelect={(chat) => void (chat.kind === 'group' ? goto(ROUTES.groupMessage(chat.id)) : goto(ROUTES.message(chat.peerUserId || chat.id)))}
			onMore={(chat) => (data.moreTarget = chat)}
			searchQuery={data.searchQuery}
		/>
	</div>

	<!-- 新会话 Sheet -->
	<Dialog.Root open={data.showNewChat} onOpenChange={(next) => (next ? undefined : (data.showNewChat = false))}>
		<Dialog.Content side="bottom" class="overflow-hidden" style="height:var(--cst-sheet-height-lg);box-shadow:var(--cst-elevation-sheet);">
			<NewConversationSheet
				defaultTab={data.newChatTab}
				onClose={() => (data.showNewChat = false)}
				onNewChat={(id) => void goto(ROUTES.message(id))}
				onCreateGroup={() => (data.showCreateGroup = true)}
			/>
		</Dialog.Content>
	</Dialog.Root>

	<!-- 建群 Sheet -->
	<Dialog.Root open={data.showCreateGroup} onOpenChange={(next) => (next ? undefined : (data.showCreateGroup = false))}>
		<Dialog.Content side="bottom" class="overflow-hidden" style="height:var(--cst-sheet-height-2xl);box-shadow:var(--cst-elevation-sheet);">
			<CreateGroupSheet
				onClose={() => (data.showCreateGroup = false)}
				onCreated={(groupId) => {
					data.showCreateGroup = false;
					void goto(ROUTES.groupMessage(groupId));
				}}
			/>
		</Dialog.Content>
	</Dialog.Root>

	<!-- 移动端会话更多操作浮动卡 -->
	<Dialog.Root open={!!data.moreTarget} onOpenChange={(next) => (next ? undefined : (data.moreTarget = null))}>
		<Dialog.Content side="bottom" floating class="p-5" aria-label={data.moreTarget ? `${data.moreTarget.name} 的会话操作` : '会话操作'}>
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

	<!-- 删除确认 -->
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

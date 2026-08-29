<script lang="ts">
	import { MessageSquareDashed } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ChatListItem from '@/components/messages/ChatListItem.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import type { ChatItem } from '@/lib/chat-list';

	// ChatListBody —— 消息页会话列表体（错误/加载/空态/列表三态），
	// 移动与桌面分栏共用；桌面传 compact + selectedId + onSelect（就地展开），
	// 移动不传 onSelect（onSelect 内自行路由跳转）。
	let {
		chats,
		loadError,
		loading,
		onRetry,
		onSelect,
		onTogglePin,
		selectedId = null,
		compact = false,
		fill = false
	}: {
		chats: ChatItem[];
		loadError: string | null;
		loading: boolean;
		onRetry: () => void;
		onSelect: (chat: ChatItem) => void;
		onTogglePin: (chat: ChatItem) => void;
		selectedId?: number | string | null;
		compact?: boolean;
		fill?: boolean;
	} = $props();
</script>

{#if loadError}
	<div class={fill ? 'flex flex-1 flex-col items-center justify-center px-4 py-10' : ''}>
		<LoadError message={loadError} {onRetry} />
	</div>
{:else if loading}
	<InlineLoading class={fill ? 'flex-1 justify-center' : ''} label="加载会话…" />
{:else if chats.length === 0}
	<!-- 空态覆盖搜索无结果场景，文案统一「没有找到会话」 -->
	<div class={fill ? 'flex flex-1 items-center justify-center' : ''}>
		<EmptyState bare class={fill ? '' : 'py-12!'} title="没有找到会话">
			{#snippet icon()}
				<MessageSquareDashed class="size-9 opacity-muted text-text-secondary" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	</div>
{:else}
	<ListContainer class={fill ? 'flex-1 overflow-y-auto no-scrollbar py-2' : 'py-1'}>
		{#each chats as chat (chat.id)}
			<div class="px-2 py-1">
				<ChatListItem
					chat={chat}
					isSelected={selectedId === chat.id}
					onClick={() => onSelect(chat)}
					onTogglePin={() => onTogglePin(chat)}
					{compact}
				/>
			</div>
		{/each}
	</ListContainer>
{/if}

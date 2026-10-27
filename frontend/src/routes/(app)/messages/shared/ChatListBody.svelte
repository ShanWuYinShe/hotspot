<script lang="ts">
	import { MessageSquareDashed } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import SkeletonList from '@/components/data-display/SkeletonList.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import ChatListItem from '@/components/messages/ChatListItem.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import type { ChatItem } from '@/lib/chat-list';
	import { contentFade } from '@/lib/motion';

	// ChatListBody —— 消息页会话列表体（错误/加载/空态/列表三态），
	// 移动与桌面分栏共用；桌面传 compact + selectedId + onSelect（就地展开），
	// 移动不传 onSelect（onSelect 内自行路由跳转）。
	let {
		chats,
		loadError,
		needsAuth,
		loading,
		onRetry,
		onSelect,
		onTogglePin,
		onMore,
		onToggleMute,
		onDelete,
		selectedId = null,
		compact = false,
		fill = false
	}: {
		chats: ChatItem[];
		loadError: string | null;
		/** 未登录（10100）：显示登录引导而非错误重试 */
		needsAuth: boolean;
		loading: boolean;
		onRetry: () => void;
		onSelect: (chat: ChatItem) => void;
		onTogglePin: (chat: ChatItem) => void;
		onMore?: (chat: ChatItem) => void;
		/** 免打扰切换（spec: update_preferences is_muted）；缺省不渲染入口 */
		onToggleMute?: (chat: ChatItem) => void;
		/** 删除会话（仅私聊有删除端点，群聊由调用方不渲染入口）；缺省不渲染 */
		onDelete?: (chat: ChatItem) => void;
		selectedId?: number | string | null;
		compact?: boolean;
		fill?: boolean;
	} = $props();
</script>

<div class={fill ? 'flex min-h-0 flex-1 flex-col' : ''} use:contentFade={loading}>
{#if needsAuth}
	<NeedsAuthNotice spotlight title="登录后查看" description="登录后即可查看和发起会话" />
{:else if loadError}
	<div class={fill ? 'flex flex-1 flex-col items-center justify-center px-4 py-10' : ''}>
		<LoadError message={loadError} {onRetry} />
	</div>
{:else if loading}
	<!-- 首屏骨架屏：几何对齐鸿蒙 MessagePage 会话骨架（前导圆形 40 / 行内 12 / 文本条 14·55% + 11·35%）；
	     分栏紧凑行与真实 ChatListItem compact 行内距一致，骨架→内容零位移。
	     N2：在线条占位由父页面按真实结构渲染（在线条 → 最近 → 列表），本组件不再自行插入。 -->
	<SkeletonList
		rows={6}
		leading={compact ? 40 : 44}
		label="加载会话…"
		class={fill ? 'flex-1' : 'mt-1'}
		rowClass={compact ? 'px-3.5 py-2.5' : ''}
	/>
{:else if chats.length === 0}
	<!-- 空态覆盖搜索无结果场景，文案统一「没有找到会话」 -->
	<div class={fill ? 'flex flex-1 items-center justify-center' : ''}>
		<EmptyState spotlight bare class={fill ? '' : 'py-12!'} title="没有找到会话">
			{#snippet icon()}
				<MessageSquareDashed class="size-9 opacity-muted text-text-secondary" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	</div>
{:else}
	<!-- 会话列表：无生硬外框，移动与桌面分栏共用；
	     桌面分栏：compact + inset；移动端：通透平铺流式列表。 -->
	<ListContainer
		bordered={false}
		inset={compact}
		class={fill ? 'flex-1 overflow-y-auto no-scrollbar' : compact ? '' : 'mt-1 divide-y divide-border-subtle/30'}
		ariaLabel="会话列表"
	>
		{#each chats as chat (chat.id)}
			<ChatListItem
				chat={chat}
				isSelected={selectedId === chat.id}
				onClick={() => onSelect(chat)}
				onTogglePin={() => onTogglePin(chat)}
				onMore={onMore ? () => onMore(chat) : undefined}
				onToggleMute={onToggleMute ? () => onToggleMute(chat) : undefined}
				onDelete={onDelete ? () => onDelete(chat) : undefined}
				{compact}
			/>
		{/each}
	</ListContainer>
{/if}
</div>

<script lang="ts">
	import { MessageSquareDashed, Plus, MessageCircle, Users, Zap } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import SkeletonList from '@/components/data-display/SkeletonList.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import ChatListItem from '@/components/messages/ChatListItem.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import type { ChatItem } from '@/lib/chat-list';
	import { contentFade } from '@/lib/motion';
	import { rowLongPress } from './row-long-press';
	import Button from '@/components/ui/button.svelte';

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
		onMore,
		searchQuery = '',
		onNewChat,
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
		/** 会话操作菜单（置顶/免打扰/删除）；移动端长按 / 桌面 hover ⋯ 打开 */
		onMore?: (chat: ChatItem) => void;
		/** 当前搜索词：用于区分「还没有会话」与「搜索无结果」两种空态文案。 */
		searchQuery?: string;
		/** 空态里的「发起新会话」入口（动线闭环）；缺省不渲染。 */
		onNewChat?: () => void;
		selectedId?: number | string | null;
		compact?: boolean;
		fill?: boolean;
	} = $props();
</script>

<div class={fill ? 'flex min-h-0 flex-1 flex-col' : ''} use:contentFade={loading}>
{#if needsAuth}
	<NeedsAuthNotice
		spotlight
		title="登录后查看消息"
		description="登录后即可查看和发起会话"
		highlights={[
			{ icon: MessageCircle, text: '私聊与群聊会话' },
			{ icon: Users, text: '在线好友与群组' },
			{ icon: Zap, text: '消息实时同步' }
		]}
	/>
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
	<!-- 空态（R13 新版 EmptyState 图标砖语言）：搜索无结果 / 还没有会话两种语境分流；
	     面板内为小空态，不开 spotlight。 -->
	<div class={fill ? 'flex flex-1 items-center justify-center' : ''}>
		<EmptyState
			class={fill ? 'px-2' : 'py-12!'}
			iconSize={56}
			iconTone={searchQuery ? 'neutral' : 'accent'}
			title={searchQuery ? '没有找到会话' : '还没有会话'}
			description={searchQuery ? '换个关键词试试，或发起一个新会话。' : '和附近的人打个招呼，会话就会出现在这里。'}
		>
			{#snippet icon()}
				<MessageSquareDashed class="size-6" strokeWidth={1.5} aria-hidden="true" />
			{/snippet}
			{#snippet action()}
				{#if onNewChat}
					<Button type="button" size="sm" variant="soft" onclick={onNewChat}>
						<Plus class="size-3.5" strokeWidth={1.75} aria-hidden="true" />
						发起新会话
					</Button>
				{/if}
			{/snippet}
		</EmptyState>
	</div>
{:else}
	<!-- 会话列表：无生硬外框，移动与桌面分栏共用；
	     桌面分栏：compact + inset；移动端：通透平铺流式列表。 -->
	<!-- display:contents 包装只用于挂长按/右键手势：不产生盒子、不插入语义节点，
	     因此 role=list 的直接子节点仍是 listitem（axe aria-required-children 不回归）。 -->
	<div
		class="contents"
		use:rowLongPress={{ count: chats.length, onMore: (idx) => { const chat = chats[idx]; if (chat) onMore?.(chat); } }}
	>
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
					onMore={onMore ? () => onMore(chat) : undefined}
					{compact}
				/>
			{/each}
		</ListContainer>
	</div>
{/if}
</div>

<script lang="ts">
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import InteractiveListRow from '@/components/navigation/InteractiveListRow.svelte';

	// 会话列表项（对齐 Figma Make ConversationListItem 契约）：
	//   - 基于 InteractiveListRow 统一行组件（hover 填充 / tap 反馈 / 键盘 / aria）
	//   - 已读/未读不靠字号或字重区分（所有 item 大小一致），未读仅由
	//     计数 chip 呈现；点击后未读清零（由页面 markRead 处理）
	//   - selected：InteractiveListRow pressed 驱动 accent 全背景 + 反色文字
	//     （content-inverse 系列）+ border-inverse 头像框；inline 背景避免
	//     全局 button hover overlay 盖出白蒙蒙效果
	//   - unread 徽章：选中时保留占位并改为反色，避免行内 flex-1 文本区
	//     宽度跳变导致点击位移

	export interface ChatItem {
		id: string;
		peerUserId: string;
		name: string;
		avatar: string;
		lastMessage: string;
		time: string;
		unread: number;
		online: boolean;
		group: boolean;
		kind?: 'private' | 'group';
	}

	let {
		chat,
		isSelected = false,
		onClick,
		compact = false
	}: {
		chat: ChatItem;
		isSelected?: boolean;
		onClick: () => void;
		compact?: boolean;
	} = $props();
</script>

<InteractiveListRow
	pressed={isSelected}
	onClick={onClick}
	ariaLabel={`打开与 ${chat.name} 的会话`}
	class="flex items-center gap-3.5 rounded-control"
	style="padding:{compact ? '10px 16px' : '12px 16px'};box-shadow:{isSelected ? '0 3px 12px var(--glow-accent)' : 'none'};"
>
	<span class="relative shrink-0">
		<span
			class="block overflow-hidden rounded-full"
			style="width:{compact ? 40 : 44}px;height:{compact ? 40 : 44}px;border:{isSelected ? '1.5px solid var(--border-inverse)' : '1.5px solid var(--border-default)'};"
		>
			<img src={chat.avatar} alt="" class="h-full w-full object-cover" />
		</span>
		{#if chat.online}
			<OnlineDot size="md" />
		{/if}
	</span>

	<span class="min-w-0 flex-1">
		<span class="mb-0.5 flex items-baseline justify-between">
			<span class="{isSelected ? 'text-content-inverse' : 'text-text-primary'} truncate text-sm">
				{chat.name}
			</span>
			<span class="{isSelected ? 'text-content-inverse-muted' : 'text-text-secondary'} shrink-0 text-xs">
				{chat.time}
			</span>
		</span>
		<span class="block truncate pr-4 {isSelected ? 'text-content-inverse-secondary' : 'text-text-secondary'} text-xs">
			{chat.lastMessage}
		</span>
	</span>

	{#if chat.unread > 0}
		<span
			class="flex shrink-0 items-center justify-center rounded-full text-xs font-semibold {isSelected ? 'bg-content-inverse text-accent-default' : 'bg-accent-default text-white'}"
			style="width:20px;height:20px;{isSelected ? '' : 'box-shadow:0 2px 8px var(--glow-accent);'}"
		>
			{chat.unread}
		</span>
	{/if}
</InteractiveListRow>

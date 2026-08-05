<script lang="ts">
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';

	// 会话列表项（对齐 Figma Make ConversationListItem 契约）：
	//   - 已读/未读不靠字号或字重区分（所有 item 大小一致），未读仅由
	//     计数 chip 呈现；点击后未读清零（由页面 markRead 处理）
	//   - hover/focus-visible：--interactive-hover-fill 背景填充
	//   - selected：accent 背景 + glow 阴影 + 反色文字（content-inverse 系列）
	//     + border-inverse 头像框。选中态用 inline background（非 bg-* 类），
	//     避免全局 button hover overlay 盖出白蒙蒙效果；aria-current 触发
	//     utilities 豁免规则（hover 时保持 accent 不变）
	//   - unread 徽章：选中时保留占位并改为反色，避免行内 flex-1 文本区
	//     宽度跳变导致点击位移
	//   - tap：按压 opacity 0.8 反馈；transition 150ms

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

<button
	type="button"
	aria-current={isSelected ? 'page' : undefined}
	class="group flex w-full cursor-pointer items-center gap-3.5 rounded-control px-4 text-left outline-none transition-[background-color,opacity] duration-150 {isSelected ? '' : 'hover:bg-interactive-hover-fill focus-visible:bg-interactive-hover-fill'} active:opacity-80"
	style="padding:{compact ? '10px 16px' : '12px 16px'};background:{isSelected ? 'var(--accent-default)' : 'transparent'};box-shadow:{isSelected ? '0 3px 12px var(--glow-accent)' : 'none'};"
	onclick={onClick}
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
</button>

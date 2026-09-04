<script lang="ts">
	import { Pin } from '@lucide/svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import type { ChatItem } from '@/lib/chat-list';

	// 会话列表项（对齐 Figma Make ConversationListItem 契约）：
	//   - 基于公共 ListItem 行组件（hover 填充 / tap 反馈 / 键盘 / aria）
	//   - 已读/未读不靠字号或字重区分（所有 item 大小一致），未读仅由
	//     计数 chip 呈现；点击后未读清零（由页面 markRead 处理）
	//   - selected：ListItem pressed 驱动 accent 全背景 + on-accent 文字
	//     （text-text-on-accent 系列，随明暗模式切换深浅）+ border-inverse
	//     头像框；inline 背景避免全局 button hover overlay 盖出白蒙蒙效果
	//   - unread 徽章：选中时保留占位并改为反色，避免行内 flex-1 文本区
	//     宽度跳变导致点击位移
	//   - 置顶：isPinned 高亮 Pin 图标；onTogglePin 可选（不传则不渲染），
	//     点击时 stopPropagation，避免触发行选中/导航

	let {
		chat,
		isSelected = false,
		onClick,
		onTogglePin,
		compact = false
	}: {
		chat: ChatItem;
		isSelected?: boolean;
		onClick: () => void;
		onTogglePin?: () => void;
		compact?: boolean;
	} = $props();
</script>

<ListItem
	pressed={isSelected}
	onClick={onClick}
	ariaLabel={`打开与 ${chat.name} 的会话`}
	class="flex items-center gap-3.5 {compact ? 'px-4 py-2.5' : 'px-4 py-3'}"
>
	<span class="relative shrink-0">
		<Avatar
			src={chat.avatar}
			name={chat.name}
			size={compact ? 40 : 44}
			class="border"
			style="border-color:{isSelected ? 'var(--cst-border-inverse)' : 'var(--cst-border-default)'};border-width:var(--cst-border-width-regular);"
		/>
		{#if chat.online}
			<OnlineDot size="md" />
		{/if}
	</span>

	<span class="min-w-0 flex-1">
		<span class="mb-0.5 flex items-baseline justify-between">
			<span class="{isSelected ? 'text-text-on-accent' : 'text-text-primary'} truncate text-sm">
				{chat.name}
			</span>
			<span class="{isSelected ? 'text-text-on-accent opacity-soft' : 'text-text-secondary'} shrink-0 text-xs">
				{chat.time}
			</span>
		</span>
		<span class="block truncate pr-4 {isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} text-xs">
			{chat.lastMessage}
		</span>
	</span>

	{#if onTogglePin}
		<button
			type="button"
			class="hover-btn scale-btn hit-slop focus-ring flex shrink-0 items-center justify-center rounded-full p-1 {isSelected || chat.isPinned ? '' : 'text-text-secondary opacity-muted hover:text-accent-default hover:opacity-full focus-visible:opacity-full'}"
			aria-label={chat.isPinned ? '取消置顶' : '置顶'}
			title={chat.isPinned ? '取消置顶' : '置顶'}
			onclick={(e) => {
				e.stopPropagation();
				onTogglePin();
			}}
		>
			<Pin
				class="size-3.5"
				strokeWidth={2}
				style="color:{isSelected ? 'var(--cst-text-on-accent)' : chat.isPinned ? 'var(--cst-accent-default)' : 'var(--cst-text-secondary)'};"
			/>
		</button>
	{/if}

	{#if chat.unread > 0}
		<span
			class="flex shrink-0 items-center justify-center rounded-full text-xs font-semibold size-5 {isSelected ? 'bg-text-on-accent text-accent-default' : 'bg-accent-default text-text-on-accent'}"
			style="{isSelected ? '' : 'box-shadow:var(--shadow-glow-sm);'}"
		>
			{chat.unread}
		</span>
	{/if}
</ListItem>

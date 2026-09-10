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
	class="group flex items-center gap-3.5 {compact ? 'px-3.5 py-2.5' : 'px-4 py-3'} {chat.isPinned && !isSelected ? 'bg-surface-subtle' : ''}"
>
	<span class="relative shrink-0">
		<Avatar
			src={chat.avatar}
			name={chat.name}
			size={compact ? 40 : 44}
			style={isSelected ? 'box-shadow:0 0 0 var(--cst-border-width-regular) var(--cst-border-inverse);' : ''}
		/>
		{#if chat.online}
			<OnlineDot size="md" />
		{/if}
	</span>

	<span class="min-w-0 flex-1 flex flex-col justify-center">
		<span class="flex items-center justify-between gap-2 mb-0.5">
			<span class="{isSelected ? 'text-text-on-accent' : 'text-text-primary font-medium'} truncate text-sm">
				{chat.name}
			</span>
			<span class="{isSelected ? 'text-text-on-accent opacity-soft' : 'text-text-secondary'} shrink-0 text-xs font-normal tabular-nums">
				{chat.time}
			</span>
		</span>
		<span class="flex items-center justify-between gap-2">
			<span class="truncate pr-1 {isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} text-xs">
				{chat.lastMessage || '暂无消息'}
			</span>
			<span class="flex items-center gap-1.5 shrink-0">
				{#if chat.isPinned}
					{#if onTogglePin}
						<button
							type="button"
							class="hover-btn scale-btn hit-slop focus-ring flex shrink-0 items-center justify-center rounded-full p-1 {isSelected ? 'text-text-on-accent' : 'text-accent-default'}"
							aria-label="取消置顶"
							title="取消置顶"
							onclick={(e) => {
								e.stopPropagation();
								onTogglePin();
							}}
						>
							<Pin
								class="size-3.5 {isSelected ? 'text-text-on-accent' : 'text-accent-default'}"
								strokeWidth={2}
							/>
						</button>
					{:else}
						<Pin
							class="size-3.5 {isSelected ? 'text-text-on-accent' : 'text-accent-default'}"
							strokeWidth={2}
						/>
					{/if}
				{:else if onTogglePin}
					<!-- 未置顶时：默认常态完全隐藏！仅在桌面端鼠标 hover 或键盘 focus 时浮现快捷置顶 -->
					<button
						type="button"
						class="hover-btn scale-btn hit-slop focus-ring shrink-0 items-center justify-center rounded-full p-1 opacity-0 group-hover:opacity-70 hover:!opacity-100 focus-visible:opacity-100 transition-opacity duration-instant hidden md:flex text-text-secondary"
						aria-label="置顶"
						title="置顶"
						onclick={(e) => {
							e.stopPropagation();
							onTogglePin();
						}}
					>
						<Pin
							class="size-3.5 {isSelected ? 'text-text-on-accent' : 'text-text-secondary'}"
							strokeWidth={2}
						/>
					</button>
				{/if}
				{#if chat.unread > 0}
					<span
						class="inline-flex min-w-5 h-5 items-center justify-center rounded-full px-1.5 text-xs font-semibold leading-none tabular-nums {isSelected ? 'bg-text-on-accent text-accent-default' : 'bg-accent-default text-text-on-accent'}"
					>
						{chat.unread}
					</span>
				{/if}
			</span>
		</span>
	</span>
</ListItem>

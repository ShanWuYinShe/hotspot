<script lang="ts">
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';

	// 会话列表项（对齐 Figma Make ConversationListItem 契约）：
	//   - hover/focus-visible：--interactive-hover-fill 背景填充；选中态通过
	//     类切换渲染 accent 背景，故 hover 填充不会覆盖选中态（inline style
	//     会压过 Tailwind hover 类导致 hover 不生效，因此不用 inline 背景）
	//   - selected：accent 背景 + glow 阴影 + 反色文字（content-inverse 系列）
	//     + border-inverse 头像框；仅 unread > 0 时名字加粗（对齐 Figma）
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
	class="group flex w-full cursor-pointer items-center gap-3.5 rounded-control px-4 text-left outline-none transition-colors duration-150 {isSelected ? 'bg-[var(--accent-default)]' : 'hover:bg-[var(--interactive-hover-fill)] focus-visible:bg-[var(--interactive-hover-fill)]'} active:opacity-80"
	style="padding:{compact ? '10px 16px' : '12px 16px'};box-shadow:{isSelected ? '0 3px 12px var(--glow-accent)' : 'none'};"
	onclick={onClick}
>
	<span class="relative flex-shrink-0">
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
			<span
				class="{isSelected ? 'text-[var(--content-inverse)]' : 'text-[var(--text-primary)]'} text-sm"
				style="font-weight:{chat.unread > 0 ? 600 : 400};"
			>
				{chat.name}
			</span>
			<span class="{isSelected ? 'text-[var(--content-inverse-muted)]' : 'text-[var(--text-secondary)]'} shrink-0 text-[11px]">
				{chat.time}
			</span>
		</span>
		<span class="block truncate pr-4 {isSelected ? 'text-[var(--content-inverse-secondary)]' : 'text-[var(--text-secondary)]'} text-xs">
			{chat.lastMessage}
		</span>
	</span>

	{#if chat.unread > 0 && !isSelected}
		<span
			class="flex flex-shrink-0 items-center justify-center rounded-full bg-[var(--accent-default)] text-white text-[11px] font-semibold"
			style="width:20px;height:20px;box-shadow:0 2px 8px var(--glow-accent);"
		>
			{chat.unread}
		</span>
	{/if}
</button>

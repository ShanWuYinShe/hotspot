<script lang="ts">
	import OnlineDot from '@/components/shared/OnlineDot.svelte';

	// 会话列表项（迁移自旧版 MessagesPage 内 ChatListItem）。

	export interface ChatItem {
		id: number;
		peerUserId: string;
		name: string;
		avatar: string;
		lastMessage: string;
		time: string;
		unread: number;
		online: boolean;
		group: boolean;
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
	data-active={isSelected}
	aria-current={isSelected ? 'page' : undefined}
	class="neu-hover-row app-list-row app-list-row--rich w-full cursor-pointer text-left outline-none neu-row-hover"
	style="padding:{compact ? '10px 16px' : '12px 16px'};border-radius:18px;"
	onclick={onClick}
>
	<span class="relative flex-shrink-0">
		<span
			class="block overflow-hidden rounded-full"
			style="width:{compact ? 40 : 44}px;height:{compact ? 40 : 44}px;border:{isSelected ? '1.5px solid var(--neu-accent-primary)' : '1.5px solid var(--neu-border)'};"
		>
			<img src={chat.avatar} alt="" class="h-full w-full object-cover" />
		</span>
		{#if chat.online}
			<OnlineDot size="md" />
		{/if}
	</span>

	<span class="min-w-0 flex-1">
		<span class="mb-0.5 flex items-baseline justify-between">
			<span class="{isSelected ? 'text-[var(--neu-accent-primary)]' : 'text-[var(--neu-text)]'} text-sm" style="font-weight:{chat.unread > 0 || isSelected ? 600 : 400};">
				{chat.name}
			</span>
			<span class="text-[var(--neu-text-muted)] text-[11px] shrink-0">
				{chat.time}
			</span>
		</span>
		<span class="block truncate pr-4 text-[var(--neu-text-muted)] text-xs">
			{chat.lastMessage}
		</span>
	</span>

	{#if chat.unread > 0 && !isSelected}
		<span
			class="flex flex-shrink-0 items-center justify-center rounded-full bg-[var(--neu-accent-primary)] text-white text-[11px] font-semibold"
			style="width:20px;height:20px;box-shadow:0 2px 8px var(--neu-glow);"
		>
			{chat.unread}
		</span>
	{/if}
</button>

<script lang="ts">
	import Avatar from '@/components/shared/Avatar.svelte';
	import { scrollFade } from '@/lib/scroll-fade';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import type { ChatItem } from '@/lib/chat-list';

	// OnlineStrip —— 消息页「在线」头像横条（移动/桌面分栏共用）。
	// variant=lg：移动端大头像 + 在线大点；sm：桌面分栏紧凑头像。
	let {
		chats,
		variant = 'lg',
		selectedId = null,
		onSelect
	}: {
		chats: ChatItem[];
		variant?: 'lg' | 'sm';
		selectedId?: number | string | null;
		onSelect: (chat: ChatItem) => void;
	} = $props();

	const isLarge = $derived(variant === 'lg');
</script>

<div class="{isLarge ? 'mb-3' : 'shrink-0 px-4 py-2.5'}">
	<!-- 分组小标统一 eyebrow（tracking-eyebrow 0.14em），与全站一致 -->
	<p class="eyebrow {isLarge ? 'mb-2.5' : 'mb-2'}">在线</p>
	<div class="flex overflow-x-auto no-scrollbar {isLarge ? 'gap-3 pb-1 pr-1' : 'gap-3 pr-1'}" use:scrollFade>
		{#each chats as c (c.id)}
			<button
				type="button"
				class="avatar-entry tap-92 flex flex-col items-center shrink-0 cursor-pointer {isLarge ? 'gap-1' : 'gap-1'}"
				onclick={() => onSelect(c)}
			>
				<div class="relative">
					<Avatar
						src={c.avatar}
						name={c.name}
						size={isLarge ? 44 : 36}
						decorative
						class="border"
						style={isLarge
							? 'border-color:var(--cst-accent-default);border-width:var(--cst-border-width-regular);'
							: `border-color:${selectedId === c.id ? 'var(--cst-accent-default)' : 'var(--cst-border-default)'};border-width:${selectedId === c.id ? 'var(--cst-border-width-strong)' : 'var(--cst-border-width-regular)'};`}
					/>
					<OnlineDot size={isLarge ? 'lg' : 'sm'} />
				</div>
				<span class="text-text-secondary {isLarge ? 'text-xs' : 'text-xxs'}">{c.name}</span>
			</button>
		{/each}
	</div>
</div>

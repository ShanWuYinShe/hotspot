<script lang="ts">
	import Avatar from '@/components/shared/Avatar.svelte';
	import { scrollFade } from '@/lib/scroll-fade';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import type { ChatItem } from '@/lib/chat-list';

	// OnlineStrip —— 消息页「在线」头像横条（移动/桌面分栏共用）。
	// variant=lg：移动端大头像 + 在线大点；sm：桌面分栏紧凑头像。
	// R13：
	//   - 分组标统一「在线 + 人数」两级层级（人数 11px tabular 次级色），
	//     与「最近」列表分开，但不再靠一整行空白分隔；
	//   - 在线头像统一走 accent 细环（在线是语义而不是装饰），选中会话再加一档
	//     强描边 + 轻投影，两态一眼可辨且不依赖颜色深浅；
	//   - 每个头像按钮补 aria-label（触屏与读屏都是唯一入口）。
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

<div class={isLarge ? 'mb-3' : 'shrink-0 px-4 py-2.5'}>
	<!-- 分组小标统一 eyebrow（tracking-eyebrow 0.14em），右侧跟在线人数 -->
	<div class="flex items-baseline gap-2 {isLarge ? 'mb-2.5' : 'mb-2'}">
		<p class="eyebrow">在线</p>
		<span class="text-xxs tabular-nums text-text-secondary">{chats.length}</span>
	</div>
	<div class="flex overflow-x-auto no-scrollbar {isLarge ? 'gap-3 pb-1 pr-1' : 'gap-3 pr-1'}" use:scrollFade>
		{#each chats as c (c.id)}
			<button
				type="button"
				class="avatar-entry tap-92 flex shrink-0 cursor-pointer flex-col items-center gap-1 focus-ring rounded-sm"
				aria-label={`打开与 ${c.name} 的会话`}
				onclick={() => onSelect(c)}
			>
				<div class="relative">
					<Avatar
						src={c.avatar}
						name={c.name}
						size={isLarge ? 44 : 36}
						decorative
						class="border"
						style={selectedId === c.id
							? 'border-color:var(--cst-accent-default);border-width:var(--cst-border-width-strong);box-shadow:var(--shadow-raised-sm);'
							: 'border-color:var(--cst-accent-border);border-width:var(--cst-border-width-regular);'}
					/>
					<OnlineDot size={isLarge ? 'lg' : 'sm'} />
				</div>
				<span class="max-w-13 truncate text-text-secondary {isLarge ? 'text-xs' : 'text-xxs'}">{c.name}</span>
			</button>
		{/each}
	</div>
</div>

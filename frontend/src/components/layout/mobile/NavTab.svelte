<script lang="ts">
	import type { Component } from 'svelte';
	import UnreadBadge from '@/components/shared/UnreadBadge.svelte';

	// 移动/平板底部导航项（对齐 Figma Make NavPill 契约）：
	// - flex-1 均分可用宽度，主导航不设固定 item 宽度
	// - 选中胶囊：inset 4px 6px + shape-panel 圆角 + navigation-active-indicator
	// - 图标/文字色走 navigation-content(-active) 语义 token
	// - count：可选未读数气泡，悬挂图标右上角（如「消息」Tab）

	let {
		label,
		isActive,
		onClick,
		Icon,
		iconSize = 22,
		count = 0
	}: {
		label: string;
		isActive: boolean;
		onClick: () => void;
		Icon: Component;
		/** 图标尺寸 px（移动 22 / 平板 24） */
		iconSize?: number;
		count?: number;
	} = $props();
</script>

<button
	type="button"
	class="hover-btn nav-tab relative flex flex-col items-center justify-center rounded-xl py-2 gap-1 focus-ring select-none w-full min-w-0 transition-transform active:scale-95 duration-rapid
		{isActive ? '' : 'hover:bg-interactive-hover'}"
	aria-label={label}
	aria-current={isActive ? 'page' : undefined}
	onclick={onClick}
>
	<div class="flex items-center justify-center relative transition-transform duration-standard {isActive ? 'scale-105' : 'scale-100'}">
		<Icon
			style="width:{iconSize}px;height:{iconSize}px;color:{isActive ? 'var(--cst-navigation-content-active)' : 'var(--cst-navigation-content)'};transition:color var(--cst-motion-standard) var(--cst-ease-standard);"
			strokeWidth={isActive ? 2.25 : 1.5}
		/>
		{#if count > 0}
			<span class="absolute -right-2.5 -top-1.5">
				<UnreadBadge {count} />
			</span>
		{/if}
	</div>
	<span
		class="text-xs leading-none relative {isActive ? 'font-semibold' : 'font-normal'}"
		style="color:{isActive ? 'var(--cst-navigation-content-active)' : 'var(--cst-navigation-content)'};transition:color var(--cst-motion-standard) var(--cst-ease-standard);"
	>
		{label}
	</span>
</button>

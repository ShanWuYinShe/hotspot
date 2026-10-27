<script lang="ts">
	import type { Component } from 'svelte';
	import UnreadBadge from '@/components/shared/UnreadBadge.svelte';

	// 移动/平板底部导航项（对齐 Figma Make NavPill 契约）：
	// - flex-1 均分可用宽度，主导航不设固定 item 宽度
	// - 选中态：图标 + 文字收进 accent-soft 胶囊（形状 + semibold 双重信号，不依赖颜色），
	//   未选中只保留图标/文字等级差，hover 给中性淡底预览——不再出现「只有图标变蓝」的弱选中态
	// - 图标/文字色走 navigation-content(-active) 语义 token
	// - count：可选未读数气泡，统一悬挂图标右上角（与横屏 RailIcon 同一口径）

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
	class="nav-tab group relative flex w-full min-w-0 flex-col items-center justify-center py-1 focus-ring select-none transition-transform active:scale-95 duration-rapid"
	aria-label={label}
	aria-current={isActive ? 'page' : undefined}
	onclick={onClick}
>
	<!-- 选中胶囊包住图标+文字：命中区仍是整格按钮（触控 ≥44），视觉重心收在内容上 -->
	<span
		class="flex flex-col items-center justify-center gap-1 rounded-xl px-3 py-1.5 transition-colors duration-standard {isActive
			? 'bg-accent-soft'
			: 'group-hover:bg-interactive-hover'}"
	>
		<span
			class="relative flex items-center justify-center transition-transform duration-standard {isActive
				? 'scale-105'
				: 'scale-100'}"
		>
			<Icon
				style="width:{iconSize}px;height:{iconSize}px;color:{isActive ? 'var(--cst-navigation-content-active)' : 'var(--cst-navigation-content)'};transition:color var(--cst-motion-standard) var(--cst-ease-standard);"
				strokeWidth={isActive ? 2.25 : 1.5}
			/>
			{#if count > 0}
				<span class="absolute -right-2 -top-1.5">
					<UnreadBadge {count} />
				</span>
			{/if}
		</span>
		<span
			class="text-xs leading-none {isActive ? 'font-semibold' : 'font-normal'}"
			style="color:{isActive ? 'var(--cst-navigation-content-active)' : 'var(--cst-navigation-content)'};transition:color var(--cst-motion-standard) var(--cst-ease-standard);"
		>
			{label}
		</span>
	</span>
</button>

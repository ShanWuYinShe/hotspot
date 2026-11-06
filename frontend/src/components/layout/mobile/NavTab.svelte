<script lang="ts">
	import type { Component } from 'svelte';
	import { cn } from '@/lib/utils';
	import UnreadBadge from '@/components/shared/UnreadBadge.svelte';

	// 移动/平板底部悬浮孤岛导航项（Floating Dock Tab）：
	// - flex-1 均分可用宽度，触控热区充实稳健（触控高度 ≥44px）
	// - 纯净优雅的 Content-First 选中态：
	//   * 废除沉重方正的背景方块/圆圈贴片，保持悬浮 Dock 通透呼吸感
	//   * 选中项图标与文字统一为高凝聚力品牌主色 text-accent-default
	//   * 图标 scale-110 微放大 + strokeWidth 2.25 笔画加粗，提供充实视觉锚点
	//   * 底部精致水平微胶囊指示条（h-0.75 w-3.5 rounded-full bg-accent-default），切换时平滑淡入展开
	// - count：可选未读数气泡，悬挂图标右上角

	let {
		label,
		isActive,
		onClick,
		Icon,
		iconSize = 25,
		count = 0
	}: {
		label: string;
		isActive: boolean;
		onClick: () => void;
		Icon: Component;
		/** 图标尺寸 px（移动/平板统一基准 25px，充实饱满） */
		iconSize?: number;
		count?: number;
	} = $props();
</script>

<button
	type="button"
	class="nav-tab group relative flex flex-1 min-w-0 items-center justify-center py-1 select-none focus-ring transition-transform active:scale-95 duration-rapid"
	aria-label={label}
	aria-current={isActive ? 'page' : undefined}
	onclick={onClick}
>
	<div
		class="relative flex flex-col items-center justify-center gap-1 rounded-xl px-2 sm:px-3 py-1 transition-colors duration-standard hover:bg-surface-primary/40"
	>
		<!-- 图标区 -->
		<span class="relative flex items-center justify-center">
			<Icon
				style="width:{iconSize / 16}rem;height:{iconSize / 16}rem;"
				class={cn(
					'transition-all duration-snappy',
					isActive
						? 'scale-110 -translate-y-0.5 text-accent-default'
						: 'scale-100 text-text-secondary group-hover:text-text-primary'
				)}
				strokeWidth={isActive ? 2.25 : 1.75}
			/>
			{#if count > 0}
				<span class="absolute -right-2.5 -top-1">
					<UnreadBadge {count} />
				</span>
			{/if}
		</span>

		<!-- 文字区 -->
		<span
			class={cn(
				'text-xs leading-none tracking-tight transition-colors duration-standard',
				isActive
					? 'font-semibold text-accent-default'
					: 'font-medium text-text-secondary group-hover:text-text-primary'
			)}
		>
			{label}
		</span>

		<!-- 底部精致微胶囊指示条（占位防抖动，选中时平滑呼吸展开） -->
		<span
			data-nav-indicator
			class={cn(
				'h-0.75 w-3.5 rounded-full bg-accent-default shadow-[0_1px_4px_rgba(var(--cst-palette-accent-primary),0.35)] transition-all duration-snappy',
				isActive ? 'opacity-100 scale-100' : 'opacity-0 scale-50 pointer-events-none'
			)}
			aria-hidden="true"
		></span>
	</div>
</button>

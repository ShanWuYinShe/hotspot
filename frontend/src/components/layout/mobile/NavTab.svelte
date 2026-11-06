<script lang="ts">
	import type { Component } from 'svelte';
	import { cn } from '@/lib/utils';
	import UnreadBadge from '@/components/shared/UnreadBadge.svelte';

	// 移动/平板底部悬浮孤岛导航项（Floating Dock Tab）：
	// - flex-1 均分可用宽度，触控热区充实稳健
	// - 选中态：图标 + 文字收进温润有厚度的 accent-soft 软质触觉胶囊（形状 + semibold 双重信号），
	//   提供充实扎实的踏实感（不依赖单薄线条或纯色块），高质感微投影与微内嵌光泽
	// - 图标尺寸 25px，未选 1.85 笔画 / 选中 2.25 笔画，杜绝单薄空洞感
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
		class={cn(
			'relative flex flex-col items-center justify-center gap-0.5 rounded-xl px-2.5 sm:px-3 py-1 transition-all duration-standard',
			isActive
				? 'bg-accent-soft text-text-accent-on-soft shadow-2xs ring-1 ring-accent-border/30'
				: 'text-text-secondary hover:text-text-primary hover:bg-surface-primary/40'
		)}
	>
		<span class="relative flex items-center justify-center">
			<Icon
				style="width:{iconSize / 16}rem;height:{iconSize / 16}rem;"
				class={cn(
					'transition-transform duration-standard',
					isActive ? 'scale-105 text-accent-default' : 'text-text-secondary group-hover:text-text-primary'
				)}
				strokeWidth={isActive ? 2.25 : 1.85}
			/>
			{#if count > 0}
				<span class="absolute -right-2.5 -top-1">
					<UnreadBadge {count} />
				</span>
			{/if}
		</span>
		<span
			class={cn(
				'text-xs leading-tight tracking-tight transition-colors duration-standard',
				isActive
					? 'font-semibold text-text-accent-on-soft'
					: 'font-medium text-text-secondary group-hover:text-text-primary'
			)}
		>
			{label}
		</span>
	</div>
</button>

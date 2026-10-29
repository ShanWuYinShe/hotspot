<script lang="ts">
	import type { Component } from 'svelte';
	import { ChevronRight } from '@lucide/svelte';
	import Badge from '@/components/data-display/Badge.svelte';

	// 管理后台模块入口卡（/admin 概览私有组件）。
	// 语言：图标砖 + 标题 + 一句说明 + 待办角标 + 进入箭头；整卡可点（button 本体承载语义，
	// 不是嵌套交互），hover 抬升、focus-ring 可见；无权限时禁用并明示「无访问权限」。
	// 栅格由调用方决定（桌面 3 列 / 平板 2 列 / 移动单列），本组件只保证单卡自洽。

	let {
		label,
		description,
		icon: Icon,
		allowed = true,
		badge,
		onOpen
	}: {
		label: string;
		description: string;
		icon: Component;
		allowed?: boolean;
		/** 待办角标文案（如「待处理 2」）；为空不渲染。 */
		badge?: string;
		onOpen: () => void;
	} = $props();
</script>

<button
	type="button"
	disabled={!allowed}
	title={allowed ? label : '无对应权限'}
	onclick={onOpen}
	class="group flex w-full items-center gap-3 rounded-lg bg-surface-island p-4 text-left shadow-xs transition-[box-shadow,transform] duration-standard ease-standard hover:-translate-y-0.5 hover:shadow-raised-md active:scale-[0.99] focus-ring disabled:cursor-not-allowed disabled:pointer-events-none disabled:opacity-faint"
>
	<span
		class="flex size-10 shrink-0 items-center justify-center rounded-md bg-accent-soft text-text-accent-on-soft"
		aria-hidden="true"
	>
		<Icon class="size-5" strokeWidth={1.75} />
	</span>
	<span class="min-w-0 flex-1">
		<span class="flex flex-wrap items-center gap-x-2 gap-y-1">
			<span class="truncate text-md font-semibold text-text-primary">{label}</span>
			{#if badge}
				<Badge tone="warning">{badge}</Badge>
			{/if}
		</span>
		<span class="mt-1 block text-xs leading-card text-text-secondary">
			{allowed ? description : '无访问权限'}
		</span>
	</span>
	<ChevronRight
		class="size-4 shrink-0 text-text-secondary opacity-subtle transition-transform duration-standard group-hover:translate-x-0.5"
		strokeWidth={1.5}
		aria-hidden="true"
	/>
</button>

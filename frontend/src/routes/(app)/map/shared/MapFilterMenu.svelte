<script lang="ts">
	import { SlidersHorizontal, Check } from '@lucide/svelte';
	import * as Popover from '@/components/ui/popover';
	import { PIN_TONES, markerSvg } from './location-shared';
	import type { PinType } from './location-data';

	// MapFilterMenu — 地图筛选菜单（桌面/移动共用）。
	// 修复此前两套复制粘贴实现的交互短板，统一为一份：
	//   - bits-ui Popover 承载：点击面板外任意处（含左侧栏/搜索框）关闭、Escape 关闭、
	//     Portal 定位自动避让边缘、入场 fade+zoom 动画
	//   - 触发按钮带选中数徽章（非全选时）与展开态 accent 实色
	//   - 至少保留一项：取消最后一项时该行禁用，避免筛到空地图
	//   - 「全部显示」一键重置
	//   - 桌面端额外显示每类计数与"我的位置"图例（counts/showLegend）
	let {
		activeFilters = $bindable(new Set<PinType>(['user', 'group', 'moment'])),
		counts,
		showLegend = false,
		align = 'end'
	}: {
		activeFilters?: Set<PinType>;
		/** 视口内各类型数量（隐藏类型传 0/省略时显示 —） */
		counts?: Record<PinType, number>;
		showLegend?: boolean;
		align?: 'start' | 'center' | 'end';
	} = $props();

	const ALL: Array<{ type: PinType; label: string }> = [
		{ type: 'user', label: '用户' },
		{ type: 'group', label: '群组' },
		{ type: 'moment', label: '瞬间' }
	];

	let open = $state(false);
	let anchorEl = $state<HTMLElement | null>(null);

	const allActive = $derived(activeFilters.size === ALL.length);

	function toggleFilter(type: PinType) {
		// 至少保留一项：全部取消会让地图清空且无引导
		if (activeFilters.size === 1 && activeFilters.has(type)) return;
		const next = new Set(activeFilters);
		next.has(type) ? next.delete(type) : next.add(type);
		activeFilters = next;
	}

	function showAll() {
		activeFilters = new Set(ALL.map((i) => i.type));
	}
</script>

<Popover.Root bind:open>
	<div bind:this={anchorEl}>
		<button
			type="button"
			class="hover-btn scale-btn focus-ring flex items-center gap-2 rounded-full border px-4 py-2 text-sm
				{open
				? 'border-transparent bg-accent-default text-text-on-accent shadow-glow-md'
				: 'border-border-default bg-surface-primary text-text-primary shadow-[var(--cst-elevation-fab)]'}"
			aria-label="筛选地图显示类型"
			aria-expanded={open}
			onclick={() => (open = !open)}
		>
			<SlidersHorizontal class="size-3.5" strokeWidth={open ? 2.5 : 1.75} />
			筛选
			{#if !allActive}
				<span class="text-xxs font-medium rounded-full" style="margin-left:2px;padding:1px 6px;background:{open ? 'var(--cst-surface-map-control-active)' : 'var(--cst-accent-soft)'};color:{open ? 'var(--cst-text-on-accent)' : 'var(--cst-accent-default)'};">{activeFilters.size}</span>
			{/if}
		</button>
	</div>

	<Popover.Content {align} side="bottom" sideOffset={8} customAnchor={anchorEl} class="w-64 overflow-hidden rounded-xl p-0">
		<div class="flex items-center justify-between border-b border-border-default px-3.5 py-2.5">
			<span class="text-sm font-medium text-text-primary">显示类型</span>
			<span class="text-xxs text-text-secondary">{activeFilters.size}/3 已选</span>
		</div>
		<div class="flex flex-col gap-1 p-1.5">
			{#each ALL as item (item.type)}
				{@const active = activeFilters.has(item.type)}
				{@const count = counts?.[item.type]}
				<button
					type="button"
					class="flex w-full items-center gap-3 rounded-lg px-2.5 py-2 text-left focus-ring transition-colors {active ? 'bg-accent-soft' : 'hover:bg-interactive-hover'} {active ? '' : 'opacity-soft'}"
					aria-pressed={active}
					onclick={() => toggleFilter(item.type)}
				>
					<span
						class="inline-flex size-8 shrink-0 items-center justify-center rounded-lg"
						style="background:{active ? PIN_TONES[item.type].bg : 'var(--cst-surface-primary)'};border:var(--cst-border-width-hairline) solid {active ? PIN_TONES[item.type].border : 'transparent'};"
					>
						{@html markerSvg(item.type, 1, { w: 14, h: 19 })}
					</span>
					<span class="min-w-0 flex-1">
						<span class="block text-sm text-text-primary">{item.label}</span>
						<span class="mt-0.5 block text-xxs text-text-secondary">{active ? '正在显示' : '已隐藏'}</span>
					</span>
					{#if count !== undefined}
						<span class="shrink-0 text-xs text-text-secondary">{active ? count : '—'}</span>
					{/if}
					<span
						class="flex size-5 shrink-0 items-center justify-center rounded-full transition-colors {active ? 'bg-accent-default text-text-on-accent' : 'bg-border-default text-transparent'}"
					>
						<Check class="size-3" strokeWidth={3} />
					</span>
				</button>
			{/each}
		</div>
		<div class="flex items-center justify-between border-t border-border-default px-3.5 py-2">
			{#if showLegend}
				<span class="flex items-center gap-2 text-xs text-text-secondary">
					<span class="size-3 rounded-full" style="background:var(--cst-marker-user);border:var(--cst-border-width-strong) solid var(--cst-surface-primary);box-shadow:var(--cst-marker-halo) var(--cst-marker-user-glow);"></span>
					我的位置
				</span>
			{:else}
				<span></span>
			{/if}
			<button
				type="button"
				class="link-btn focus-ring rounded-sm px-1 text-xs text-accent-default {allActive ? 'pointer-events-none opacity-soft' : ''}"
				disabled={allActive}
				onclick={showAll}
			>
				全部显示
			</button>
		</div>
	</Popover.Content>
</Popover.Root>

<script lang="ts">
	import { SlidersHorizontal } from '@lucide/svelte';
	import * as Popover from '@/components/ui/popover';
	import MenuPanel from '@/components/overlays/MenuPanel.svelte';
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
				? 'border-transparent bg-accent-default text-text-on-accent shadow-raised-sm'
				: 'border-border-default bg-surface-primary text-text-primary shadow-raised-sm'}"
			aria-label="筛选地图显示类型"
			aria-expanded={open}
			onclick={() => (open = !open)}
		>
			<SlidersHorizontal class="size-3.5" strokeWidth={open ? 2.5 : 1.75} />
			筛选
			{#if !allActive}
				<!-- 展开态按钮底为 accent 实色，徽章改用「反色对」（on-accent 底 + accent 字）——
				     与 LocationDesktopPanel 的选中图钉标签同款。此前用 25% 半透明白底配
				     on-accent 文字，浅色模式下为白字压浅底（≈3.4:1，低于 WCAG AA 4.5:1）
				     且叠在 accent 实底上近乎隐形。 -->
				<span class="text-xxs font-medium rounded-full" style="margin-left:2px;padding:1px 6px;background:{open ? 'var(--cst-text-on-accent)' : 'var(--cst-accent-soft)'};color:var(--cst-accent-default);">{activeFilters.size}</span>
			{/if}
		</button>
	</div>

	<Popover.Content {align} side="bottom" sideOffset={8} customAnchor={anchorEl} class="p-0">
		<MenuPanel
			title="显示类型"
			meta="{activeFilters.size}/3 已选"
			ariaLabel="筛选地图显示类型"
			multi
			options={ALL.map((item) => {
				const active = activeFilters.has(item.type);
				const count = counts?.[item.type];
				return {
					id: item.type,
					title: item.label,
					desc: active ? '正在显示' : '已隐藏',
					badge: count !== undefined ? (active ? `${count}` : '—') : undefined,
					selected: active,
					dimmed: !active
				};
			})}
			onSelect={(id) => toggleFilter(id as PinType)}
			footer={{ actionLabel: '全部显示', actionDisabled: allActive, onAction: showAll }}
		>
			{#snippet leading(opt)}
				{@const type = opt.id as PinType}
				{@const active = activeFilters.has(type)}
				<span
					class="inline-flex size-8 shrink-0 items-center justify-center rounded-lg"
					style="background:{active ? PIN_TONES[type].bg : 'var(--cst-surface-primary)'};border:var(--cst-border-width-hairline) solid {active ? PIN_TONES[type].border : 'transparent'};"
				>
					{@html markerSvg(type, 1, { w: 14, h: 19 })}
				</span>
			{/snippet}
			{#snippet footerLeading()}
				{#if showLegend}
					<span class="size-3 rounded-full" style="background:var(--cst-marker-user);border:var(--cst-border-width-strong) solid var(--cst-surface-primary);box-shadow:var(--cst-marker-halo) var(--cst-marker-user-glow);"></span>
					我的位置
				{/if}
			{/snippet}
		</MenuPanel>
	</Popover.Content>
</Popover.Root>

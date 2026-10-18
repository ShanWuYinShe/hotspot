<script lang="ts">
	// 地图源快速切换（高德 / OpenStreetMap）：地图页与设置页共享同一偏好
	// store，切换即时生效（MapCanvas 按源换瓦片 URL 与纠偏策略）。
	// 浮层走统一 Popover 原语（与 MapFilterMenu 同款：Portal 定位自动避让边缘、
	// 外点/Escape 关闭、入场动画），不再手写 absolute 定位——此前手写版在移动端
	// 永远向下展开，会压住底部 Sheet。
	import { Layers } from '@lucide/svelte';
	import * as Popover from '@/components/ui/popover';
	import MenuPanel from '@/components/overlays/MenuPanel.svelte';
	import { mapSourceStore, setMapSource, MAP_SOURCE_OPTIONS, type MapSource } from '@/stores/mapSource.svelte';

	let {
		align = 'start',
		side = 'bottom',
		size = 'sm'
	}: { align?: 'start' | 'end'; side?: 'top' | 'bottom'; size?: 'sm' | 'md' } = $props();

	let open = $state(false);
	let anchorEl = $state<HTMLElement | null>(null);

	function pick(id: MapSource) {
		setMapSource(id);
		open = false;
	}
</script>

<Popover.Root bind:open>
	<div bind:this={anchorEl} style="display:inline-block;">
		<button
			type="button"
			class="hover-btn hit-slop flex items-center justify-center rounded-full focus-ring {size === 'md' ? 'size-10.5' : 'size-10'}"
			style="background:var(--cst-surface-primary);color:var(--cst-text-primary);border:var(--cst-border-width-hairline) solid var(--cst-border-default);box-shadow:var(--cst-elevation-fab);"
			aria-label="切换地图源（当前：{MAP_SOURCE_OPTIONS.find((o) => o.id === mapSourceStore.source)?.label}）"
			aria-expanded={open}
			title="切换地图源"
			onclick={() => (open = !open)}
		>
			<Layers class="{size === 'md' ? 'size-4.5' : 'size-4'} text-accent-default" strokeWidth={1.75} />
		</button>
	</div>

	<Popover.Content {align} {side} sideOffset={8} customAnchor={anchorEl} class="p-0">
		<MenuPanel
			title="地图源"
			ariaLabel="地图源"
			widthClass="w-50"
			options={MAP_SOURCE_OPTIONS.map((o) => ({
				id: o.id,
				title: o.label,
				desc: o.note,
				selected: mapSourceStore.source === o.id
			}))}
			onSelect={(id) => pick(id as MapSource)}
		/>
	</Popover.Content>
</Popover.Root>

<script lang="ts">
	// 地图源快速切换（高德 / OpenStreetMap）：地图页与设置页共享同一偏好
	// store，切换即时生效（MapCanvas 按源换瓦片 URL 与纠偏策略）。
	import { Layers } from '@lucide/svelte';
	import { mapSourceStore, setMapSource, MAP_SOURCE_OPTIONS, type MapSource } from '@/stores/mapSource.svelte';

	let {
		align = 'start',
		size = 'sm'
	}: { align?: 'start' | 'end'; size?: 'sm' | 'md' } = $props();

	let open = $state(false);

	function pick(id: MapSource) {
		setMapSource(id);
		open = false;
	}
</script>

<div class="relative" style="display:inline-block;">
	<button
		type="button"
		class="hover-btn flex items-center justify-center rounded-full focus-ring {size === 'md' ? 'size-10.5' : 'size-10'}"
		style="background:var(--cst-surface-primary);color:var(--cst-text-primary);border:var(--cst-border-width-hairline) solid var(--cst-border-default);box-shadow:var(--cst-elevation-fab);"
		aria-label="切换地图源（当前：{MAP_SOURCE_OPTIONS.find((o) => o.id === mapSourceStore.source)?.label}）"
		aria-expanded={open}
		title="切换地图源"
		onclick={(e) => {
			e.stopPropagation();
			open = !open;
		}}
	>
		<Layers class="{size === 'md' ? 'size-4.5' : 'size-4'} text-accent-default" strokeWidth={1.75} />
	</button>

	{#if open}
		<!-- 外点关闭：透明遮罩层，不影响下层地图手势 -->
		<button
			type="button"
			class="fixed inset-0 z-40 cursor-default"
			style="background:transparent;"
			aria-label="关闭地图源菜单"
			onclick={() => (open = false)}
		></button>
		<div
			class="absolute z-50 rounded-xl bg-surface-primary p-1.5 shadow-raised-md"
			style="border:var(--cst-border-width-hairline) solid var(--cst-border-default);width:200px;{align === 'end' ? 'right:0;' : 'left:0;'}top:calc(100% + 8px);"
			role="menu"
		>
			<p class="px-2.5 pt-1 pb-1.5 text-xxs" style="color:var(--cst-text-secondary);">地图源</p>
			{#each MAP_SOURCE_OPTIONS as opt (opt.id)}
				<button
					type="button"
					role="menuitemradio"
					aria-checked={mapSourceStore.source === opt.id}
					class="flex w-full items-start gap-2 rounded-lg px-2.5 py-2 text-left focus-ring {mapSourceStore.source === opt.id ? 'bg-accent-soft' : 'hover:bg-accent-soft'}"
					onclick={() => pick(opt.id)}
				>
					<span class="flex-1 min-w-0">
						<span class="block text-sm text-text-primary">{opt.label}</span>
						<span class="block text-xxs text-text-secondary">{opt.note}</span>
					</span>
					{#if mapSourceStore.source === opt.id}
						<span class="text-accent-default text-xs" aria-hidden="true">✓</span>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>

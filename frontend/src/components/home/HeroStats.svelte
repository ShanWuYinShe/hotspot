<script lang="ts">
	import { Radar, ChevronRight } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';

	// Hero 统计横幅——单一焦点 + 环境光晕。

	let {
		onExplore,
		stats,
		locationName = ''
	}: {
		onExplore: () => void;
		stats: { nearby: string; active: string; online: string };
		locationName?: string;
	} = $props();

	const rangeLabel = $derived(
		platform.platform === 'mobile'
			? (locationName ? `${locationName} · 5km 内 · 持续更新` : '5km 内 · 持续更新')
			: '5km 内 · 持续更新'
	);
</script>

	<button
		type="button"
		class="hover-btn tap-97 rounded-xl border border-border-default bg-surface-primary shadow-raised-sm cursor-pointer overflow-hidden text-left w-full py-4.5 px-5"
		aria-label="查看附近的人"
		onclick={onExplore}
	>
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-2.5">
				<div class="flex items-center justify-center rounded-full bg-accent-default size-9.5">
					<Radar class="size-4.5 text-text-on-accent"   strokeWidth={2} />
				</div>
				<div>
					<p class="text-sm font-medium text-text-primary" >实时附近</p>
					<p class="text-xs mt-px text-text-secondary"  >
						{rangeLabel}
					</p>
				</div>
			</div>
			<ChevronRight class="size-4 text-text-secondary"   strokeWidth={1.5} />
		</div>

		<!-- Stats row：display 数字（大号粗体 + 紧字距），对齐 Figma stats 模式 -->
		<div class="grid grid-cols-3 mt-4 bg-border-default gap-px rounded-md overflow-hidden">
			<div class="flex flex-col items-center justify-center py-3 bg-surface-primary">
				<span class="text-2xl font-bold leading-none tracking-tight tabular-nums text-text-primary" >{stats.nearby}</span>
				<span class="text-xxs mt-1.5 text-text-secondary"  >人附近</span>
			</div>
			<div class="flex flex-col items-center justify-center py-3 bg-surface-primary">
				<span class="text-2xl font-bold leading-none tracking-tight tabular-nums text-text-primary" >{stats.active}</span>
				<span class="text-xxs mt-1.5 text-text-secondary"  >动态</span>
			</div>
			<div class="flex flex-col items-center justify-center py-3 bg-surface-primary">
				<span class="text-2xl font-bold leading-none tracking-tight tabular-nums text-text-primary" >{stats.online}</span>
				<span class="text-xxs mt-1.5 text-text-secondary"  >在线</span>
			</div>
		</div>
	</button>

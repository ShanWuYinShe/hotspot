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

<div class="ambient-glow">
	<button
		type="button"
		class="hover-btn tap-99 surface-hero rounded-feature cursor-pointer overflow-hidden text-left w-full"
		style="padding:18px 20px;"
		aria-label="查看附近的人"
		onclick={onExplore}
	>
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-2.5">
				<div
					class="flex items-center justify-center rounded-full bg-accent-default"
					style="width:38px;height:38px;box-shadow:0 4px 14px var(--glow-accent);"
				>
					<Radar style="width:17px;height:17px;color:var(--text-on-accent);" strokeWidth={2} />
				</div>
				<div>
					<p class="text-base" style="color:var(--text-primary);font-weight:500;">实时附近</p>
					<p class="text-xs" style="color:var(--text-secondary);margin-top:1px;">
						{rangeLabel}
					</p>
				</div>
			</div>
			<ChevronRight style="width:16px;height:16px;color:var(--text-secondary);" strokeWidth={1.5} />
		</div>

		<!-- Stats row -->
		<div class="grid grid-cols-3 mt-4 bg-border-default" style="gap:1px;border-radius:var(--shape-control);overflow:hidden;">
			<div class="flex flex-col items-center justify-center py-2.5 bg-surface-primary">
				<span class="text-[19px] font-semibold" style="color:var(--accent-default);line-height:1;">{stats.nearby}</span>
				<span class="text-xxs" style="color:var(--text-secondary);margin-top:4px;">人附近</span>
			</div>
			<div class="flex flex-col items-center justify-center py-2.5 bg-surface-primary">
				<span class="text-[19px] font-semibold" style="color:var(--status-warning);line-height:1;">{stats.active}</span>
				<span class="text-xxs" style="color:var(--text-secondary);margin-top:4px;">活动中</span>
			</div>
			<div class="flex flex-col items-center justify-center py-2.5 bg-surface-primary">
				<span class="text-[19px] font-semibold" style="color:var(--status-online);line-height:1;">{stats.online}</span>
				<span class="text-xxs" style="color:var(--text-secondary);margin-top:4px;">在线</span>
			</div>
		</div>
	</button>
</div>

<script lang="ts">
	import { Radar, ChevronRight } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';

	// Hero 统计横幅——单一焦点 + 环境光晕（迁移自旧版 NearbyFeedPage）。

	let {
		onExplore,
		stats
	}: {
		onExplore: () => void;
		stats: { nearby: string; active: string; online: string };
	} = $props();

	const rangeLabel = $derived(
		platform.platform === 'mobile' ? '徐州 · 5km 内 · 持续更新' : '5km 内 · 持续更新'
	);
</script>

<div class="neu-ambient-glow">
	<button
		type="button"
		class="neu-hover-btn neu-tap-99 neu-surface-hero rounded-feature cursor-pointer overflow-hidden text-left w-full"
		style="padding:18px 20px;"
		aria-label="查看附近的人"
		onclick={onExplore}
	>
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-2.5">
				<div
					class="flex items-center justify-center rounded-full bg-[var(--neu-accent-primary)]"
					style="width:38px;height:38px;box-shadow:0 4px 14px var(--neu-glow);"
				>
					<Radar style="width:17px;height:17px;color:var(--text-on-accent);" strokeWidth={2} />
				</div>
				<div>
					<p class="text-[13px]" style="color:var(--neu-text);font-weight:500;">实时附近</p>
					<p class="text-[11px]" style="color:var(--neu-text-muted);margin-top:1px;">
						{rangeLabel}
					</p>
				</div>
			</div>
			<ChevronRight style="width:16px;height:16px;color:var(--neu-text-muted);" strokeWidth={1.5} />
		</div>

		<!-- Stats row -->
		<div class="grid grid-cols-3 mt-4 bg-[var(--neu-border)]" style="gap:1px;border-radius:var(--shape-control);overflow:hidden;">
			<div class="flex flex-col items-center justify-center py-2.5 bg-[var(--neu-bg)]">
				<span class="text-[19px] font-semibold" style="color:var(--neu-accent-primary);line-height:1;">{stats.nearby}</span>
				<span class="text-[10px]" style="color:var(--neu-text-muted);margin-top:4px;">人附近</span>
			</div>
			<div class="flex flex-col items-center justify-center py-2.5 bg-[var(--neu-bg)]">
				<span class="text-[19px] font-semibold" style="color:var(--semantic-warning);line-height:1;">{stats.active}</span>
				<span class="text-[10px]" style="color:var(--neu-text-muted);margin-top:4px;">活动中</span>
			</div>
			<div class="flex flex-col items-center justify-center py-2.5 bg-[var(--neu-bg)]">
				<span class="text-[19px] font-semibold" style="color:var(--semantic-online);line-height:1;">{stats.online}</span>
				<span class="text-[10px]" style="color:var(--neu-text-muted);margin-top:4px;">在线</span>
			</div>
		</div>
	</button>
</div>

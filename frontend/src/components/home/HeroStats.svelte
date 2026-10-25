<script lang="ts">
import { formatCount } from '@/lib/utils';
	import { Radar, ChevronRight } from '@lucide/svelte';

	// Hero 统计横幅——单一焦点 + 环境光晕。

	let {
		onExplore,
		stats,
		locationName = ''
	}: {
		onExplore: () => void;
		stats: { nearby: number; active: number; online: number };
		locationName?: string;
	} = $props();

	// 位置名两端统一显示：首页与附近页共用浏览锚点/默认位置基准（此前桌面端
	// 恒为「5km 内 · 实时更新」，用户自定义锚点后看不到当前基准，与附近页
	// 列表数对不上时无从解释）。
	const rangeLabel = $derived(locationName ? `${locationName} · 5km 内` : '5km 内 · 实时更新');
</script>

	<button
		type="button"
		class="group hover-btn tap-97 rounded-2xl border border-border-subtle/50 bg-surface-primary/85 backdrop-blur-xs shadow-xs hover:border-accent-default/30 hover:shadow-raised-sm cursor-pointer overflow-hidden text-left w-full py-4.5 px-5 transition-all"
		aria-label="查看附近的人"
		onclick={onExplore}
	>
		<!-- 标题行弹性契约（对齐 PageHeader）：左侧文本簇 flex-1 min-w-0 先收缩，
		     右箭头 shrink-0 锁定；定位名过长时副标题单行截断，不顶出箭头、不贴卡边裁字。 -->
		<div class="flex items-center justify-between gap-2">
			<div class="flex min-w-0 flex-1 items-center gap-3">
				<!-- 雷达声纳声波呼吸环动效 -->
				<div class="relative flex shrink-0 items-center justify-center rounded-full bg-accent-default size-10 shadow-sm">
					<span class="absolute inset-0 rounded-full bg-accent-default/30 animate-ping"></span>
					<Radar class="relative size-5 text-text-on-accent transition-transform duration-standard group-hover:scale-110" strokeWidth={2} />
				</div>
				<div class="min-w-0">
					<div class="flex items-center gap-2">
						<p class="text-sm font-semibold text-text-primary">实时附近</p>
						<span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-status-online/10 text-status-online text-xxs font-medium">
							<span class="size-1.5 rounded-full bg-status-online animate-pulse"></span>
							实时雷达
						</span>
					</div>
					<p class="text-xs mt-0.5 truncate text-text-secondary">
						{rangeLabel}
					</p>
				</div>
			</div>
			<ChevronRight class="size-4 shrink-0 text-text-secondary transition-transform duration-rapid group-hover:translate-x-0.5 group-hover:text-accent-default" strokeWidth={1.5} />
		</div>

		<!-- Stats row：现代通透 Bento 指标列，告别生硬方框与切线 -->
		<div class="grid grid-cols-3 mt-4 gap-2.5">
			<div class="flex flex-col items-center justify-center py-3 rounded-xl bg-surface-subtle/40 backdrop-blur-xs transition-colors group-hover:bg-surface-subtle/70">
				<span class="text-2xl font-bold leading-none tracking-tight tabular-nums text-text-primary">{formatCount(stats.nearby)}</span>
				<span class="text-xxs mt-1.5 text-text-secondary font-medium">人附近</span>
			</div>
			<div class="flex flex-col items-center justify-center py-3 rounded-xl bg-surface-subtle/40 backdrop-blur-xs transition-colors group-hover:bg-surface-subtle/70">
				<span class="text-2xl font-bold leading-none tracking-tight tabular-nums text-text-primary">{formatCount(stats.active)}</span>
				<span class="text-xxs mt-1.5 text-text-secondary font-medium">瞬间</span>
			</div>
			<div class="flex flex-col items-center justify-center py-3 rounded-xl bg-surface-subtle/40 backdrop-blur-xs transition-colors group-hover:bg-surface-subtle/70">
				<span class="text-2xl font-bold leading-none tracking-tight tabular-nums text-text-primary">{formatCount(stats.online)}</span>
				<span class="text-xxs mt-1.5 text-text-secondary font-medium">在线</span>
			</div>
		</div>
	</button>

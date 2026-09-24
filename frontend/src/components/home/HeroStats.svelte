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
		class="hover-btn tap-97 rounded-xl border border-border-default bg-surface-primary shadow-raised-sm hover:border-accent-default hover:shadow-raised-md cursor-pointer overflow-hidden text-left w-full py-4.5 px-5"
		aria-label="查看附近的人"
		onclick={onExplore}
	>
		<!-- 标题行弹性契约（对齐 PageHeader）：左侧文本簇 flex-1 min-w-0 先收缩，
		     右箭头 shrink-0 锁定；定位名过长时副标题单行截断，不顶出箭头、不贴卡边裁字。 -->
		<div class="flex items-center justify-between gap-2">
			<div class="flex min-w-0 flex-1 items-center gap-2.5">
				<div class="flex shrink-0 items-center justify-center rounded-full bg-accent-default size-9.5">
					<Radar class="size-4.5 text-text-on-accent"   strokeWidth={2} />
				</div>
				<div class="min-w-0">
					<p class="text-sm font-medium text-text-primary" >实时附近</p>
					<p class="text-xs mt-px truncate text-text-secondary"  >
						{rangeLabel}
					</p>
				</div>
			</div>
			<ChevronRight class="size-4 shrink-0 text-text-secondary"   strokeWidth={1.5} />
		</div>

		<!-- Stats row：display 数字（大号粗体 + 紧字距），对齐 Figma stats 模式 -->
		<div class="grid grid-cols-3 mt-4 bg-border-default gap-px rounded-md overflow-hidden">
			<div class="flex flex-col items-center justify-center py-3 bg-surface-primary">
				<span class="text-2xl font-semibold leading-none tracking-tight tabular-nums text-text-primary" >{formatCount(stats.nearby)}</span>
				<span class="text-xxs mt-1.5 text-text-secondary"  >人附近</span>
			</div>
			<div class="flex flex-col items-center justify-center py-3 bg-surface-primary">
				<span class="text-2xl font-semibold leading-none tracking-tight tabular-nums text-text-primary" >{formatCount(stats.active)}</span>
				<span class="text-xxs mt-1.5 text-text-secondary"  >动态</span>
			</div>
			<div class="flex flex-col items-center justify-center py-3 bg-surface-primary">
				<span class="text-2xl font-semibold leading-none tracking-tight tabular-nums text-text-primary" >{formatCount(stats.online)}</span>
				<span class="text-xxs mt-1.5 text-text-secondary"  >在线</span>
			</div>
		</div>
	</button>

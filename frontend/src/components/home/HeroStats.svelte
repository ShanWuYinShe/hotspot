<script lang="ts">
	import { Radar, ChevronRight } from '@lucide/svelte';
	import MetricRow from '@/components/data-display/MetricRow.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import { formatCount } from '@/lib/utils';

	// Hero 统计横幅——单一焦点 + 发丝分隔的数字节奏（R13「数字与秩序」母题）。
	// 旧的三个灰底圆角指标块（卡片里的卡片）已去掉，改为标题行 / 发丝线 / MetricRow(strip)：
	// 数字 24px semibold tabular 是唯一视觉锚点，标签 12px 次级，靠竖发丝线建立秩序。

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

	// 指标项：数值字符串化交给 MetricRow 统一排版；仅「在线」用一处状态强调（2px 圆点 + 状态色），
	// 其余保持中性，避免"每个数字都染色"。
	const metrics = $derived([
		{ value: formatCount(stats.nearby), label: '人' },
		{ value: formatCount(stats.active), label: '瞬间' },
		{ value: formatCount(stats.online), label: '在线', tone: 'positive' as const }
	]);
</script>

	<button
		type="button"
		class="group hover-btn tap-97 rounded-lg bg-surface-island shadow-xs hover:shadow-raised-sm cursor-pointer overflow-hidden text-left w-full pt-4 pb-4 px-5 transition-all"
		aria-label="查看附近的人"
		onclick={onExplore}
	>
		<!-- 标题行弹性契约（对齐 PageHeader）：左侧文本簇 flex-1 min-w-0 先收缩，
		     右箭头 shrink-0 锁定；定位名过长时副标题单行截断，不顶出箭头、不贴卡边裁字。 -->
		<div class="flex items-center justify-between gap-2">
			<div class="flex min-w-0 flex-1 items-center gap-3">
				<!-- 雷达声纳声波呼吸环动效（单处动效：徽章圆点保持静态，避免叠两层动画） -->
				<div class="relative flex shrink-0 items-center justify-center rounded-full bg-accent-default size-10 shadow-sm">
					<span class="absolute inset-0 rounded-full bg-accent-default/30 animate-ping"></span>
					<Radar class="relative size-5 text-text-on-accent transition-transform duration-standard group-hover:scale-110" strokeWidth={2} />
				</div>
				<div class="min-w-0">
					<div class="flex items-center gap-2">
						<p class="text-md font-semibold text-text-primary">实时附近</p>
						<Badge tone="online">
							<span class="size-1.5 rounded-full bg-status-online"></span>
							实时雷达
						</Badge>
					</div>
					<p class="text-xs mt-0.5 truncate text-text-secondary">
						{rangeLabel}
					</p>
				</div>
			</div>
			<ChevronRight class="size-4 shrink-0 text-text-secondary transition-transform duration-rapid group-hover:translate-x-0.5 group-hover:text-accent-default" strokeWidth={1.5} />
		</div>

		<!-- 数字与秩序：标题行 → 发丝线 → 三等分指标（无灰底块、无卡片里的卡片） -->
		<MetricRow items={metrics} variant="strip" class="mt-3.5 border-t border-border-subtle pt-4" />
	</button>

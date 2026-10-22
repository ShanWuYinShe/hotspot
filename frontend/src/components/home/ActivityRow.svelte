<script lang="ts">
	import { ChevronRight, Coffee, Music, Users, Star } from '@lucide/svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import type { MomentItem } from '@/api/types';
	import { formatDistance, formatRelativeDays } from '@/lib/utils';

	// 附近瞬间行（首页「附近活动」区共用）：色块 icon + title/sub + chevron。
	// 图标/配色按 index 轮换（与 Figma HomePage 活动列表契约一致）。
	// onClick 携带 moment_id 交给页面导航到 /moments/[momentId]。

	const ICONS = [Coffee, Music, Users, Star] as const;
	const COLORS = [
		'var(--cst-status-warning)',
		'var(--cst-accent-default)',
		'var(--cst-marker-group)',
		'var(--cst-status-danger)'
	] as const;
	const BGS = [
		'var(--cst-status-warning-bg)',
		'var(--cst-accent-soft)',
		'var(--cst-marker-group-bg)',
		'var(--cst-status-danger-bg)'
	] as const;

	// showDistance：行内 meta 是否带「距离」。
	// 附近轴（附近瞬间/附近活动/搜索发现态的附近语义）需要它；**非地理轴**（首页「关注」关系流、
	// 首页「发现」全站热榜）不需要——全局热榜配「300m」会让用户误以为是身边内容。
	// 默认 true 保持既有调用方行为不变。
	let {
		moment,
		index,
		onClick,
		showDistance = true
	}: { moment: MomentItem; index: number; onClick: () => void; showDistance?: boolean } = $props();

	const Icon = $derived(ICONS[index % ICONS.length]);
	const color = $derived(COLORS[index % COLORS.length]);
	const bg = $derived(BGS[index % BGS.length]);

	// meta 行补实时距离与相对时间(数据已下发未映射;Gemini 审查 r01:活动项缺决策信息)
	const distanceLabel = $derived(showDistance ? formatDistance(moment.location?.distance) : '');
	const timeLabel = $derived.by(() => {
		const t = Date.parse(moment.created_at ?? '');
		if (Number.isNaN(t)) return '';
		return formatRelativeDays(Math.max(0, Math.floor((Date.now() - t) / 86_400_000)));
	});
	const metaLine = $derived(
		[moment.location?.location_name ?? '', distanceLabel, timeLabel].filter(Boolean).join(' · ')
	);
</script>

<ListItem
	onClick={onClick}
	ariaLabel={moment.description}
	class="flex items-center gap-3 px-4 py-3"
>
	<div
		class="rounded-sm flex items-center justify-center shrink-0 size-8.5"
		style="background:{bg};"
	>
		<Icon class="size-4" style="color:{color};" strokeWidth={1.75} />
	</div>
	<div class="flex-1 min-w-0">
		<p class="text-sm text-text-primary"  >{moment.description}</p>
		<p class="text-xs mt-px text-text-secondary"  >{metaLine}</p>
	</div>
	<ChevronRight class="size-3.5 text-text-secondary"   strokeWidth={1.5} />
</ListItem>

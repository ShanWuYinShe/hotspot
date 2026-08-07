<script lang="ts">
	import { ChevronRight, Coffee, Music, Users, Star } from '@lucide/svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import type { MomentItem } from '@/api/types';

	// 附近动态行（首页「附近活动」区共用）：色块 icon + title/sub + chevron。
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

	let { moment, index, onClick }: { moment: MomentItem; index: number; onClick: () => void } = $props();

	const Icon = $derived(ICONS[index % ICONS.length]);
	const color = $derived(COLORS[index % COLORS.length]);
	const bg = $derived(BGS[index % BGS.length]);
</script>

<ListItem
	onClick={onClick}
	ariaLabel={moment.description}
	class="flex items-center gap-3 px-4 py-3"
>
	<div
		class="rounded-tight flex items-center justify-center shrink-0"
		style="width:34px;height:34px;background:{bg};"
	>
		<Icon style="width:16px;height:16px;color:{color};" strokeWidth={1.75} />
	</div>
	<div class="flex-1 min-w-0">
		<p class="text-base"  style="color:var(--cst-text-primary)">{moment.description}</p>
		<p class="text-xs mt-px"  style="color:var(--cst-text-secondary)">{moment.location?.location_name ?? ''}</p>
	</div>
	<ChevronRight class="size-3.25"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
</ListItem>

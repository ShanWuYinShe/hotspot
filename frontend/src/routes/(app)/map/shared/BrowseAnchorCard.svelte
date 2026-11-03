<script lang="ts">
	import { MapPin, Navigation, RotateCcw } from '@lucide/svelte';
	import Button from '@/components/ui/button.svelte';
	import type { MapPoint } from '@/lib/map-auto-locate';
	import { formatDistance } from '@/lib/utils';
	import { anchorDisplayName, anchorDistanceMeters, anchorPurposeHint } from './browse-anchor';

	// 浏览锚点卡片（桌面左面板 / 移动底部 Sheet 共用）：
	// 明确区分「已保存的浏览锚点」与「地图当前中心」两个概念，并提供恢复默认动作。
	// 用户反馈「不知道怎么设置浏览锚点」的根因：旧 UI 只有一张叫「浏览位置」的卡，
	// 实际显示的是地图中心，且没有已保存锚点的状态与地图标记。
	let {
		anchor,
		center,
		centerName,
		defaultName,
		onReset
	}: {
		anchor: MapPoint | null;
		center: { lat: number; lng: number } | null;
		centerName: string;
		defaultName?: string;
		onReset: () => void;
	} = $props();

	const distance = $derived(anchor && center ? anchorDistanceMeters(center, anchor) : 0);
	const hasAnchor = $derived(anchor != null);
</script>

<section
	aria-label="浏览锚点设置"
	class="rounded-md border border-border-default/50 bg-surface-primary p-3"
>
	<div class="flex items-center gap-2.5">
		<span class="flex size-9 shrink-0 items-center justify-center rounded-full bg-accent-soft text-accent-default">
			<Navigation class="size-4" strokeWidth={2} aria-hidden="true" />
		</span>
		<div class="min-w-0 flex-1">
			<span class="block text-xxs font-medium uppercase tracking-eyebrow text-accent-default">浏览锚点</span>
			<p class="mt-0.5 truncate text-md font-medium text-text-primary">
				{hasAnchor ? anchorDisplayName(anchor) : '未设置'}
			</p>
		</div>
		{#if hasAnchor}
			<Button variant="soft" size="sm" class="hit-slop shrink-0 gap-1 px-2" aria-label="恢复默认浏览锚点" onclick={onReset}>
				<RotateCcw class="size-3.5" strokeWidth={1.75} aria-hidden="true" />
				恢复默认
			</Button>
		{/if}
	</div>

	<p class="mt-1.5 text-xxs text-text-secondary">
		{hasAnchor ? anchorPurposeHint(true) : (defaultName ? `默认位置：${defaultName}` : anchorPurposeHint(false))}
	</p>

	<div class="mt-2 flex items-center gap-2 rounded-sm bg-surface-subtle/60 px-2.5 py-2">
		<MapPin class="size-3.5 shrink-0 text-text-secondary" strokeWidth={1.75} aria-hidden="true" />
		<span class="shrink-0 text-xxs text-text-secondary">地图中心</span>
		<span class="min-w-0 flex-1 truncate text-right text-xs font-medium text-text-primary">{centerName || '未命名区域'}</span>
	</div>
	{#if hasAnchor && distance > 0}
		<p class="mt-1 text-right text-xxs tabular-nums text-text-secondary">距锚点 {formatDistance(distance)}</p>
	{:else if hasAnchor}
		<p class="mt-1 text-right text-xxs text-text-secondary">地图中心即当前锚点</p>
	{/if}
</section>

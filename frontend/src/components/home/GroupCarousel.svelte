<script lang="ts">
	import type { Component } from 'svelte';
	import { goto } from '$app/navigation';
	import { scrollFade } from '@/lib/scroll-fade';
	import { MapPin, Users, Lock, Globe } from '@lucide/svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import { ROUTES } from '@/lib/routes';

	// 附近群组横向轮播（对齐 Figma Make HomePage GroupCarousel）。
	// 140×140 正方形卡片：banner 铺满 + 底部 card-menu 信息 tray 覆盖在图上。
	// 基于 NearbyImageCard 统一图片优先原语：hover 抬升/图片缩放由 card-hover 驱动，
	// tray/scrim 与附近内容卡一致；卡头走 SectionHead（icon/count/action 与枢纽三段统一）。

	export interface NearbyGroup {
		id: string;
		name: string;
		members: number;
		banner: string;
		distance: string;
		privacy: 'public' | 'private';
	}

	let {
		groups,
		onSeeAll,
		icon,
		count,
		actionLabel = '更多'
	}: {
		groups: NearbyGroup[];
		onSeeAll: () => void;
		/** 卡头图标（枢纽页传 UsersRound，与其它两段语言一致） */
		icon?: Component;
		/** 卡头计数（群组总数） */
		count?: number;
		/** 右侧动作文案（枢纽页统一「查看全部」，独立使用时保留「更多」） */
		actionLabel?: string;
	} = $props();

	const openGroup = (id: string) => void goto(ROUTES.group(id));
</script>

<!-- 大纲页密度：无群组时整段隐藏（标题+空横滚），专区经导航可达 -->
{#if groups.length > 0}
<div>
	<SectionHead title="附近群组" {icon} {count} action={actionLabel} onAction={onSeeAll} />
	<div class="flex gap-3 overflow-x-auto no-scrollbar pb-2 pr-1" use:scrollFade>
		{#each groups as g (g.id)}
			<NearbyImageCard
				imageSrc={g.banner}
				imageAlt={g.name}
				onClick={() => openGroup(g.id)}
				class="aspect-square w-35 shrink-0 rounded-lg shadow-xs"
			>
				{#snippet placeholder()}
					<span class="text-accent-default text-3xl font-semibold">{Array.from(g.name)[0] ?? '群'}</span>
				{/snippet}

				<!-- 右上：隐私徽章（纯图标，title/aria 提供语义；与其它卡片角标同位置/同尺寸） -->
				<div
					class="absolute right-2.5 top-2.5 flex size-5 items-center justify-center rounded-full bg-surface-overlay-strong"
					title={g.privacy === 'private' ? '私密群组' : '公开群组'}
					aria-hidden="true"
				>
					{#if g.privacy === 'private'}
						<Lock class="size-2.5 text-content-inverse" strokeWidth={2} />
					{:else}
						<Globe class="size-2.5 text-content-inverse" strokeWidth={2} />
					{/if}
				</div>

				{#snippet footer()}
					<div class="px-2.5 pb-2.5 pt-2">
						<p class="truncate text-sm font-semibold text-text-primary">{g.name}</p>
						<div class="mt-1 flex items-center justify-between gap-1">
							<span class="flex min-w-0 items-center gap-1 text-xs tabular-nums text-text-secondary">
								<Users class="size-2.5 shrink-0" strokeWidth={2} />
								{g.members} 成员
							</span>
							<span class="flex shrink-0 items-center gap-0.5 text-xs tabular-nums text-accent-default">
								<MapPin class="size-2.5 shrink-0" strokeWidth={2} />
								{g.distance}
							</span>
						</div>
					</div>
				{/snippet}
			</NearbyImageCard>
		{/each}
	</div>
</div>
{/if}

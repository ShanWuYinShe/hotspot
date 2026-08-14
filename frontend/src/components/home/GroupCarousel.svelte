<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, Users, Lock, Globe } from '@lucide/svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import { ROUTES } from '@/lib/routes';

	// 附近群组横向轮播（对齐 Figma Make HomePage GroupCarousel）。
	// 140×140 正方形卡片：banner 铺满 + 底部 card-menu 信息 tray 覆盖在图上。
	// 基于 NearbyImageCard 统一图片优先原语：hover 内嵌 accent 描边、键盘契约
	// （role=button + Enter/Space）、tray/scrim 与附近内容卡一致。

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
		onSeeAll
	}: {
		groups: NearbyGroup[];
		onSeeAll: () => void;
	} = $props();

	const openGroup = (id: string) => void goto(ROUTES.group(id));
</script>

<div>
	<SectionHead title="附近群组" action="更多" onAction={onSeeAll} />
	<div class="flex gap-3 overflow-x-auto no-scrollbar pb-1">
		{#each groups as g (g.id)}
			<NearbyImageCard
				imageSrc={g.banner}
				imageAlt={g.name}
				onClick={() => openGroup(g.id)}
				class="aspect-square w-35 shrink-0 rounded-md"
			>
				{#snippet placeholder()}
					<span class="text-accent-default text-3xl font-semibold">{Array.from(g.name)[0] ?? '群'}</span>
				{/snippet}

				<!-- 右上：隐私徽章 -->
				<div class="absolute right-2 top-2 flex h-5 w-5 items-center justify-center rounded-full bg-surface-overlay-strong">
					{#if g.privacy === 'private'}
						<Lock class="h-2.5 w-2.5 text-content-inverse" strokeWidth={2} />
					{:else}
						<Globe class="h-2.5 w-2.5 text-content-inverse" strokeWidth={2} />
					{/if}
				</div>

				{#snippet footer()}
					<div class="px-2.5 pb-2.5 pt-2">
						<p class="truncate text-sm font-medium text-text-primary">{g.name}</p>
						<div class="mt-1 flex items-center justify-between">
							<span class="flex items-center gap-1 text-xxs text-text-secondary">
								<Users class="h-2.5 w-2.5" strokeWidth={2} />
								{g.members} 成员
							</span>
							<span class="flex items-center gap-0.5 text-xxs text-accent-default">
								<MapPin class="h-2.5 w-2.5" strokeWidth={2} />
								{g.distance}
							</span>
						</div>
					</div>
				{/snippet}
			</NearbyImageCard>
		{/each}
	</div>
</div>

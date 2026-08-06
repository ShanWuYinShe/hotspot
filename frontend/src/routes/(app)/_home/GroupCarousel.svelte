<script lang="ts">
	import { MapPin, Users, Lock, Globe } from '@lucide/svelte';
	import SectionHead from '@/routes/(app)/_home/SectionHead.svelte';

	// 附近群组横向轮播（对齐 Figma Make HomePage GroupCarousel）。
	// 140×140 正方形卡片：banner 铺满 + 底部 card-menu 信息 tray 覆盖在图上。

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
</script>

<div>
	<SectionHead title="附近群组" action="更多" onAction={onSeeAll} />
	<div class="flex gap-3 overflow-x-auto no-scrollbar pb-1">
		{#each groups as g (g.id)}
			<div
				class="image-hover-frame card-hover tap-98 relative aspect-square w-35 shrink-0 cursor-pointer overflow-hidden rounded-control"
			>
				<img src={g.banner} alt={g.name} class="image-hover-frame__image absolute inset-0 h-full w-full object-cover" />
				<div class="absolute inset-0" style="background:var(--surface-media-scrim-soft);"></div>
				<div class="absolute right-2 top-2 flex h-5 w-5 items-center justify-center rounded-full" style="background:var(--surface-overlay-strong);">
					{#if g.privacy === 'private'}
						<Lock class="h-2.5 w-2.5 text-white" strokeWidth={2} />
					{:else}
						<Globe class="h-2.5 w-2.5 text-white" strokeWidth={2} />
					{/if}
				</div>
				<div class="absolute inset-x-0 bottom-0 border-t px-2.5 pb-2.5 pt-2" style="background:var(--surface-card-menu);border-color:var(--border-card-menu);">
					<p class="truncate text-sm" style="color:var(--text-primary);font-weight:500;">{g.name}</p>
					<div class="mt-1 flex items-center justify-between">
						<span class="flex items-center gap-1 text-[10px]"  style="color:var(--text-secondary)">
							<Users class="h-2.25 w-2.25" strokeWidth={1.5} />
							{g.members} 成员
						</span>
						<span class="flex items-center gap-0.5 text-[10px]"  style="color:var(--accent-default)">
							<MapPin class="h-2.25 w-2.25" strokeWidth={2} />
							{g.distance}
						</span>
					</div>
				</div>
			</div>
		{/each}
	</div>
</div>

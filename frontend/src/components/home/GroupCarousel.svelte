<script lang="ts">
	import { MapPin, Users, Lock, Globe } from '@lucide/svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';

	// 附近群组横向轮播（迁移自 React NearbyFeedPage.tsx 内 GroupCarousel）。

	export interface NearbyGroup {
		id: number;
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
			<div class="neu-hover-card flex-shrink-0 rounded-[16px] overflow-hidden cursor-pointer neu-surface-flat" style="width:140px;">
				<div class="relative" style="height:72px;">
					<img src={g.banner} alt={g.name} class="w-full h-full object-cover" />
					<div class="absolute inset-0" style="background:linear-gradient(to top, rgba(0,0,0,0.5) 0%, transparent 55%);"></div>
					<div class="absolute top-1.5 right-1.5">
						{#if g.privacy === 'private'}
							<Lock style="width:10px;height:10px;color:rgba(255,255,255,0.85);" strokeWidth={2} />
						{:else}
							<Globe style="width:10px;height:10px;color:rgba(255,255,255,0.85);" strokeWidth={2} />
						{/if}
					</div>
					<div class="absolute bottom-1.5 left-2 flex items-center gap-0.5">
						<MapPin style="width:9px;height:9px;color:rgba(255,255,255,0.9);" strokeWidth={2} />
						<span class="text-[10px]" style="color:rgba(255,255,255,0.95);">{g.distance}</span>
					</div>
				</div>
				<div class="px-2.5 pt-2 pb-2.5">
					<p class="text-xs truncate" style="color:var(--neu-text);font-weight:500;">{g.name}</p>
					<div class="flex items-center gap-1 mt-1">
						<Users style="width:9px;height:9px;color:var(--neu-text-muted);" strokeWidth={1.5} />
						<span class="text-[10px]" style="color:var(--neu-text-muted);">{g.members} 成员</span>
					</div>
				</div>
			</div>
		{/each}
	</div>
</div>

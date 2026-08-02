<script lang="ts">
	import { CalendarDays, Globe, Lock, MapPin, Users } from '@lucide/svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import JoinButton from './JoinButton.svelte';
	import type { Group } from './group-data';

	// 附近群组卡片：基于 NearbyImageCard 统一图片优先原语。
	// 分类/隐私/人数/加入等菜单以覆盖层显示在图片上。
	let { group, onToggle, onOpen }: { group: Group; onToggle: () => void; onOpen: () => void } = $props();
</script>

<NearbyImageCard
	imageSrc={group.banner ?? group.name}
	imageAlt={group.name}
	onClick={onOpen}
	class="rounded-[18px] aspect-[3/4] neu-surface-flat"
>
	<!-- 无封面时占位：覆盖在图片层之上 -->
	{#if !group.banner}
		<div
			class="absolute inset-0 z-[1] flex items-center justify-center"
			style="background:linear-gradient(135deg, color-mix(in srgb, var(--neu-accent-primary) 20%, var(--neu-bg)) 0%, color-mix(in srgb, var(--neu-accent-primary) 6%, var(--neu-bg)) 100%);"
		>
			<span class="text-[var(--neu-accent-primary)] text-3xl font-semibold">{Array.from(group.name)[0] ?? '群'}</span>
		</div>
	{/if}

	<!-- 左上：分类 -->
	<span class="absolute top-2 left-2 z-[2] px-1.5 py-0.5 rounded-full text-white bg-[var(--neu-accent-primary)] text-[9px]">{group.category}</span>

	<!-- 右上：隐私 -->
	<span class="absolute top-2 right-2 z-[2] flex items-center justify-center rounded-full" style="width:20px;height:20px;background:var(--overlay-strong);">
		{#if group.privacy === 'private'}
			<Lock size={10} color="white" />
		{:else}
			<Globe size={10} color="white" />
		{/if}
	</span>

	{#snippet footer()}
		<div class="px-3 pt-2 pb-2.5">
			<p class="line-clamp-1 text-[var(--neu-text)] text-[13px] font-semibold">{group.name}</p>
			<p class="flex items-center gap-1 truncate text-[var(--neu-text-muted)] text-[10px] mt-0.5">
				<CalendarDays size={10} />
				{group.activity}
			</p>
			<div class="mt-2 flex items-center justify-between gap-2">
				<div class="flex items-center gap-1.5 min-w-0">
					{#if group.avatars.length > 0}
						<div class="flex -space-x-2 flex-shrink-0">
							{#each group.avatars as avatar, index (index)}
								<img src={avatar} alt="群成员" class="w-5 h-5 rounded-full object-cover" style="border:1px solid var(--border-card-menu);z-index:{3 - index};" />
							{/each}
						</div>
					{/if}
					<span class="flex items-center gap-1 truncate text-[var(--neu-text-muted)] text-[10px]">
						<Users size={10} />
						{group.members} 人
						<MapPin size={10} class="ml-1" />
						{group.distance}
					</span>
				</div>
				<JoinButton {group} {onToggle} />
			</div>
		</div>
	{/snippet}
</NearbyImageCard>

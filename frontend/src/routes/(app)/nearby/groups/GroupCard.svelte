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
	class="rounded-[18px] aspect-[3/4] surface-flat"
>
	<!-- 无封面时占位：覆盖在图片层之上 -->
	{#if !group.banner}
		<div
			class="absolute inset-0 z-[1] flex items-center justify-center"
			style="background:linear-gradient(135deg, color-mix(in srgb, var(--accent-default) 20%, var(--surface-primary)) 0%, color-mix(in srgb, var(--accent-default) 6%, var(--surface-primary)) 100%);"
		>
			<span class="text-accent-default text-3xl font-semibold">{Array.from(group.name)[0] ?? '群'}</span>
		</div>
	{/if}

	<!-- 左上：分类 -->
	<span class="absolute top-2 left-2 z-[2] px-1.5 py-0.5 rounded-full text-white bg-accent-default text-[9px]">{group.category}</span>

	<!-- 右上：隐私 -->
	<span class="absolute top-2 right-2 z-[2] flex items-center justify-center rounded-full" style="width:20px;height:20px;background:var(--surface-overlay-strong);">
		{#if group.privacy === 'private'}
			<Lock size={10} color="white" />
		{:else}
			<Globe size={10} color="white" />
		{/if}
	</span>

	{#snippet footer()}
		<div class="p-3">
			<p class="line-clamp-1 text-text-primary text-[13px] font-semibold">{group.name}</p>
			<!-- stats 行：头像堆叠 + 成员 / 距离（对齐 Figma GroupCard 行序） -->
			<div class="mt-1 flex items-center justify-between gap-2 text-text-secondary text-[10px]">
				<div class="flex items-center gap-1.5 min-w-0">
					{#if group.avatars.length > 0}
						<div class="flex -space-x-2 flex-shrink-0">
							{#each group.avatars as avatar, index (index)}
								<img src={avatar} alt="群成员" class="w-5 h-5 rounded-full object-cover" style="border:1px solid var(--border-card-menu);z-index:{3 - index};" />
							{/each}
						</div>
					{/if}
					<span class="flex items-center gap-1 truncate">
						<Users size={10} />
						{group.members} 人
					</span>
				</div>
				<span class="flex items-center gap-1 flex-shrink-0 text-accent-default">
					<MapPin size={10} />
					{group.distance}
				</span>
			</div>
			<!-- activity 独立行（对齐 Figma：semantic-online 色 + CalendarDays 图标） -->
			<div class="mt-2 flex items-center justify-between gap-2">
				<span class="min-w-0 truncate text-status-online text-[10px]">
					<CalendarDays class="inline mr-1" style="width:10px;height:10px;" />
					{group.activity}
				</span>
				<JoinButton {group} {onToggle} />
			</div>
		</div>
	{/snippet}
</NearbyImageCard>

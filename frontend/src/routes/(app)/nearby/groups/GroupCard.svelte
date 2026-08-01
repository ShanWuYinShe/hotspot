<script lang="ts">
	import { CalendarDays, Globe, Lock, MapPin, Users } from '@lucide/svelte';
	import JoinButton from './JoinButton.svelte';
	import type { Group } from './group-data';

	// 附近群组卡片：封面铺满整卡，分类/隐私/人数/加入等菜单以覆盖层显示在图片上。

	let { group, onToggle, onOpen }: { group: Group; onToggle: () => void; onOpen: () => void } = $props();
</script>

<div
	class="neu-hover-card relative h-full w-full overflow-hidden rounded-[18px] aspect-[3/4] cursor-pointer neu-surface-flat neu-card-hover neu-focus-ring"
	style="background:var(--neu-bg);"
	role="button"
	tabindex="0"
	onclick={onOpen}
	onkeydown={(e) => {
		if (e.target !== e.currentTarget) return;
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onOpen();
		}
	}}
>
	{#if group.banner}
		<img src={group.banner} alt={group.name} class="absolute inset-0 h-full w-full object-cover" />
	{:else}
		<!-- 无封面：accent 主题渐变占位 -->
		<div
			class="absolute inset-0 flex items-center justify-center"
			style="background:linear-gradient(135deg, color-mix(in srgb, var(--neu-accent-primary) 20%, var(--neu-bg)) 0%, color-mix(in srgb, var(--neu-accent-primary) 6%, var(--neu-bg)) 100%);"
		>
			<span class="text-[var(--neu-accent-primary)] text-3xl font-semibold">{Array.from(group.name)[0] ?? '群'}</span>
		</div>
	{/if}

	<!-- 左上：分类 -->
	<span class="absolute top-2 left-2 px-1.5 py-0.5 rounded-full text-white bg-[var(--neu-accent-primary)] text-[9px]">{group.category}</span>

	<!-- 右上：隐私 -->
	<span class="absolute top-2 right-2 flex items-center justify-center rounded-full" style="width:20px;height:20px;background:var(--overlay-strong);">
		{#if group.privacy === 'private'}
			<Lock size={10} color="white" />
		{:else}
			<Globe size={10} color="white" />
		{/if}
	</span>

	<!-- 底部：群名 / 活动 / 成员与加入（半透明深色玻璃背景，有图无图都保证可读） -->
	<div class="absolute inset-x-0 bottom-0 bg-[var(--card-scrim)] px-3 pt-2 pb-2.5" style="backdrop-filter:blur(12px);">
		<p class="line-clamp-1 text-white text-[13px] font-semibold">{group.name}</p>
		<p class="flex items-center gap-1 truncate text-white/80 text-[10px] mt-0.5">
			<CalendarDays size={10} />
			{group.activity}
		</p>
		<div class="mt-2 flex items-center justify-between gap-2">
			<div class="flex items-center gap-1.5 min-w-0">
				<div class="flex -space-x-2 flex-shrink-0">
					{#each group.avatars as avatar, index}
						<img src={avatar} alt="群成员" class="w-5 h-5 rounded-full object-cover" style="border:1px solid rgba(255,255,255,0.45);z-index:{3 - index};" />
					{/each}
				</div>
				<span class="flex items-center gap-1 truncate text-white/80 text-[10px]">
					<Users size={10} />
					{group.members} 人
					<MapPin size={10} class="ml-1" />
					{group.distance}
				</span>
			</div>
			<JoinButton {group} {onToggle} />
		</div>
	</div>
</div>

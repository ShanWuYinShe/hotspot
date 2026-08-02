<script lang="ts">
	import { CalendarDays, Sparkles, Users } from '@lucide/svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import JoinButton from './JoinButton.svelte';
	import type { Group } from './group-data';

	// 附近群组精选卡（移动端全宽）：基于 NearbyImageCard 统一图片优先原语。
	// 精选徽章/群信息/加入按钮覆盖在图片上。
	let { group, onToggle, onOpen }: { group: Group; onToggle: () => void; onOpen: () => void } = $props();
</script>

<NearbyImageCard
	imageSrc={group.banner ?? group.name}
	imageAlt={group.name}
	onClick={onOpen}
	class="rounded-feature min-h-[340px] neu-surface-hero"
>
	<!-- 无封面时占位：覆盖在图片层之上 -->
	{#if !group.banner}
		<div
			class="absolute inset-0 z-[1] flex items-center justify-center"
			style="background:linear-gradient(135deg, color-mix(in srgb, var(--neu-accent-primary) 20%, var(--neu-bg)) 0%, color-mix(in srgb, var(--neu-accent-primary) 6%, var(--neu-bg)) 100%);"
		>
			<span class="text-[var(--neu-accent-primary)] text-4xl font-semibold">{Array.from(group.name)[0] ?? '群'}</span>
		</div>
	{/if}

	<!-- 左上：精选徽章 -->
	<div class="absolute top-3 left-3 z-[2] flex items-center gap-1.5 px-2 py-1 rounded-full" style="background:var(--overlay-strong);backdrop-filter:blur(8px);">
		<Sparkles size={11} color="white" />
		<span class="text-white text-[10px]">本周精选</span>
	</div>

	{#snippet footer()}
		<div class="px-4 pt-2 pb-3">
			<p class="text-[var(--neu-text)] text-lg font-semibold">{group.name}</p>
			<p class="truncate text-[var(--neu-text-muted)] text-[11px] mt-[3px]">{group.desc}</p>
			<div class="mt-2.5 flex items-center justify-between gap-2">
				<div class="flex items-center gap-3 text-[var(--neu-text-muted)] text-[11px] min-w-0">
					<span class="flex items-center gap-1 flex-shrink-0">
						<Users size={12} />
						{group.members}
					</span>
					<span class="flex items-center gap-1 flex-shrink-0 text-[var(--status-online)]">
						<span class="rounded-full" style="width:6px;height:6px;background:var(--status-online);"></span>
						{group.online} 在线
					</span>
					<span class="flex-shrink-0">{group.distance}</span>
					<span class="flex items-center gap-1 truncate text-[var(--neu-accent-primary)]">
						<CalendarDays size={12} />
						{group.activity}
					</span>
				</div>
				<JoinButton {group} {onToggle} />
			</div>
		</div>
	{/snippet}
</NearbyImageCard>

<script module lang="ts">
	export interface GroupCardItem {
		id: string;
		name: string;
		desc: string;
		category: string;
		members: number;
		online: number;
		distance: string;
		activity: string;
		privacy: 'public' | 'private';
		joined: boolean;
		requiresApproval?: boolean;
		banner: string;
		avatars: string[];
	}
</script>

<script lang="ts">
	import { CalendarDays, Globe, Lock, MapPin, Sparkles, Users } from '@lucide/svelte';
	import MediaCard from '@/cst-ui/cards/MediaCard.svelte';
	import Avatar from '@/cst-ui/data-display/Avatar.svelte';
	import JoinButton from '@/cst-ui/buttons/JoinButton.svelte';
	import type { Tier } from '@/lib/bsp-layout';
	import { cn } from '@/lib/utils';

	// 统一群组图片优先卡片（基于 CSTUI MediaCard 原语）
	// - 支持 BSP 四档离散布局：1x1（方格标准）、1x2（高竖）、2x1（宽横）、2x2（巨幕精选）；
	// - featured / 2x2：精选卡（丰富信息与成员头像）；
	// - 2x1：自动横排变体（图左 + 信息右面板）；
	// - 加入/审核按钮统一 JoinButton 原语；
	// - 覆盖态非交互元素 pointer-events-none，点击 100% 直透整卡进入群组详情。
	let {
		group,
		pending = false,
		pendingApproval = false,
		onToggle,
		onOpen,
		featured = false,
		tier = '1x1',
		horizontal = false,
		cardWidth = 0,
		cardHeight = 0,
		priority = false,
		class: className = ''
	}: {
		group: GroupCardItem;
		pending?: boolean;
		pendingApproval?: boolean;
		onToggle: () => void;
		onOpen: () => void;
		featured?: boolean;
		tier?: Tier;
		priority?: boolean;
		horizontal?: boolean;
		cardWidth?: number;
		cardHeight?: number;
		class?: string;
	} = $props();

	const isHorizontal = $derived(
		horizontal || tier === '2x1' || Boolean(cardWidth && cardHeight && cardWidth > cardHeight * 1.25)
	);
	const isHero = $derived(featured || tier === '2x2');
</script>

{#snippet compactFooter({ onImage, horizontal }: { onImage: boolean; horizontal: boolean })}
	{@const onCover = onImage && !horizontal}
	<div class="p-3.5 sm:p-4">
		<p class="line-clamp-1 text-md font-semibold {onCover ? 'text-content-inverse' : 'text-text-primary'}">{group.name}</p>
		<div class="mt-1.5 flex items-center justify-between gap-2 text-xs tabular-nums {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">
			<div class="flex items-center gap-1.5 min-w-0">
				{#if group.avatars.length > 0}
					<div class="flex -space-x-2 shrink-0">
						{#each group.avatars as avatar, index (index)}
							{#if avatar}
								<Avatar src={avatar} name="" size="xs" class={onCover ? 'ring-1 ring-content-inverse opacity-emphasis' : 'border border-border-card-menu'} style="z-index:{3 - index};" />
							{/if}
						{/each}
					</div>
				{/if}
				<span class="flex items-center gap-1 truncate">
					<Users size={11} />
					{group.members} 人
				</span>
			</div>
			<span class="flex items-center gap-1 shrink-0 {onCover ? 'text-content-inverse' : 'text-accent-default'}">
				<MapPin size={11} />
				{group.distance}
			</span>
		</div>
		<div class="mt-2 flex items-center justify-between gap-2">
			{#if group.activity}
				<span class="min-w-0 truncate text-xs {onCover ? 'text-content-inverse opacity-emphasis' : 'text-status-online'}">
					<CalendarDays class="inline mr-1 size-3" />
					{group.activity}
				</span>
			{:else}
				<span></span>
			{/if}
			<JoinButton {group} {pending} {pendingApproval} onMedia={onCover} {onToggle} />
		</div>
	</div>
{/snippet}

<MediaCard
	imageSrc={group.banner}
	imageAlt={group.name}
	{priority}
	onClick={onOpen}
	horizontal={isHorizontal}
	class={cn(
		'h-full w-full rounded-lg shadow-xs',
		className
	)}
>
	{#snippet placeholder()}
		<div class="flex items-center justify-center {isHorizontal ? 'h-full w-full' : isHero ? 'mt-12' : 'mt-8'}">
			<span class="flex items-center justify-center rounded-full bg-accent-soft shadow-xs transition-transform group-hover:scale-105 {isHero ? 'size-22' : 'size-16'}">
				<span class="text-accent-default {isHero ? 'text-4xl' : 'text-3xl'} font-semibold">{Array.from(group.name)[0] ?? '群'}</span>
			</span>
		</div>
	{/snippet}

	{#if isHero}
		<div class="pointer-events-none media-badge absolute top-3 left-3 z-[2] gap-1.5">
			<Sparkles size={11} class="text-content-inverse" />
			<span class="text-content-inverse text-xxs font-medium">本周精选</span>
		</div>
	{:else}
		<span class="pointer-events-none media-badge absolute top-2.5 left-2.5 z-[2]">{group.category}</span>

		<span
			class="pointer-events-none media-badge media-badge--icon absolute top-2 right-2 z-[2]"
			title={group.privacy === 'private' ? '私密群组' : group.requiresApproval ? '加入需管理员审核' : '公开群组'}
		>
			<span class="sr-only">{group.privacy === 'private' ? '私密群组' : group.requiresApproval ? '加入需管理员审核' : '公开群组'}</span>
			{#if group.privacy === 'private'}
				<Lock size={10} color="var(--cst-content-inverse)" aria-hidden="true" />
			{:else}
				<Globe size={10} color="var(--cst-content-inverse)" aria-hidden="true" />
			{/if}
		</span>
	{/if}

	{#snippet footer({ onImage, horizontal }: { onImage: boolean; horizontal: boolean })}
		{@const onCover = onImage && !horizontal}
		{#if featured}
			<div class="flex h-full items-center gap-4 p-4 sm:p-5">
				<div class="min-w-0 flex-1">
					<p class="text-lg font-semibold {onCover ? 'text-content-inverse' : 'text-text-primary'}">{group.name}</p>
					<p class="mt-1 truncate text-sm {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">{group.desc}</p>
					<div class="mt-2.5 flex flex-wrap items-center gap-3 text-sm tabular-nums {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">
						<span class="flex items-center gap-1 shrink-0">
							<Users size={12} />
							{group.members}
						</span>
						<span class="flex items-center gap-1 shrink-0 font-medium {onCover ? 'text-content-inverse' : 'text-status-online'}">
							<span class="rounded-full size-1.5 bg-status-online"></span>
							{group.online} 在线
						</span>
						<span class="shrink-0">{group.distance}</span>
						{#if group.activity}
							<span class="flex min-w-0 items-center gap-1 {onCover ? 'text-content-inverse opacity-emphasis' : 'text-accent-default'}">
								<CalendarDays class="inline size-3 shrink-0" />
								<span class="truncate">{group.activity}</span>
							</span>
						{/if}
					</div>
				</div>
				<JoinButton {group} {pending} {pendingApproval} onMedia={onCover} {onToggle} />
			</div>
		{:else}
			{@render compactFooter({ onImage, horizontal })}
		{/if}
	{/snippet}
</MediaCard>

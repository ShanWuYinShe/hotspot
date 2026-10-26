<script lang="ts">
	import { CalendarDays, Globe, Lock, MapPin, Sparkles, Users } from '@lucide/svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import JoinButton from './JoinButton.svelte';
	import type { Group } from './group-data';
	import { cn } from '@/lib/utils';

	// 附近群组卡片（基于 NearbyImageCard 统一图片优先原语）。
	// Round 2 Image-first Card：整卡图片 + 信息覆盖（无外框、无实底托盘）。
	// - featured：精选卡（桌面横排 图左 40% + 信息右面板，登记例外；移动整卡覆盖式）
	// - 普通卡：竖向整卡覆盖；BSP 横高格（autoLandscape）保留横排变体。
	// 覆盖态文字/按钮统一白字 + 白色描边（onMedia），fallback 用主题色。
	let {
		group,
		pending = false,
		pendingApproval = false,
		onToggle,
		onOpen,
		featured = false,
		horizontal = false,
		cardWidth = 0,
		cardHeight = 0,
		class: className = ''
}: {
		group: Group;
		/** 加入/退出请求进行中（JoinButton 禁点防重复提交） */
		pending?: boolean;
		/** 入群申请待审（JoinButton 显示「审核中 · 点击取消」，点击=取消申请） */
		pendingApproval?: boolean;
		onToggle: () => void;
		onOpen: () => void;
		featured?: boolean;
		/** 横排精选(桌面,图片左 40% + 内容右;r03 精选卡过高吃首屏) */
		horizontal?: boolean;
		/** BSP 格子实际宽度（形态分派判定用） */
		cardWidth?: number;
		/** BSP 格子实际高度 */
		cardHeight?: number;
		class?: string;
} = $props();

	// 高格子自动横排：横格（宽>高）且高过首屏一半（420px）时切
	// 「图左 40% + 信息右面板」，信息保持视线高度。
	const autoLandscape = $derived(
		Boolean(cardWidth && cardHeight && cardWidth > cardHeight && cardHeight > 420)
	);
	</script>

{#snippet compactFooter({ onImage, horizontal }: { onImage: boolean; horizontal: boolean })}
	{@const onCover = onImage && !horizontal}
	<div class="p-4">
		<p class="line-clamp-1 text-sm font-semibold {onCover ? 'text-content-inverse' : 'text-text-primary'}">{group.name}</p>
		<!-- stats 行：头像堆叠 + 成员 / 距离（对齐 Figma GroupCard 行序） -->
		<div class="mt-1 flex items-center justify-between gap-2 text-xxs {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">
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
					<Users size={10} />
					{group.members} 人
				</span>
			</div>
			<span class="flex items-center gap-1 shrink-0 {onCover ? 'text-content-inverse' : 'text-accent-default'}">
				<MapPin size={10} />
				{group.distance}
			</span>
		</div>
		<!-- activity 独立行（对齐 Figma：semantic-online 色 + CalendarDays 图标） -->
		<div class="mt-2 flex items-center justify-between gap-2">
			{#if group.activity}
				<span class="min-w-0 truncate text-xxs {onCover ? 'text-content-inverse opacity-emphasis' : 'text-status-online'}">
					<CalendarDays class="inline mr-1 size-2.5"  />
					{group.activity}
				</span>
			{:else}
				<span></span>
			{/if}
			<JoinButton {group} {pending} {pendingApproval} onMedia={onCover} {onToggle} />
		</div>
	</div>
{/snippet}

	<NearbyImageCard
		imageSrc={group.banner}
		imageAlt={group.name}
		onClick={onOpen}
		horizontal={featured ? horizontal : autoLandscape}
		class={featured
			? (horizontal
				? cn('rounded-2xl min-h-56 shadow-xs hover:shadow-raised-sm', className)
				: cn('min-h-[320px] rounded-lg shadow-xs hover:shadow-raised-sm sm:min-h-64', className))
			: cn(
				'rounded-lg shadow-xs hover:shadow-raised-sm',
				className || 'aspect-[3/4] h-auto'
			)}
	>
		{#snippet watermark()}
			<Users size={72} strokeWidth={1} />
		{/snippet}

		{#snippet placeholder()}
			<span class="text-accent-default {featured ? 'text-4xl' : 'text-3xl'} font-semibold">{Array.from(group.name)[0] ?? '群'}</span>
		{/snippet}
	
		{#if featured}
			<!-- 左上：精选徽章 -->
			<div class="absolute top-3 left-3 z-[2] flex items-center gap-1.5 px-2 py-1 rounded-full bg-surface-overlay-strong" style="backdrop-filter:blur(var(--cst-blur-sm));">
				<Sparkles size={11} color="var(--cst-content-inverse)" />
				<span class="text-content-inverse text-xxs">本周精选</span>
			</div>
		{:else}
			<!-- 左上：分类（毛玻璃底与「本周精选」/距离徽章同语言，亮蓝实底让位核心按钮；Gemini 审查 r03） -->
			<span class="absolute top-2.5 left-2.5 z-[2] inline-flex h-5 items-center justify-center rounded-full bg-surface-overlay-strong px-2.5 text-xxs font-medium leading-none text-content-inverse" style="backdrop-filter:blur(var(--cst-blur-sm));">{group.category}</span>
	
			<!-- 右上：隐私（锁/球图标表意不明，补悬停与读屏文案；Gemini 审查 r03） -->
			<span
				class="absolute top-2 right-2 z-[2] flex items-center justify-center rounded-full size-5 bg-surface-overlay-strong"
				title={group.privacy === 'private' ? '私密群组' : group.requiresApproval ? '加入需管理员审核' : '公开群组'}
				aria-label={group.privacy === 'private' ? '私密群组' : group.requiresApproval ? '加入需管理员审核' : '公开群组'}
			>
				{#if group.privacy === 'private'}
					<Lock size={10} color="var(--cst-content-inverse)" />
				{:else}
					<Globe size={10} color="var(--cst-content-inverse)" />
				{/if}
			</span>
		{/if}
	
		{#snippet footer({ onImage, horizontal }: { onImage: boolean; horizontal: boolean })}
			{@const onCover = onImage && !horizontal}
			{#if featured}
				<div class="p-4 h-full flex flex-col justify-center">
					<p class="text-lg font-semibold {onCover ? 'text-content-inverse' : 'text-text-primary'}">{group.name}</p>
					<p class="mt-1 truncate text-xs {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">{group.desc}</p>
					<div class="mt-3 flex items-center gap-3">
						<div class="min-w-0 flex-1">
							<!-- stats 行：成员 / 在线 / 距离（对齐 Figma FeaturedGroupCard） -->
							<div class="flex items-center gap-3 text-xs {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">
								<span class="flex items-center gap-1 shrink-0">
									<Users size={12} />
									{group.members}
								</span>
								<span class="flex items-center gap-1 shrink-0 font-medium {onCover ? 'text-content-inverse' : 'text-status-online'}">
									<span class="rounded-full size-1.5 bg-status-online"></span>
									{group.online} 在线
								</span>
								<span class="shrink-0">{group.distance}</span>
							</div>
							{#if group.activity}
								<!-- activity 独立 accent 行（对齐 Figma：与 stats 行分离） -->
								<p class="mt-2 truncate text-xs {onCover ? 'text-content-inverse opacity-emphasis' : 'text-accent-default'}">
									<CalendarDays class="inline mr-1 size-3"  />
									{group.activity}
								</p>
							{/if}
						</div>
						<JoinButton {group} {pending} {pendingApproval} onMedia={onCover} {onToggle} />
					</div>
				</div>
			{:else}
				{@render compactFooter({ onImage, horizontal })}
			{/if}
		{/snippet}
	</NearbyImageCard>


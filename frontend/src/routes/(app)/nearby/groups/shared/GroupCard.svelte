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
		priority = false,
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
		/** 首屏前 2 张（R6/R17 策略）：透传 NearbyImageCard 的 eager + fetchpriority=high */
		priority?: boolean;
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
		<!-- 群名是卡内唯一主标题（14px semibold），与 MomentCard 作者/标题档一致 -->
		<p class="line-clamp-1 text-md font-semibold {onCover ? 'text-content-inverse' : 'text-text-primary'}">{group.name}</p>
		<!-- stats 行：头像堆叠 + 成员 / 距离（数字等宽，右对齐距离不因位数抖动） -->
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
		<!-- activity 独立行（对齐 Figma：semantic-online 色 + CalendarDays 图标） -->
		<div class="mt-2 flex items-center justify-between gap-2">
			{#if group.activity}
				<span class="min-w-0 truncate text-xs {onCover ? 'text-content-inverse opacity-emphasis' : 'text-status-online'}">
					<CalendarDays class="inline mr-1 size-3"  />
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
		{priority}
		onClick={onOpen}
		horizontal={featured ? horizontal : autoLandscape}
		class={featured
			? (horizontal
				? cn('rounded-md min-h-48 shadow-xs', className)
				: cn('min-h-[320px] rounded-lg shadow-xs sm:min-h-64', className))
			: cn(
				'rounded-lg shadow-xs',
				className || 'aspect-[3/4] h-auto sm:aspect-[4/5]'
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
			<div class="media-badge absolute top-3 left-3 z-[2] gap-1.5">
				<Sparkles size={11} class="text-content-inverse" />
				<span class="text-content-inverse text-xxs">本周精选</span>
			</div>
		{:else}
			<!-- 左上：分类（毛玻璃底与「本周精选」/距离徽章同语言，亮蓝实底让位核心按钮；Gemini 审查 r03） -->
			<span class="media-badge absolute top-2.5 left-2.5 z-[2]">{group.category}</span>
	
			<!-- 右上：隐私（锁/球图标表意不明，补悬停与读屏文案；Gemini 审查 r03） -->
			<span
				class="media-badge media-badge--icon absolute top-2 right-2 z-[2]"
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
				<!-- 精选卡右面板 = 两栏（文案 flex-1 / CTA auto）：CTA 不再孤悬最右，
				     横向精选卡高度收敛到 192，网格首行信息回到首屏 -->
				<div class="flex h-full items-center gap-4 p-4 sm:p-5">
					<div class="min-w-0 flex-1">
						<p class="text-lg font-semibold {onCover ? 'text-content-inverse' : 'text-text-primary'}">{group.name}</p>
						<p class="mt-1 truncate text-sm {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">{group.desc}</p>
						<!-- stats 行：成员 / 在线 / 距离（对齐 Figma FeaturedGroupCard） -->
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
								<!-- activity 与 stats 同行：精选卡不超过两行信息，避免右侧面板大片留白 -->
								<span class="flex min-w-0 items-center gap-1 {onCover ? 'text-content-inverse opacity-emphasis' : 'text-accent-default'}">
									<CalendarDays class="inline size-3 shrink-0"  />
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
	</NearbyImageCard>

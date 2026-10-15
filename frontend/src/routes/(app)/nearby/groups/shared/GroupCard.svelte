<script lang="ts">
	import { CalendarDays, Globe, Lock, MapPin, Sparkles, Users } from '@lucide/svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import JoinButton from './JoinButton.svelte';
	import type { Group } from './group-data';

	// 附近群组卡片（基于 NearbyImageCard 统一图片优先原语）。
	// featured=false：网格卡（分类/隐私徽章 + 紧凑 footer）；
	// featured=true：移动全宽精选卡（精选徽章 + 大号 footer）。
	// 两种形态共用卡片语言：card-hover 抬升（随 NearbyImageCard 统一）、
	// rounded-md；精选的突出靠全宽大尺寸 + 徽章 + 大号 footer 表达。
	let {
		group,
		pending = false,
		pendingApproval = false,
		onToggle,
		onOpen,
		featured = false,
		horizontal = false
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
	} = $props();
</script>

<NearbyImageCard
	imageSrc={group.banner}
	imageAlt={group.name}
	onClick={onOpen}
	horizontal={featured && horizontal}
	class={featured ? (horizontal ? 'rounded-md min-h-56' : 'rounded-md min-h-85') : 'rounded-md aspect-[3/4]'}
>
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

	{#snippet footer()}
		{#if featured}
			<div class="p-4 h-full flex flex-col justify-center">
				<p class="text-text-primary text-lg font-semibold">{group.name}</p>
				<p class="truncate text-text-secondary text-xs mt-1">{group.desc}</p>
				<div class="mt-3 flex items-center gap-3">
					<div class="min-w-0 flex-1">
						<!-- stats 行：成员 / 在线 / 距离（对齐 Figma FeaturedGroupCard） -->
						<div class="flex items-center gap-3 text-text-secondary text-xs">
							<span class="flex items-center gap-1 shrink-0">
								<Users size={12} />
								{group.members}
							</span>
							<span class="flex items-center gap-1 shrink-0 font-medium text-status-online">
							<span class="rounded-full size-1.5 bg-status-online"></span>
								{group.online} 在线
							</span>
							<span class="shrink-0">{group.distance}</span>
						</div>
						{#if group.activity}
							<!-- activity 独立 accent 行（对齐 Figma：与 stats 行分离） -->
							<p class="mt-2 truncate text-xs text-accent-default">
								<CalendarDays class="inline mr-1 size-3"  />
								{group.activity}
							</p>
						{/if}
					</div>
					<JoinButton {group} {pending} {pendingApproval} {onToggle} />
				</div>
			</div>
		{:else}
			<div class="p-3">
				<p class="line-clamp-1 text-text-primary text-sm font-semibold">{group.name}</p>
				<!-- stats 行：头像堆叠 + 成员 / 距离（对齐 Figma GroupCard 行序） -->
				<div class="mt-1 flex items-center justify-between gap-2 text-text-secondary text-xxs">
					<div class="flex items-center gap-1.5 min-w-0">
						{#if group.avatars.length > 0}
							<div class="flex -space-x-2 shrink-0">
								{#each group.avatars as avatar, index (index)}
									{#if avatar}
										<Avatar src={avatar} name="" size="xs" class="border border-border-card-menu" style="z-index:{3 - index};" />
									{/if}
								{/each}
							</div>
						{/if}
						<span class="flex items-center gap-1 truncate">
							<Users size={10} />
							{group.members} 人
						</span>
					</div>
					<span class="flex items-center gap-1 shrink-0 text-accent-default">
						<MapPin size={10} />
						{group.distance}
					</span>
				</div>
				<!-- activity 独立行（对齐 Figma：semantic-online 色 + CalendarDays 图标） -->
				<div class="mt-2 flex items-center justify-between gap-2">
					{#if group.activity}
						<span class="min-w-0 truncate text-status-online text-xxs">
							<CalendarDays class="inline mr-1 size-2.5"  />
							{group.activity}
						</span>
					{:else}
						<span></span>
					{/if}
					<JoinButton {group} {pending} {pendingApproval} {onToggle} />
				</div>
			</div>
		{/if}
	{/snippet}
</NearbyImageCard>

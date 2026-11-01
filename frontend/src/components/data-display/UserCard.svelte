<script module lang="ts">
	export interface NearbyUser {
		id: string;
		name: string;
		distance: string;
		avatar: string;
		isOnline: boolean;
		tag: string;
		tags?: string[];
		mutual?: number;
		note?: string;
		isFollowing?: boolean;
	}
</script>

<script lang="ts">
	import { MapPin, Sparkles, Users } from '@lucide/svelte';
	import FollowButton from '@/cst-ui/buttons/FollowButton.svelte';
	import type { Tier } from '@/lib/bsp-layout';
	import { cn } from '@/lib/utils';
	import { cardPress } from '@/lib/motion';

	// ═══ 统一用户名片（CSTUI UserCard）═══
	// 全站通用的个人身份展示名片。自适应 1x1、2x1（焦点横排）与 2x2（巨幕精选）：
	// - 2x1：横排焦点卡，左头像 + 中间资料/标签/距离 + 右关注按钮，结构紧凑信息饱满；
	// - 1x1：标准近方卡，顶栏标签/距离 + 居中头像/姓名/签名 + 底栏关注按钮；
	// - 2x2：巨幕精选卡，大尺寸精选勋章 + 大头像 + 丰富个人简介 + 关注按钮；
	// - 零几何位移：点击走全局稳定契约，整卡使用 cardPress 驱动无跳变按压，非交互元素点击直达个人主页。

	let {
		user,
		followed,
		pending = false,
		priority = false,
		tier = '1x1',
		cardWidth = 0,
		cardHeight = 0,
		onFollow,
		onNavigate,
		class: className = ''
	}: {
		user: NearbyUser;
		pending?: boolean;
		priority?: boolean;
		tier?: Tier;
		cardWidth?: number;
		cardHeight?: number;
		followed: boolean;
		onFollow: () => void;
		onNavigate: () => void;
		class?: string;
	} = $props();

	let avatarFailed = $state(false);
	$effect(() => {
		void user.avatar;
		avatarFailed = false;
	});

	const isHorizontal = $derived(tier === '2x1' || (Boolean(cardWidth && cardHeight) && cardWidth > cardHeight * 1.25));
	const isHero = $derived(tier === '2x2');
	const hasAvatar = $derived(Boolean(user.avatar) && !avatarFailed);
</script>

{#if isHorizontal}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<article
		class={cn(
			'card-hover isolate group relative flex h-full w-full flex-col justify-between overflow-hidden rounded-xl border border-border-default/80 bg-surface-card-menu/80 p-3.5 sm:p-4.5 backdrop-blur-md transition-all duration-standard shadow-xs hover:shadow-raised-sm hover:border-accent-default/40',
			className
		)}
		use:cardPress
		onclick={(e) => {
			const target = e.target as HTMLElement | null;
			if (target?.closest('button:not([data-cover]), a[href]')) return;
			onNavigate();
		}}
	>
		<!-- 氛围底纹 -->
		<div class="surface-accent-gradient pointer-events-none absolute inset-0 opacity-20 group-hover:opacity-35 transition-opacity"></div>

		<!-- 顶栏：左推荐徽章，右距离与在线状态 -->
		<div class="relative z-10 flex items-center justify-between gap-2">
			<span class="inline-flex items-center gap-1 rounded-full bg-accent-soft/90 px-2 py-0.5 text-xxs font-semibold text-text-accent-on-soft shadow-xs">
				<Sparkles class="size-2.5 text-accent-default" />
				推荐邻居
			</span>
			<div class="flex items-center gap-2 text-xxs text-text-secondary font-medium">
				{#if user.distance}
					<span class="inline-flex items-center gap-0.5">
						<MapPin class="size-2.5 text-accent-default" />
						{user.distance}
					</span>
				{/if}
				{#if user.isOnline}
					<span class="inline-flex items-center gap-1 text-status-online">
						<span class="size-1.5 rounded-full bg-status-online"></span>
						在线
					</span>
				{/if}
			</div>
		</div>

		<!-- 中部：头像 + 姓名 + 简介 -->
		<div class="relative z-10 flex items-center gap-3.5 my-auto py-1">
			<!-- 头像 -->
			<div class="relative shrink-0 group-hover:scale-105 transition-transform duration-standard">
				<div class="relative flex items-center justify-center size-13 sm:size-15 rounded-full overflow-hidden shadow-xs ring-2 ring-accent-default/30 bg-accent-soft">
					{#if hasAvatar}
						<img
							src={user.avatar}
							alt={user.name}
							loading={priority ? 'eager' : 'lazy'}
							fetchpriority={priority ? 'high' : 'auto'}
							decoding="async"
							class="size-full object-cover"
							onerror={() => (avatarFailed = true)}
						/>
					{:else}
						<span class="text-accent-default font-bold text-xl sm:text-2xl select-none">{Array.from(user.name)[0] ?? '?'}</span>
					{/if}
				</div>
				{#if user.isOnline}
					<span
						class="absolute bottom-0 right-0 size-3 rounded-full bg-status-online ring-2 ring-surface-page"
						title="在线"
					></span>
				{/if}
			</div>

			<!-- 资料文本 -->
			<div class="min-w-0 flex-1">
				<p class="truncate text-base font-bold text-text-primary group-hover:text-accent-default transition-colors">
					{user.name}
				</p>
				<p class="mt-0.5 text-xs text-text-secondary line-clamp-1 leading-relaxed">
					{user.note || '在附近发现精彩生活'}
				</p>
			</div>
		</div>

		<!-- 底栏：标签/共同好友 + 关注按钮 -->
		<div class="relative z-20 flex items-center justify-between gap-2 pt-1 border-t border-border-subtle/30">
			<!-- 标签与共同好友 -->
			<div class="min-w-0 flex-1 flex items-center gap-1.5 overflow-hidden">
				{#if user.tag}
					<span class="rounded bg-surface-subtle px-1.5 py-0.5 text-xxs font-medium text-text-secondary truncate">#{user.tag}</span>
				{:else}
					<span class="rounded bg-surface-subtle px-1.5 py-0.5 text-xxs font-medium text-text-secondary">#活跃邻居</span>
				{/if}
				{#if user.tags && user.tags.length > 1}
					<span class="rounded bg-surface-subtle px-1.5 py-0.5 text-xxs font-medium text-text-secondary truncate">#{user.tags[1]}</span>
				{/if}
				{#if user.mutual}
					<span class="inline-flex items-center gap-0.5 text-xxs text-accent-default font-medium shrink-0 ml-0.5">
						<Users class="size-2.5" />
						{user.mutual}位好友
					</span>
				{/if}
			</div>

			<!-- 关注按钮 -->
			<div class="shrink-0">
				<FollowButton
					size="pill"
					active={followed}
					pending={pending}
					onToggle={onFollow}
					ariaLabel={(followed ? '取消关注 ' : '关注 ') + user.name}
				/>
			</div>
		</div>

		<!-- 点击整卡蒙层 -->
		<button
			type="button"
			class="absolute inset-0 z-10 cursor-pointer focus-ring"
			data-cover
			aria-label={`查看${user.name}的主页`}
			onclick={onNavigate}
		></button>
	</article>
{:else if isHero}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<article
		class={cn(
			'card-hover isolate group relative flex h-full w-full flex-col justify-between overflow-hidden rounded-xl border border-border-default/80 bg-surface-card-menu/80 p-6 backdrop-blur-md transition-all duration-standard shadow-raised-sm hover:shadow-raised-md hover:border-accent-default/50',
			className
		)}
		use:cardPress
		onclick={(e) => {
			const target = e.target as HTMLElement | null;
			if (target?.closest('button:not([data-cover]), a[href]')) return;
			onNavigate();
		}}
	>
		<!-- 氛围底纹 -->
		<div class="surface-accent-gradient pointer-events-none absolute inset-0 opacity-30 group-hover:opacity-45 transition-opacity"></div>

		<!-- 顶栏：徽章与距离 -->
		<div class="relative z-10 flex items-center justify-between gap-2">
			<span class="inline-flex items-center gap-1.5 rounded-full bg-accent-soft/80 px-3 py-1 text-xs font-semibold text-text-accent-on-soft shadow-xs">
				<Sparkles class="size-3 text-accent-default" />
				本周精选邻居
			</span>
			{#if user.distance}
				<span class="inline-flex items-center gap-1 text-xs text-text-secondary bg-surface-subtle/80 px-2.5 py-1 rounded-full">
					<MapPin class="size-3 text-accent-default" />
					{user.distance}
				</span>
			{/if}
		</div>

		<!-- 中部：大头像与主体信息 -->
		<div class="relative z-10 flex flex-col items-center text-center my-auto py-4">
			<div class="relative shrink-0 group-hover:scale-105 transition-transform duration-standard">
				<div class="relative flex items-center justify-center size-22 sm:size-24 rounded-full overflow-hidden shadow-raised-sm ring-2 ring-accent-default/30 bg-accent-soft">
					{#if hasAvatar}
						<img
							src={user.avatar}
							alt={user.name}
							loading={priority ? 'eager' : 'lazy'}
							fetchpriority={priority ? 'high' : 'auto'}
							decoding="async"
							class="size-full object-cover"
							onerror={() => (avatarFailed = true)}
						/>
					{:else}
						<span class="text-accent-default font-bold text-4xl flex items-center justify-center h-full select-none">{Array.from(user.name)[0] ?? '?'}</span>
					{/if}
				</div>
				{#if user.isOnline}
					<span
						class="absolute bottom-1 right-1 size-4 rounded-full bg-status-online ring-2 ring-surface-page shadow-xs"
						title="在线"
					></span>
				{/if}
			</div>

			<p class="mt-3.5 text-xl font-bold text-text-primary group-hover:text-accent-default transition-colors">
				{user.name}
			</p>

			<p class="mt-1 text-sm text-text-secondary line-clamp-2 max-w-sm px-2 leading-relaxed">
				{user.note || '热心探索附近，随时随地开启精彩对话'}
			</p>

			<div class="mt-3 flex flex-wrap justify-center items-center gap-1.5">
				{#if user.tag}
					<span class="rounded-full bg-accent-soft/60 px-2.5 py-0.5 text-xs text-text-accent-on-soft font-medium">#{user.tag}</span>
				{/if}
				{#if user.tags && user.tags.length > 1}
					{#each user.tags.slice(1, 4) as t}
						<span class="rounded-full bg-surface-subtle px-2.5 py-0.5 text-xs text-text-secondary">#{t}</span>
					{/each}
				{/if}
				{#if user.mutual}
					<span class="text-xs text-accent-default font-medium ml-1 flex items-center gap-1">
						<Users class="size-3" />
						{user.mutual}位共同好友
					</span>
				{/if}
			</div>
		</div>

		<!-- 底栏：关注操作 -->
		<div class="relative z-20 flex justify-center pt-2">
			<FollowButton
				size="wide"
				active={followed}
				pending={pending}
				onToggle={onFollow}
				ariaLabel={(followed ? '取消关注 ' : '关注 ') + user.name}
			/>
		</div>

		<!-- 点击整卡蒙层 -->
		<button
			type="button"
			class="absolute inset-0 z-10 cursor-pointer focus-ring"
			data-cover
			aria-label={`查看${user.name}的主页`}
			onclick={onNavigate}
		></button>
	</article>
{:else}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<article
		class={cn(
			'card-hover isolate group relative flex h-full w-full flex-col justify-between overflow-hidden rounded-xl border border-border-default/80 bg-surface-card-menu/80 p-3 sm:p-3.5 backdrop-blur-md transition-all duration-standard shadow-xs hover:shadow-raised-sm hover:border-accent-default/40',
			className
		)}
		use:cardPress
		onclick={(e) => {
			const target = e.target as HTMLElement | null;
			if (target?.closest('button:not([data-cover]), a[href]')) return;
			onNavigate();
		}}
	>
		<!-- 氛围底纹 -->
		<div class="surface-accent-gradient pointer-events-none absolute inset-0 opacity-20 group-hover:opacity-35 transition-opacity"></div>

		<!-- 顶栏：标签与距离 -->
		<div class="relative z-10 flex items-center justify-between gap-1 text-xxs text-text-secondary">
			{#if user.tag}
				<span class="truncate max-w-[55%] rounded bg-surface-subtle px-1.5 py-0.5 font-medium text-text-secondary">
					#{user.tag}
				</span>
			{:else}
				<span class="rounded bg-surface-subtle/80 px-1.5 py-0.5 font-medium text-text-secondary">
					#热点邻居
				</span>
			{/if}
			{#if user.distance}
				<span class="shrink-0 flex items-center gap-0.5 font-medium">
					<MapPin class="size-2 text-accent-default" />
					{user.distance}
				</span>
			{/if}
		</div>

		<!-- 中部：头像与姓名签名 -->
		<div class="relative z-10 flex flex-col items-center text-center my-auto py-1">
			<div class="relative shrink-0 group-hover:scale-105 transition-transform duration-standard">
				<div class="relative flex items-center justify-center size-12 sm:size-13.5 rounded-full overflow-hidden shadow-xs ring-2 ring-accent-default/20 bg-accent-soft">
					{#if hasAvatar}
						<img
							src={user.avatar}
							alt={user.name}
							loading={priority ? 'eager' : 'lazy'}
							fetchpriority={priority ? 'high' : 'auto'}
							decoding="async"
							class="size-full object-cover"
							onerror={() => (avatarFailed = true)}
						/>
					{:else}
						<span class="text-accent-default font-bold text-lg sm:text-xl flex items-center justify-center h-full select-none">{Array.from(user.name)[0] ?? '?'}</span>
					{/if}
				</div>
				{#if user.isOnline}
					<span
						class="absolute bottom-0 right-0 size-2.5 rounded-full bg-status-online ring-2 ring-surface-page"
						title="在线"
					></span>
				{/if}
			</div>

			<p class="mt-2 truncate text-sm font-bold text-text-primary group-hover:text-accent-default transition-colors w-full px-1">
				{user.name}
			</p>

			<p class="mt-0.5 truncate text-xxs text-text-secondary w-full px-1">
				{user.note || '热点邻居'}
			</p>
		</div>

		<!-- 底栏：关注按钮 -->
		<div class="relative z-20 flex justify-center pt-1">
			<FollowButton
				size="pill"
				active={followed}
				pending={pending}
				onToggle={onFollow}
				ariaLabel={(followed ? '取消关注 ' : '关注 ') + user.name}
			/>
		</div>

		<!-- 点击整卡蒙层 -->
		<button
			type="button"
			class="absolute inset-0 z-10 cursor-pointer focus-ring"
			data-cover
			aria-label={`查看${user.name}的主页`}
			onclick={onNavigate}
		></button>
	</article>
{/if}

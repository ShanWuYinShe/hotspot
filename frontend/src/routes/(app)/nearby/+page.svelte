<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, Map as MapIcon, Search } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import HeroStats from '@/components/home/HeroStats.svelte';
	import NearbyHubSections from './shared/NearbyHubSections.svelte';
	import { type NearbyUser } from '@/components/home/nearby-users';
	import type { MomentItem } from '@/api/types';
	import { fetchNearbyFeed } from '@/components/home/nearby-feed';
	import type { NearbyGroup } from '@/components/home/GroupCarousel.svelte';
	import { auth, requireAuth } from '@/stores/auth.svelte';
	import { ROUTES } from '@/lib/routes';
	import { startDefaultLocation, type DefaultLocation } from '@/lib/start-default-location';
	import { LOCATION_ANCHOR_UPDATED_EVENT, readLastLocation } from '@/lib/map-auto-locate';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import Skeleton from '@/components/data-display/Skeleton.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import { createOptimisticFollow } from '@/lib/relationships.svelte';

	// ═══ 「附近」枢纽页（2026-10-19 信息架构调整）═══
	// 背景：此前 /nearby 是 307 跳 /nearby/people（原聚合门户被删，段位导航一并删除），
	// 而首页承载的恰是附近三区块——一级 Tab 的语义（附近这一整个域）比它的落点（只有人）
	// 更宽，首页则事实上成了附近枢纽，两处内容重复。
	// 现调整为三条轴各占一个 Tab（首页=热门算法轴 / 附近=地理轴 / 关注流=关系轴，后者待接入）：
	// 本页承接地理轴，三段分区（人/瞬间/群组）+ Hero 统计，各段「查看全部」进既有叶子页
	// （/nearby/people|moments|groups 保持独立路由与各自筛选，深链不变）。
	// 与旧门户的差别（刻意）：不做搜索框与四入口卡片（搜索/地图已收敛为页头工具入口），
	// 不恢复段位胶囊（各叶子页已自带筛选胶囊，再叠一行会形成双 pill 行）——用分区而非分段。

	let nearbyUsers = $state<NearbyUser[]>([]);
	let moments = $state<MomentItem[]>([]);
	let nearbyGroups = $state<NearbyGroup[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let reloadKey = $state(0);
	let defaultLocation = $state<DefaultLocation | null>(null);
	const device = $derived(platform.platform);

	$effect(() => startDefaultLocation((loc) => {
		defaultLocation = loc ?? null;
	}));

	// 浏览锚点版本号：地图/位置选择更新锚点后经事件触发重拉（与各叶子页同规则）。
	let anchorVersion = $state(0);
	$effect(() => {
		const onChanged = () => { anchorVersion++; };
		window.addEventListener(LOCATION_ANCHOR_UPDATED_EVENT, onChanged);
		return () => window.removeEventListener(LOCATION_ANCHOR_UPDATED_EVENT, onChanged);
	});

	// 位置基准与各叶子页统一（spec api_spec_types.md reference_point = 当前在看哪里）：
	// 浏览锚点优先，回退服务端默认位置。同一数据的多个入口必须同一基准，否则
	// 枢纽统计与叶子页列表会自相矛盾（此前首页与附近页就出现过这类线上问题）。
	function currentReferencePoint(): { latitude: number; longitude: number } | null {
		const anchor = readLastLocation();
		if (anchor) return { latitude: anchor.lat, longitude: anchor.lng };
		return defaultLocation?.point ?? null;
	}

	$effect(() => {
		void reloadKey;
		void anchorVersion;
		let cancelled = false;
		async function fetchData() {
			try {
				loading = true;
				error = null;
				const feed = await fetchNearbyFeed(currentReferencePoint());
				if (cancelled) return;
				moments = feed.moments;
				nearbyUsers = feed.users;
				nearbyGroups = feed.groups;
			} catch (e) {
				if (!cancelled) error = e instanceof Error ? e.message : '加载失败';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchData();
		return () => {
			cancelled = true;
		};
	});

	// 关注基准来自 users/search relation.is_following（NearbyUser.isFollowing）：
	// 已关注的用户显示「已关注」（乐观切换与失败回滚由共享模块承载）。
	const baseFollowed = (id: string) => nearbyUsers.find((u) => u.id === id)?.isFollowing ?? false;
	const followState = createOptimisticFollow();
	// 游客点关注先过登录门控（与 users/[userId]、nearby/people 同一 requireAuth 契约），
	// 登录成功后自动重放；否则匿名请求注定失败，只会得到误导性的「操作失败」toast。
	const toggleFollow = (id: string) => {
		if (!requireAuth({ action: '关注', targetId: id, run: () => void toggleFollow(id) })) return;
		void followState.toggle(id, baseFollowed(id));
	};
	// 登录态排除自己（对齐 nearby 页与鸿蒙端）：自己不该出现在「附近的人」里，
	// 更不能对着自己的行显示「关注」。匿名无身份，全展示。
	const visibleNearbyUsers = $derived.by(() => {
		const selfId = auth.user?.user_id;
		return selfId ? nearbyUsers.filter((u) => u.id !== selfId) : nearbyUsers;
	});
	const onlineCount = $derived(visibleNearbyUsers.filter((u) => u.isOnline).length);
	// 位置名与基准同源（锚点名 → 默认位置名）；anchorVersion 触发重读。
	let locationName = $state('');
	$effect(() => {
		void anchorVersion;
		// 锚点存在（即使无名称）时基准就是锚点，位置名不得回退默认位置名——
		// 否则统计按锚点算、文案却写默认位置名，0 人与文案自相矛盾（线上截图实测）。
		const a = readLastLocation();
		locationName = a?.name ?? (a ? '自定义位置' : defaultLocation?.location_name ?? '');
	});
	// 空态补充说明：与基准同源，让「没有匹配的用户」可解释（对齐叶子页空态）。
	const nearbyFilterHint = $derived(
		locationName ? `正在按「${locationName}」5km 范围筛选` : undefined
	);
	const heroStats = $derived({
		nearby: visibleNearbyUsers.length,
		active: moments.length,
		online: onlineCount
	});
</script>
<svelte:head>
	<title>附近 · HotSpot</title>
</svelte:head>

<!-- 页面根容器：直接承载 HeroStats + 三个带标题分区（人/瞬间/群组），按 RULES §2 例外用
     分区间距 24（gap-6）；页底留白只由 contentClassName pb-6 提供 -->
<div class="flex flex-col flex-1 min-h-full">
<StandardPageLayout contentClassName="animate-fade-in flex flex-col gap-6 pt-4 pb-6 max-w-6xl xl:max-w-7xl 2xl:max-w-[88rem] mx-auto w-full">
	{#snippet header()}
		<PageHeader
			variant="integrated"
			title="附近"
			subtitle={locationName ? `${locationName} · 5km 内` : '5km 内 · 实时更新'}
		>
			{#snippet subtitleIcon()}
				<MapPin class="size-2.5 text-accent-default" strokeWidth={1.75} />
			{/snippet}
			{#snippet trailing()}
				<!-- 搜索/地图是页面级工具，收敛在标题右侧（对齐附近样板页与鸿蒙 NearbyPage） -->
				<div class="flex shrink-0 items-center gap-2">
					<!-- R17：不再放「N 在线」chip —— 与下方 Hero 指标行同屏重复（审计 P2-6）。
					     在线数由 Hero MetricRow 单一表达，页头只保留搜索/地图工具入口。 -->
					<!-- integrated 页头的 trailing 图标动作用 ghost（与 nearby 三叶一致）：
					     页头无承载面，圆钮会重新变成悬浮控件块 -->
					<ActionIconButton variant="ghost" icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} size={36} class={device === 'desktop' ? 'hidden' : ''} />
					<ActionIconButton variant="ghost" icon={MapIcon} label="地图浏览" onClick={() => void goto(ROUTES.map)} size={36} />
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if loading}
		<!-- 首屏骨架屏：Hero 统计卡 + 三段分区（人/瞬间/群组）的预览卡行；
		     与真实枢纽页纵向节奏一致（gap-6 = 分区块间距 24） -->
		<div class="flex flex-col gap-6" role="status" aria-busy="true" aria-label="加载附近数据…" data-skeleton-root>
			<Skeleton class="h-36 w-full rounded-lg" />
			{#each [0, 1, 2] as section (section)}
				<div class="flex flex-col gap-3">
					<div class="flex items-center justify-between">
						<Skeleton class="h-4 w-24 rounded-sm" />
						<Skeleton class="h-4 w-12 rounded-sm" />
					</div>
					<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
						<Skeleton class="aspect-[3/4] w-full rounded-lg" />
						<Skeleton class="aspect-[3/4] w-full rounded-lg" />
						<Skeleton class="hidden aspect-[3/4] w-full rounded-lg sm:block" />
						<Skeleton class="hidden aspect-[3/4] w-full rounded-lg sm:block" />
					</div>
				</div>
			{/each}
		</div>
	{:else if error}
		<LoadError message={error} onRetry={() => reloadKey++} />
	{:else if moments.length === 0 && visibleNearbyUsers.length === 0 && nearbyGroups.length === 0}
		<!-- 页面级空态走新版 EmptyState：72px 图标软底 + spotlight（不再用裸图标 + 灰字） -->
		<EmptyState
			spotlight
			iconTone="accent"
			title="附近还没有内容"
			description="去发布第一条瞬间，或换个位置看看。"
		>
			{#snippet icon()}
				<MapPin class="size-7" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else}
		<HeroStats onExplore={() => void goto(ROUTES.nearbyPeople)} stats={heroStats} locationName={locationName} />
		<NearbyHubSections
			users={visibleNearbyUsers}
			{moments}
			groups={nearbyGroups}
			emptyHint={nearbyFilterHint}
			isFollowed={(id) => followState.isFollowed(id, baseFollowed(id))}
			isPending={(id) => followState.isPending(id)}
			onToggleFollow={toggleFollow}
			onSeeAllUsers={() => void goto(ROUTES.nearbyPeople)}
			onSeeAllMoments={() => void goto(ROUTES.nearbyMoments)}
			onSeeAllGroups={() => void goto(ROUTES.nearbyGroups)}
			onOpenMoment={(id) => void goto(ROUTES.moment(id))}
			onOpenUser={(id) => void goto(ROUTES.user(id))}
		/>
	{/if}
</StandardPageLayout>
</div>
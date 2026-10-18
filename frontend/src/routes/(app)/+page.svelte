<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, Search, Radar, Map } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import HeroStats from '@/components/home/HeroStats.svelte';
	import { type NearbyUser } from '@/components/home/nearby-users';
	import type { MomentItem } from '@/api/types';
	import { fetchHomeFeed } from '@/components/home/home-feed';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ActivityRow from '@/components/home/ActivityRow.svelte';
	import NearbyUserRows from '@/components/home/NearbyUserRows.svelte';
	import { auth, requireAuth } from '@/stores/auth.svelte';
	import GroupCarousel, { type NearbyGroup } from '@/components/home/GroupCarousel.svelte';
	import { ROUTES } from '@/lib/routes';
	import { startDefaultLocation, type DefaultLocation } from '@/lib/start-default-location';
	import { LOCATION_ANCHOR_UPDATED_EVENT, readLastLocation } from '@/lib/map-auto-locate';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
import { createOptimisticFollow } from '@/lib/relationships.svelte';

	// 首页「附近」：Hero 统计 + 附近动态 + 群组轮播 + 推荐列表。
	// 三端统一 PageHeader（title + 额外信息 + trailing 工具区）+ 内容区：
	// 桌面/平板：左侧 340px 摘要栏 + 右侧完整用户列表（2 栏 grid）。
	// 移动：单栏堆叠。

	// ── 数据获取（Promise.all + cancelled 竞态）────────
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

	// 浏览锚点版本号：地图/位置选择更新锚点后经事件触发重拉（与附近页同规则）。
	let anchorVersion = $state(0);
	$effect(() => {
		const onChanged = () => { anchorVersion++; };
		window.addEventListener(LOCATION_ANCHOR_UPDATED_EVENT, onChanged);
		return () => window.removeEventListener(LOCATION_ANCHOR_UPDATED_EVENT, onChanged);
	});

	// 位置基准与附近页统一（spec api_spec_types.md reference_point = 当前在看哪里）：
	// 浏览锚点优先，回退服务端默认位置。此前首页只认默认位置、附近页锚点优先，
	// 用户设置自定义锚点后首页「5 人附近」与附近页「没有匹配的用户」自相矛盾
	//（线上实测）。同一数据的两个入口必须同一基准。
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
				const feed = await fetchHomeFeed(currentReferencePoint());
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
	let heroLocationName = $state('');
	$effect(() => {
		void anchorVersion;
		// 锚点存在（即使无名称）时基准就是锚点，位置名不得回退默认位置名——
		// 否则统计按锚点算、文案却写「南天门」，0 人与文案自相矛盾（线上截图实测）。
		const a = readLastLocation();
		heroLocationName = a?.name ?? (a ? '自定义位置' : defaultLocation?.location_name ?? '');
	});
	// 空态补充说明：与基准同源，让「没有匹配的用户」可解释（对齐附近页空态）。
	const nearbyFilterHint = $derived(
		heroLocationName ? `正在按「${heroLocationName}」5km 范围筛选` : undefined
	);
	const heroStats = $derived({
		nearby: visibleNearbyUsers.length,
		active: moments.length,
		online: onlineCount
	});

</script>
<svelte:head>
	<title>首页 · HotSpot</title>
</svelte:head>


{#snippet momentRows(items: MomentItem[], onClick: (momentId: string) => void)}
	<ListContainer bordered>
		{#each items as moment, i (moment.moment_id)}
			<ActivityRow {moment} index={i} onClick={() => onClick(moment.moment_id)} />
		{/each}
	</ListContainer>
{/snippet}

{#snippet heroColumn(momentItems: MomentItem[], withMoreAction: boolean, showGroups = true)}
	<!-- Hero 统计 + 附近活动 + 群组轮播：移动/桌面双布局共用，
	     活动列表条数与「更多」入口经参数区分；桌面把群组轮播挪到右栏底部
	     平衡双栏重心（第二轮审查 r90），故 showGroups 由桌面传入 false -->
	<HeroStats onExplore={() => void goto(ROUTES.nearbyPeople)} stats={heroStats} locationName={heroLocationName} />
	<!-- 大纲页密度：无内容时整段隐藏（标题+空框），专区经导航/更多入口可达 -->
	{#if momentItems.length > 0}
		<div>
			<SectionHead title="附近活动" action={withMoreAction ? '查看全部' : undefined} onAction={() => void goto(ROUTES.nearbyMoments)} />
			{@render momentRows(momentItems, (id) => void goto(ROUTES.moment(id)))}
		</div>
	{/if}
	{#if showGroups}
		<GroupCarousel groups={nearbyGroups} onSeeAll={() => void goto(ROUTES.nearbyGroups)} />
	{/if}
{/snippet}

{#snippet homeContent()}
	<!-- 移动端成功态内容（loading/error/empty 三态由外层统一承载，只写一次）。
	     首页为大纲页：列表限长 + 「更多」链接导航到专区 -->
	{@render heroColumn(moments.slice(0, 3), true)}
	{#if visibleNearbyUsers.length > 0}
	<div>
		<SectionHead title="人物推荐" action="查看全部" onAction={() => void goto(ROUTES.nearbyPeople)} />
		<!-- 对齐 Figma HomePage：人物推荐仅展示前 4 位，其余引导到附近的人专区 -->
		<NearbyUserRows
			users={visibleNearbyUsers.slice(0, 4)}
			isFollowed={(id) => followState.isFollowed(id, baseFollowed(id))}
			isPending={(id) => followState.isPending(id)}
			onToggle={toggleFollow}
			emptyHint={nearbyFilterHint}
		/>
	</div>
	{/if}
{/snippet}

	<StandardPageLayout
		contentClassName={platform.platform === 'desktop' ? 'animate-fade-in pt-4 pb-8' : 'animate-fade-in flex flex-col gap-5 pt-4 pb-16'}
	>
	{#snippet header()}
		<!-- ═══ 首页 — 统一 PageHeader（三端同标题；附近内容拆分到 /nearby 聚合页）═══ -->
		<PageHeader title="首页" titleSize={device === 'desktop' ? 'xl' : '2xl'} subtitle="发现周边有趣的人和事" sticky>
			{#snippet trailing()}
				<!-- trailing 簇锁定不收缩：chip 单行不折行（whitespace-nowrap + leading-none，
				     徽标排版契约），窄屏下由标题副标题先截断让位，操作钮与在线计数始终完整。 -->
				<div class="flex shrink-0 items-center gap-2">
					{#if onlineCount > 0}
						<div class="inline-flex shrink-0 items-center justify-center gap-1.5 rounded-full border border-border-default bg-surface-primary px-3 py-1.5 leading-none whitespace-nowrap shadow-raised-sm">
							<span class="size-1.5 shrink-0 rounded-full bg-status-online"></span>
							<span class="text-xs whitespace-nowrap text-text-secondary tabular-nums">{onlineCount} 在线</span>
						</div>
					{/if}
					{#if platform.platform !== 'desktop'}
						<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} variant="ghost" size={36} />
						<ActionIconButton icon={Bell} label="通知" onClick={() => void goto(ROUTES.notify)} variant="ghost" size={36} />
					{/if}
					<ActionIconButton icon={Map} label="位置选择" onClick={() => void goto(ROUTES.map)} variant="ghost" size={36} />
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if loading}
		<!-- 加载中：与移动端一致的统一 spinner（桌面/平板不再静默空渲染） -->
		<InlineLoading label="加载首页数据…" />
	{:else if error}
		<LoadError message={error} onRetry={() => reloadKey++} />
	{:else if moments.length === 0 && visibleNearbyUsers.length === 0 && nearbyGroups.length === 0}
		<EmptyState title="附近还没有内容" description="去发布第一条动态，或到附近的人认识新朋友。">
			{#snippet icon()}
				<Radar class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else if platform.platform === 'desktop'}
		<!-- Desktop：左摘要栏（340px fixed）+ 右用户列表（两栏 grid）
		     平板（600-1279）不进入此分支：固定的 340px 在 ≤880vp 时会把右列压到
		     ≤500px、600-700vp 时压到 ≤260px，使 NearbyUserRows 不可读。平板直接走
		     homeContent（单栏堆叠，4 条人物推荐 + 查看全部），与移动端保持同款
		     内容密度。 -->
		<!-- min-h-full：让双列行至少撑满正文列（flex-1 已撑满视口剩余），左列
			 border-r 分割线随之贯穿到页底，短内容时不再像中途截断。 -->
		<div class="grid min-h-full grid-cols-[21.25rem_minmax(0,1fr)] gap-6">
			<!-- Left: hero + activity + groups (340px)，与移动端共用 heroColumn。
		     活动限长 6 条 + 「查看全部」：全量渲染时左栏比右栏（用户网格+群组轮播）
		     长出约一屏（审查 P1-4 残留），限长后双栏重心平衡，溢出内容由入口承接。 -->
			<div class="flex flex-col gap-6 border-r border-border-default pr-6">
				{@render heroColumn(moments.slice(0, 6), true, false)}
			</div>

			<!-- Right: full nearby-users list（标题统一 SectionHead 语言,r01:与「附近活动」字号一致;
			     双列卡片网格消除单列行过宽与底部空旷） -->
			<div>
				<div class="mb-5">
					<SectionHead
						title="附近的人"
						subtitle={heroLocationName ? `${heroLocationName} · 5km 内` : '5km 内 · 实时更新'}
						action="查看全部"
						onAction={() => void goto(ROUTES.nearbyPeople)}
					/>
				</div>

				<NearbyUserRows
					users={visibleNearbyUsers}
					isFollowed={(id) => followState.isFollowed(id, baseFollowed(id))}
					isPending={(id) => followState.isPending(id)}
					onToggle={toggleFollow}
					emptyHint={nearbyFilterHint}
					columns={2}
				/>
				<!-- 右栏底部群组轮播：平衡双栏重心（第二轮审查 r90），移动/平板仍走左栏 -->
				<div class="mt-6">
					<GroupCarousel groups={nearbyGroups} onSeeAll={() => void goto(ROUTES.nearbyGroups)} />
				</div>
			</div>
		</div>
	{:else}
		{@render homeContent()}
	{/if}
</StandardPageLayout>

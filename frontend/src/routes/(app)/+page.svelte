<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Bell, Search, Compass, ArrowRight, Sparkles, TrendingUp, Users, LogIn, MapPin, Radio, ChevronRight } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import { homeFeed, type HomeFeedTab } from '@/stores/home-feed.svelte';
	import { formatDistance } from '@/lib/utils';
	import type { MomentItem } from '@/api/types';
	import { searchMoments, fetchFollowingMoments } from '@/api';
	import HomeFeedItem from '@/components/home/HomeFeedItem.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { ROUTES } from '@/lib/routes';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Button from '@/components/ui/button.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import RailMomentThumb from '@/components/rail/RailMomentThumb.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import AdaptiveCardGrid from '@/components/layout/AdaptiveCardGrid.svelte';
	import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import { contentFade } from '@/lib/motion';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import imgHotSpot from '@/assets/hotspot-logo.png';

	// ═══ 首页 = 社交动态流（为你推荐 · 正在关注）═══
	// 采用成熟现代社区（Twitter/Threads/即刻）的社交动态流设计语言：
	// 左侧作者头像 + 右侧完整社交实体（昵称/时间/位置/正文/图集/标签/互动栏）。
	// 桌面端采用 SplitPageLayout（640px 黄金阅读流 + 右侧活跃邻居/热议话题）。
	// 移动端全宽平铺，顶栏品牌字标 + 分段切换，纯白极简通透。

	const PAGE_SIZE = 20;
	const device = $derived(platform.platform);

	type FeedTab = HomeFeedTab;
	// 桌面端分段器在顶栏灵动岛、移动端在页内品牌头，两处共用同一份状态。
	const activeTab = $derived(homeFeed.activeTab);
	let booting = $state(true);

	onMount(() => {
		// 回到首页时重置为「为你推荐」，与改造前页内组件卸载重建的初始行为一致。
		homeFeed.reset();
	});

	// 桌面端：内容区分段器滚出视口后交接给顶栏灵动岛；回到顶部再交还。
	let desktopTabsEl = $state<HTMLElement | null>(null);
	$effect(() => {
		if (device !== 'desktop') {
			homeFeed.setIslandVisible(false);
			return;
		}
		const el = desktopTabsEl;
		if (!el) return;
		const rootMargin = `-${Math.round(52 * Math.max(1, platform.uiScale))}px 0px 0px 0px`;
		const io = new IntersectionObserver(
			([entry]) => homeFeed.setIslandVisible(!entry.isIntersecting),
			{ rootMargin, threshold: 0 }
		);
		io.observe(el);
		return () => {
			io.disconnect();
			homeFeed.setIslandVisible(false);
		};
	});

	function createMomentPager(
		fetchPage: (cursor: string | null) => Promise<{ items: MomentItem[]; nextCursor: string | null }>
	) {
		let items = $state<MomentItem[]>([]);
		let loading = $state(false);
		let loadingMore = $state(false);
		let error = $state<string | null>(null);
		let nextCursor = $state<string | null>(null);
		let loaded = $state(false);

		async function load() {
			if (loading) return;
			loading = true;
			error = null;
			try {
				const page = await fetchPage(null);
				items = page.items;
				nextCursor = page.nextCursor;
				loaded = true;
			} catch (e) {
				error = e instanceof Error ? e.message : '加载失败';
			} finally {
				loading = false;
			}
		}

		async function loadMore() {
			if (loadingMore || !nextCursor) return;
			loadingMore = true;
			try {
				const page = await fetchPage(nextCursor);
				const seen = new Set(items.map((m) => m.moment_id));
				items = [...items, ...page.items.filter((m) => !seen.has(m.moment_id))];
				nextCursor = page.nextCursor;
			} catch (e) {
				error = e instanceof Error ? e.message : '加载失败';
			} finally {
				loadingMore = false;
			}
		}

		function reset() {
			items = [];
			nextCursor = null;
			loaded = false;
			error = null;
		}

		return {
			get items() { return items; },
			get loading() { return loading; },
			get loadingMore() { return loadingMore; },
			get error() { return error; },
			get loaded() { return loaded; },
			get hasMore() { return Boolean(nextCursor); },
			load,
			loadMore,
			reset
		};
	}

	const paged = (res: { code: number; message?: string; data?: { items?: MomentItem[]; next_cursor?: string | null } | null }) => {
		if (res.code !== 0) throw new Error(res.message || '加载失败');
		return { items: res.data?.items ?? [], nextCursor: res.data?.next_cursor ?? null };
	};

	const discover = createMomentPager(async (cursor) =>
		paged(await searchMoments({ sort_by: 'popular', pagination: { limit: PAGE_SIZE, ...(cursor ? { cursor } : {}) } }))
	);

	const following = createMomentPager(async (cursor) =>
		paged(await fetchFollowingMoments({ pagination: { limit: PAGE_SIZE, ...(cursor ? { cursor } : {}) } }))
	);

	// 发现流：全局预加载（每次挂载只自动拉一次）。
	//
	// 必须带 !error 守卫：本 effect 依赖 loaded/loading 两个 rune，而 load() 无论
	// 成功失败都会把 loading 置回 false。失败时 loaded 仍为 false，若只判
	// 「!loaded && !loading」，load 一结束 effect 就再次触发 → 无限重试风暴
	// （实测接口不可用时 8 秒内打出 165 个 POST，弱网下会把本来就不稳的链路彻底
	// 打满）。失败态由下方 LoadError 的手动「重试」接管（load() 开头会清 error）。
	$effect(() => {
		if (!discover.loaded && !discover.loading && !discover.error) {
			void discover.load().finally(() => {
				booting = false;
			});
		}
	});

	// 关注流：登录用户加载（同上的 !error 守卫，失败不自动重试）
	$effect(() => {
		if (auth.status === 'authenticated') {
			if (!following.loaded && !following.loading && !following.error) {
				void following.load();
			}
		}
	});

	// 当前激活 Tab 对应的数据与加载状态
	const currentPager = $derived(activeTab === 'discover' ? discover : following);
	const currentItems = $derived(currentPager.items);
	const demoMode = $derived(page.url.searchParams.get('demo'));
	const currentLoading = $derived(booting || (currentItems.length === 0 && currentPager.loading));
	const currentError = $derived(currentPager.error);
	const hasMore = $derived(currentPager.hasMore);
	const loadingMore = $derived(currentPager.loadingMore);

	// 从发现流中智能提取活跃邻居（去重展示至多 3 位）
	const activeNeighbors = $derived.by(() => {
		const seen = new Set<string>();
		const list: { userId: string; nickname: string; avatarUrl: string | null; location?: string }[] = [];
		for (const m of discover.items) {
			if (m.user?.user_id && !seen.has(m.user.user_id) && m.user.user_id !== auth.user?.user_id) {
				seen.add(m.user.user_id);
				list.push({
					userId: m.user.user_id,
					nickname: m.user.nickname,
					avatarUrl: m.user.avatar_url,
					location: m.location?.location_name ?? undefined
				});
				if (list.length >= 3) break;
			}
		}
		return list;
	});

	// 右 rail「附近活动」：复用已加载的发现流里带位置信息的瞬间（零额外请求），
	// 用位置 + 距离作为地理轴信息（主列是算法轴，rail 补一条地理轴线索）。
	const railNearbyMoments = $derived.by(() =>
		discover.items
			.filter((m) => m.moment_id && m.location?.location_name)
			.slice(0, 3)
			.map((m) => ({
				id: m.moment_id,
				image: m.images?.[0]?.url ?? '',
				title: m.description?.trim() || '无标题瞬间',
				meta: [m.location?.location_name, formatDistance(m.location?.distance)].filter(Boolean).join(' · ')
			}))
	);

	// 社区精选热议话题（带瞬间计数与搜索入口）
	const trendingTopics = [
		{ tag: '周末去哪儿', count: 32, desc: '周末好去处灵感分享' },
		{ tag: '老街咖啡香', count: 24, desc: '寻访城市角落的特调' },
		{ tag: '云龙湖夜跑', count: 18, desc: '一起出发，用脚步丈量夜色' },
		{ tag: '徐州历史漫步', count: 15, desc: '古城砖瓦间的岁月温度' },
		{ tag: '宝藏小店', count: 11, desc: '街头巷尾的惊喜发现' }
	];
</script>

<svelte:head>
	<title>首页 · HotSpot</title>
</svelte:head>

{#snippet desktopFeedTabs()}
	<!-- 桌面端信息流分段器：随内容滚动；滚出视口后由顶栏灵动岛接手（同一状态、无缝切换） -->
	<div bind:this={desktopTabsEl} class="flex justify-center pb-2 pt-1 pointer-events-none">
		<div class="pointer-events-auto rounded-full bg-surface-primary/90 p-0.5 shadow-raised-sm backdrop-blur-md">
			<TabBar
				variant="segmented"
				size="md"
				items={[
					{ id: 'discover', label: '为你推荐' },
					{ id: 'following', label: '正在关注' }
				]}
				value={activeTab}
				onChange={(id) => homeFeed.setTab(id as FeedTab)}
				class="min-w-[15rem] bg-transparent ring-0"
			/>
		</div>
	</div>
{/snippet}

{#snippet feedItems()}
	{#each currentItems as moment, index (moment.moment_id)}
		<HomeFeedItem
			{moment}
			showDistance={false}
			priority={index < 2}
			onClick={() => void goto(ROUTES.moment(moment.moment_id))}
		/>
	{/each}
{/snippet}

{#snippet nearbyBanner()}
	<button
		type="button"
		class="hover-btn group flex w-full items-center gap-3.5 rounded-lg bg-surface-island px-4 py-4 text-left focus-ring transition hover:shadow-raised-sm shadow-xs"
		onclick={() => void goto(ROUTES.nearby)}
	>
		<span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft transition-transform group-hover:scale-105">
			<Compass class="size-5" strokeWidth={1.75} />
		</span>
		<div class="min-w-0 flex-1">
			<div class="flex items-center gap-1.5">
				<span class="text-sm font-semibold text-text-primary">看看附近正在发生什么</span>
			</div>
			<p class="truncate text-xs text-text-secondary mt-0.5">附近的人 · 群组 · 瞬间，5km 内实时雷达</p>
		</div>
		<ArrowRight class="size-4 shrink-0 text-text-secondary transition-transform group-hover:translate-x-0.5 group-hover:text-accent-default" strokeWidth={1.75} />
	</button>
{/snippet}

<div class="flex flex-col flex-1 min-h-full">
	<SplitPageLayout
		contentClassName="animate-fade-in flex flex-col pt-0 pb-8"
	>
		{#snippet header()}
			{#if device !== 'desktop'}
				<!-- 移动端顶栏：品牌 Logo + 快捷动作 + 居中胶囊分段器一体化（告别生硬下划线与单调布局） -->
				<!-- 去硬边线（Round 2）：画布色半透明 + 模糊承担分隔，不再画 border -->
				<div class="bg-surface-page/80 backdrop-blur-md sticky top-0 z-20 pb-2">
					<div class="flex items-center justify-between px-4 pt-3 pb-2">
						<div class="flex items-center gap-2">
							<img src={imgHotSpot} decoding="async" alt="HotSpot" class="size-6.5 rounded-full object-cover shadow-xs" />
							<span class="text-md font-bold tracking-tight text-text-primary">HotSpot</span>
						</div>
						<h1 class="sr-only">首页</h1>
						<div class="flex items-center gap-1">
							<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} variant="ghost" size={36} />
							<ActionIconButton icon={Bell} label="通知" onClick={() => void goto(ROUTES.notify)} variant="ghost" size={36} />
						</div>
					</div>
					<div class="flex justify-center px-4">
						<TabBar
							variant="segmented"
							size="sm"
							items={[
								{ id: 'discover', label: '为你推荐' },
								{ id: 'following', label: '正在关注' }
							]}
							value={activeTab}
							onChange={(id) => homeFeed.setTab(id as FeedTab)}
							class="w-full max-w-[13.75rem]"
						/>
					</div>
				</div>
		{:else}
			<!-- 桌面端无移动品牌头，仍需每页唯一 h1（axe page-has-heading-one）；
			     视觉由顶栏品牌与「为你推荐」Tab 承担，标题仅对读屏可见 -->
			<h1 class="sr-only">首页</h1>
		{/if}
		{/snippet}

		{#if device === 'desktop'}
			{@render desktopFeedTabs()}
		{/if}

		<!-- 关注流未登录态：温馨引导登录卡片 -->
		<div use:contentFade={currentLoading} class="min-h-[50vh] flex flex-col justify-between">
		{#if activeTab === 'following' && auth.status !== 'authenticated'}
			<div class="mx-4 my-8 rounded-lg bg-surface-island p-6 text-center shadow-xs">
				<div class="mx-auto flex size-12 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft mb-3">
					<Users class="size-6" strokeWidth={1.75} />
				</div>
				<h2 class="text-md font-semibold text-text-primary">登录后查看关注动态</h2>
				<p class="mt-1 text-xs text-text-secondary max-w-sm mx-auto">
					关注同频的好友与活跃邻居，即时获取 TA 们的最新瞬间与生活记录。
				</p>
				<Button type="button" size="md" class="mt-4" onclick={() => void goto(ROUTES.login)}>
					<LogIn class="size-3.5" />
					<span>立即登录</span>
				</Button>
			</div>
		{:else if currentLoading}
			<!-- 首屏骨架屏：卡壳与 HomeFeedItem 一致（骨架→内容零位移），栅格同真实内容 -->
			<SkeletonFeed
				variant="feed"
				count={4}
				label="加载动态中…"
				class="grid grid-cols-2 gap-2.5 px-2.5 pt-2 sm:grid-cols-3 sm:gap-4 sm:px-0 lg:grid-cols-[repeat(auto-fill,minmax(20rem,1fr))]"
			/>
		{:else if currentError && currentItems.length === 0}
			<LoadError message={currentError} onRetry={() => void currentPager.load()} />
		{:else if currentItems.length > 0}
			<!-- 高质感社区 Feed 流：单条居中陈列避免半屏塌陷，多条移动端规整 2 列满铺提效；
			     md 起按 340px 最小轨自动增列，避免 2K/4K 下固定 3 列把卡片拉成 600px+ 巨卡 -->
			{#if currentItems.length === 1}
				<div class="mx-auto w-full max-w-md px-2.5 pt-2 sm:px-0">
					{@render feedItems()}
				</div>
			{:else if demoMode === 'a' || demoMode === 'b'}
				<div class="flex flex-col gap-3.5 px-3 pt-2 sm:gap-4 sm:px-0">
					{@render feedItems()}
				</div>
			{:else if demoMode === 'c' || demoMode === 'd'}
				<div class="grid grid-cols-1 gap-4 px-3 pt-2 2xl:grid-cols-2 sm:px-0">
					{@render feedItems()}
				</div>
			{:else}
				<!-- 21.25rem(340) 在 1024–1279（rail 已收起、主列约 688px）只够 1 列，
				     会把单卡拉成 688×860 巨卡；20rem(320) 让该区间回到 2 列 336（≈阅读列宽），
				     1440/1920 的列数与卡宽不变。 -->
				<AdaptiveCardGrid
					count={currentItems.length}
					minTrackRem={20}
					gapRem={[0.625, 1, 1]}
					class="px-2.5 pt-2 sm:px-0"
				>
					{@render feedItems()}
				</AdaptiveCardGrid>
			{/if}

			{#if hasMore}
				<div class="mt-5 flex justify-center px-4">
					<LoadMoreButton loading={loadingMore} onLoad={() => void currentPager.loadMore()} />
				</div>
			{:else}
				<ListEnd />
			{/if}

			{#if currentItems.length <= 2 && !hasMore}
				<!-- 数据稀疏时（<=2条）提供探索热门话题卡片，充实版面、引导互动，消除大面积死白与断崖空旷感 -->
				<div class="px-3.5 sm:px-0 my-3">
					<div class="rounded-lg bg-surface-island p-4 shadow-xs">
						<div class="flex items-center justify-between mb-2.5">
							<div class="flex items-center gap-1.5">
								<Sparkles class="size-4 text-accent-default" strokeWidth={1.75} />
								<span class="text-xs font-semibold text-text-primary">探索更多感兴趣的话题</span>
							</div>
							<span class="text-xxs text-text-secondary">社区热议</span>
						</div>
						<div class="flex flex-wrap gap-1.5">
							{#each trendingTopics.slice(0, 4) as topic (topic.tag)}
								<TagPill
									size="sm"
									onClick={() => void goto(`${ROUTES.search}?q=${encodeURIComponent('#' + topic.tag)}`)}
								>
									<span>#{topic.tag}</span>
									<span class="text-xxs opacity-70">({topic.count})</span>
								</TagPill>
							{/each}
						</div>
					</div>
				</div>
			{/if}

			<div class="px-3.5 sm:px-0">
				{@render nearbyBanner()}
			</div>
		{:else}
			<EmptyState
				title={activeTab === 'following' ? '暂无关注动态' : '还没有动态'}
				description={activeTab === 'following' ? '去「为你推荐」发现更多有趣的人，或去附近看看身边正在发生什么。' : '去附近看看，或者发布属于你的第一条瞬间。'}
				bare
				spotlight
			>
				{#snippet icon()}
					<Sparkles class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
				{/snippet}
			</EmptyState>

			<div class="px-3.5 sm:px-0">
				{@render nearbyBanner()}
			</div>
		{/if}
		</div>

		{#snippet rail()}
			{#if demoMode === 'd'}
				<!-- 方案 D：LBS 双核全景大屏的右翼实时雷达控制中心 -->
				<div class="flex flex-col gap-4 py-2">
					<!-- 实时雷达探索卡片 -->
					<div class="relative overflow-hidden rounded-lg border border-border-default bg-surface-primary shadow-raised-sm">
						<!-- 地图雷达视窗背景模拟 -->
						<div class="relative h-60 w-full bg-surface-subtle flex items-center justify-center overflow-hidden border-b border-border-subtle">
							<!-- 仿雷达脉冲扫描同心圆 -->
							<div class="absolute inset-0 flex items-center justify-center pointer-events-none opacity-subtle">
								<div class="size-28 rounded-full border border-accent-default/50 animate-ping"></div>
								<div class="absolute size-44 rounded-full border border-border-default/60"></div>
								<div class="absolute size-64 rounded-full border border-border-default/40"></div>
								<div class="absolute size-84 rounded-full border border-border-default/20"></div>
							</div>
							<!-- 模拟地标图钉 -->
							<div class="absolute top-10 left-10 flex items-center gap-1.5 rounded-full bg-surface-primary/95 backdrop-blur-sm px-3 py-1 shadow-raised-sm border border-border-subtle text-xs font-medium text-text-primary">
								<span class="size-2 rounded-full bg-status-online"></span> 云龙湖 · 8条动态
							</div>
							<div class="absolute bottom-12 right-10 flex items-center gap-1.5 rounded-full bg-surface-primary/95 backdrop-blur-sm px-3 py-1 shadow-raised-sm border border-border-subtle text-xs font-medium text-text-primary">
								<span class="size-2 rounded-full bg-accent-default"></span> 回龙窝街区 · 14人
							</div>
							<!-- 中心我所在的点 -->
							<div class="relative flex flex-col items-center">
								<span class="relative flex size-6">
									<span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-accent-default opacity-soft"></span>
									<span class="relative inline-flex size-6 rounded-full bg-accent-default items-center justify-center text-text-on-accent shadow-raised-md">
										<Radio class="size-3.5" />
									</span>
								</span>
								<span class="mt-2 rounded-full bg-surface-primary px-3 py-0.5 text-xs font-bold text-text-primary shadow-xs border border-border-subtle">
									当前定位 · 5km 探索圈
								</span>
							</div>

							<!-- 右上角快捷切换至全屏地图 -->
							<button
								type="button"
								class="absolute top-3 right-3 flex items-center gap-1.5 rounded-lg bg-surface-primary/95 backdrop-blur-sm px-3 py-1.5 text-xs font-semibold text-text-primary shadow-xs border border-border-subtle hover:bg-surface-primary hover:text-accent-default transition"
								onclick={() => void goto(ROUTES.map)}
							>
								<MapPin class="size-3.5 text-accent-default" />
								<span>全景地图</span>
							</button>
						</div>

						<!-- 雷达控制面板与指标 -->
						<div class="p-4 space-y-3">
							<div class="grid grid-cols-3 gap-2.5 text-center">
								<div class="rounded-xl bg-surface-subtle p-3">
									<p class="text-xs text-text-secondary">周边动态</p>
									<p class="text-xl font-bold text-text-primary mt-0.5">38 <span class="text-xxs font-normal text-text-secondary">条</span></p>
								</div>
								<div class="rounded-xl bg-surface-subtle p-3">
									<p class="text-xs text-text-secondary">活跃邻居</p>
									<p class="text-xl font-bold text-text-primary mt-0.5">12 <span class="text-xxs font-normal text-text-secondary">人</span></p>
								</div>
								<div class="rounded-xl bg-surface-subtle p-3">
									<p class="text-xs text-text-secondary">同城圈子</p>
									<p class="text-xl font-bold text-text-primary mt-0.5">7 <span class="text-xxs font-normal text-text-secondary">个</span></p>
								</div>
							</div>

							<div class="flex items-center justify-between pt-1">
								<span class="text-xs text-text-secondary">当前状态：实时雷达已就绪</span>
								<button
									type="button"
									class="text-xs font-semibold text-accent-default hover:underline"
									onclick={() => void goto(ROUTES.nearby)}
								>
									打开周边探索中心 &rarr;
								</button>
							</div>
						</div>
					</div>

					<!-- 方案 D 活跃邻居与热议 -->
					<div class="grid grid-cols-1 xl:grid-cols-2 gap-4">
						{#if activeNeighbors.length > 0}
							<RailSection title="活跃邻居" variant="island">
								<div class="flex flex-col divide-y divide-border-subtle/30">
									{#each activeNeighbors as neighbor (neighbor.userId)}
										<div class="flex items-center gap-3 py-2.5 first:pt-0 last:pb-0">
											<Avatar src={neighbor.avatarUrl} name={neighbor.nickname} size="sm" />
											<div class="min-w-0 flex-1">
												<button
													type="button"
													class="text-left text-xs font-semibold text-text-primary hover:text-accent-default hover:underline transition-colors truncate block w-full"
													onclick={() => void goto(ROUTES.user(neighbor.userId))}
												>
													{neighbor.nickname}
												</button>
												{#if neighbor.location}
													<span class="text-xxs text-text-secondary truncate block mt-0.5">{neighbor.location}</span>
												{/if}
											</div>
										</div>
									{/each}
								</div>
							</RailSection>
						{/if}

						<RailSection title="社区热议" variant="island">
							<div class="flex flex-col divide-y divide-border-subtle/30">
								{#each trendingTopics.slice(0, 4) as topic (topic.tag)}
									<button
										type="button"
										class="hover-btn flex w-full items-center justify-between py-2 text-left transition hover:text-accent-default focus-ring first:pt-0 last:pb-0"
										onclick={() => void goto(`${ROUTES.search}?q=${encodeURIComponent('#' + topic.tag)}`)}
									>
										<div class="min-w-0 flex-1">
											<div class="flex items-center gap-1.5">
												<TrendingUp class="size-3 text-accent-default shrink-0" strokeWidth={1.75} />
												<span class="text-xs font-medium text-text-primary truncate">#{topic.tag}</span>
											</div>
										</div>
										<span class="text-xxs text-text-secondary tabular-nums shrink-0 ml-2">{topic.count} 条</span>
									</button>
								{/each}
							</div>
						</RailSection>
					</div>
				</div>
		{:else}
			<!-- 桌面端右侧 Rail：三块卡统一 RailSection + SectionHead(rail) 卡头（标题 + 右侧动作），
			     内容一律「行式」节奏；不再手写卡头、也不再每行堆一个「查看」按钮（整行可点）。 -->
			<div class="flex flex-col gap-4 py-2">
				{#if activeNeighbors.length > 0}
					<RailSection
						title="活跃邻居"
						icon={Users}
						action="附近的人"
						onAction={() => void goto(ROUTES.nearbyPeople)}
						variant="island"
					>
						<div class="flex flex-col divide-y divide-border-subtle/30">
							{#each activeNeighbors as neighbor (neighbor.userId)}
								<button
									type="button"
									class="hover-bg focus-ring group/row -mx-2 flex min-w-0 items-center gap-2.5 rounded-sm px-2 py-2.5 text-left transition-colors first:pt-0 last:pb-0"
									aria-label={`查看 ${neighbor.nickname} 的个人主页`}
									onclick={() => void goto(ROUTES.user(neighbor.userId))}
								>
									<Avatar src={neighbor.avatarUrl} name={neighbor.nickname} size={32} />
									<span class="min-w-0 flex-1">
										<span class="block truncate text-sm font-medium text-text-primary">{neighbor.nickname}</span>
										{#if neighbor.location}
											<span class="mt-0.5 block truncate text-xxs text-text-secondary">{neighbor.location}</span>
										{/if}
									</span>
									<ChevronRight
										class="size-3.5 shrink-0 text-text-secondary transition-transform duration-rapid group-hover/row:translate-x-0.5 group-hover/row:text-accent-default"
										strokeWidth={1.5}
									/>
								</button>
							{/each}
						</div>
					</RailSection>
				{/if}

				<!-- 社区热议：序号 + 话题 + 计数构成「数字与秩序」节奏（数字等宽、不抖动） -->
				<RailSection
					title="社区热议"
					icon={TrendingUp}
					action="查看全部"
					onAction={() => void goto(ROUTES.search)}
					variant="island"
				>
					<div class="flex flex-col divide-y divide-border-subtle/30">
						{#each trendingTopics.slice(0, 5) as topic, i (topic.tag)}
							<button
								type="button"
								class="hover-bg focus-ring group/row -mx-2 flex w-full items-center gap-2.5 rounded-sm px-2 py-2 text-left transition-colors first:pt-0 last:pb-0"
								aria-label={`搜索话题 #${topic.tag}`}
								onclick={() => void goto(`${ROUTES.search}?q=${encodeURIComponent('#' + topic.tag)}`)}
							>
								<span class="w-4 shrink-0 text-xxs font-medium tabular-nums text-text-secondary">{String(i + 1).padStart(2, '0')}</span>
								<span class="min-w-0 flex-1">
									<span class="block truncate text-sm font-medium text-text-primary transition-colors group-hover/row:text-accent-default">#{topic.tag}</span>
									<span class="mt-0.5 block truncate text-xxs text-text-secondary">{topic.desc}</span>
								</span>
								<span class="shrink-0 text-xs tabular-nums text-text-secondary">{topic.count} 条</span>
							</button>
						{/each}
					</div>
				</RailSection>

				<!-- 附近活动：rail 的地理轴线索（位置 + 距离），补足主列高度 -->
				{#if railNearbyMoments.length > 0}
					<RailSection
						title="附近活动"
						icon={MapPin}
						action="去附近"
						onAction={() => void goto(ROUTES.nearby)}
						variant="island"
					>
						<div class="flex flex-col divide-y divide-border-subtle/30">
							{#each railNearbyMoments as item (item.id)}
								<div class="py-2 first:pt-0 last:pb-0">
									<RailMomentThumb
										image={item.image}
										title={item.title}
										meta={item.meta}
										onClick={() => void goto(ROUTES.moment(item.id))}
									/>
								</div>
							{/each}
						</div>
					</RailSection>
				{/if}

				<!-- 探索雷达：改成「入口卡」形态——图标砖 + 一句说明 + 箭头 -->
				<RailSection
					title="探索雷达"
					icon={Compass}
					action="全景地图"
					onAction={() => void goto(ROUTES.map)}
					variant="island"
				>
					<button
						type="button"
						class="hover-bg focus-ring group/entry -mx-2 flex w-full items-center gap-3 rounded-md px-2 py-2 text-left transition-colors"
						onclick={() => void goto(ROUTES.nearby)}
					>
						<span
							class="flex size-10 shrink-0 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft transition-transform duration-rapid group-hover/entry:scale-105"
						>
							<Compass class="size-5" strokeWidth={1.75} />
						</span>
						<span class="min-w-0 flex-1">
							<span class="block text-sm font-semibold text-text-primary">发现身边 5km</span>
							<span class="mt-0.5 block truncate text-xxs text-text-secondary">人 · 群组 · 瞬间三合一雷达</span>
						</span>
						<ArrowRight
							class="size-4 shrink-0 text-text-secondary transition-all duration-rapid group-hover/entry:translate-x-0.5 group-hover/entry:text-accent-default"
							strokeWidth={1.75}
						/>
					</button>
				</RailSection>
			</div>
		{/if}
		{/snippet}
	</SplitPageLayout>
</div>

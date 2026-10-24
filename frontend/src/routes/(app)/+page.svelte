<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, Search, Compass, ArrowRight, Sparkles, TrendingUp, Users, LogIn, ExternalLink } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import type { MomentItem } from '@/api/types';
	import { searchMoments, fetchFollowingMoments } from '@/api';
	import HomeFeedItem from '@/components/home/HomeFeedItem.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { ROUTES } from '@/lib/routes';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
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

	type FeedTab = 'discover' | 'following';
	let activeTab = $state<FeedTab>('discover');
	let booting = $state(true);

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

	// 发现流：全局预加载
	$effect(() => {
		if (!discover.loaded && !discover.loading) {
			void discover.load().finally(() => {
				booting = false;
			});
		}
	});

	// 关注流：登录用户加载
	$effect(() => {
		if (auth.status === 'authenticated') {
			if (!following.loaded && !following.loading) {
				void following.load();
			}
		}
	});

	// 当前激活 Tab 对应的数据与加载状态
	const currentPager = $derived(activeTab === 'discover' ? discover : following);
	const currentItems = $derived(currentPager.items);
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

{#snippet feedTabs()}
	<!-- 顶部 Tab 栏：居中紧凑对齐，滑动下划线与吸顶毛玻璃质感 -->
	<div class="px-4 sm:px-5 bg-surface-primary/90 backdrop-blur-md sticky top-0 z-10 border-b border-border-subtle/30">
		<div class="max-w-[280px] mx-auto">
			<TabBar
				variant="underline"
				items={[
					{ id: 'discover', label: '为你推荐' },
					{ id: 'following', label: '正在关注' }
				]}
				value={activeTab}
				onChange={(id) => { activeTab = id as FeedTab; }}
			/>
		</div>
	</div>
{/snippet}

{#snippet nearbyBanner()}
	<button
		type="button"
		class="hover-btn group flex w-full items-center gap-3.5 rounded-2xl border border-border-default bg-surface-primary px-4 py-4 text-left focus-ring transition mt-4 hover:border-accent-default/40 hover:shadow-raised-md shadow-raised-sm"
		onclick={() => void goto(ROUTES.nearby)}
	>
		<span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-accent-soft text-accent-default transition-transform group-hover:scale-105">
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
		contentMaxWidth="640px"
		contentClassName="animate-fade-in flex flex-col pt-0 pb-8"
	>
		{#snippet header()}
			{#if device !== 'desktop'}
				<!-- 移动端顶栏：品牌 Logo + 快捷动作 + 分段 Tab 一体化（单一吸顶容器，杜绝双重边框） -->
				<div class="bg-surface-primary/95 backdrop-blur-md border-b border-border-subtle/30 sticky top-0 z-20">
					<div class="flex items-center justify-between px-4 pt-3 pb-1">
						<div class="flex items-center gap-2">
							<img src={imgHotSpot} alt="HotSpot" class="size-6.5 rounded-full object-cover shadow-xs" />
							<span class="text-base font-bold tracking-tight text-text-primary">HotSpot</span>
						</div>
						<h1 class="sr-only">首页</h1>
						<div class="flex items-center gap-1">
							<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} variant="ghost" size={36} />
							<ActionIconButton icon={Bell} label="通知" onClick={() => void goto(ROUTES.notify)} variant="ghost" size={36} />
						</div>
					</div>
					<div class="max-w-[260px] mx-auto px-4">
						<TabBar
							variant="underline"
							items={[
								{ id: 'discover', label: '为你推荐' },
								{ id: 'following', label: '正在关注' }
							]}
							value={activeTab}
							onChange={(id) => { activeTab = id as FeedTab; }}
						/>
					</div>
				</div>
			{/if}
		{/snippet}

		{#if device === 'desktop'}
			<!-- 桌面端居中 Tab 栏 -->
			{@render feedTabs()}
		{/if}

		<!-- 关注流未登录态：温馨引导登录卡片 -->
		{#if activeTab === 'following' && auth.status !== 'authenticated'}
			<div class="mx-4 my-8 rounded-2xl border border-border-default bg-surface-subtle p-6 text-center shadow-raised-sm">
				<div class="mx-auto flex size-12 items-center justify-center rounded-full bg-accent-soft text-accent-default mb-3">
					<Users class="size-6" strokeWidth={1.75} />
				</div>
				<h2 class="text-base font-semibold text-text-primary">登录后查看关注动态</h2>
				<p class="mt-1 text-xs text-text-secondary max-w-sm mx-auto">
					关注同频的好友与活跃邻居，即时获取 TA 们的最新瞬间与生活记录。
				</p>
				<button
					type="button"
					class="hover-btn mt-4 inline-flex items-center gap-1.5 rounded-full bg-accent-default px-5 py-2 text-xs font-medium text-text-on-accent focus-ring transition shadow-raised-sm"
					onclick={() => void goto(ROUTES.login)}
				>
					<LogIn class="size-3.5" />
					<span>立即登录</span>
				</button>
			</div>
		{:else if currentLoading}
			<InlineLoading label="加载动态中…" class="py-16" />
		{:else if currentError && currentItems.length === 0}
			<LoadError message={currentError} onRetry={() => void currentPager.load()} />
		{:else if currentItems.length > 0}
			<!-- 高质感社区 Feed 流：独立画廊岛屿卡片流 -->
			<div class="flex flex-col gap-3.5 sm:gap-4.5 px-3.5 sm:px-0 pt-3.5">
				{#each currentItems as moment (moment.moment_id)}
					<HomeFeedItem
						{moment}
						showDistance={false}
						onClick={() => void goto(ROUTES.moment(moment.moment_id))}
					/>
				{/each}
			</div>

			{#if hasMore}
				<div class="mt-5 flex justify-center px-4">
					<LoadMoreButton loading={loadingMore} onLoad={() => void currentPager.loadMore()} />
				</div>
			{:else}
				<ListEnd class="py-4" />
			{/if}

			<div class="px-3.5 sm:px-0">
				{@render nearbyBanner()}
			</div>
		{:else}
			<EmptyState
				title={activeTab === 'following' ? '暂无关注动态' : '还没有动态'}
				description={activeTab === 'following' ? '去「为你推荐」发现更多有趣的人，或去附近看看身边正在发生什么。' : '去附近看看，或者发布属于你的第一条瞬间。'}
				bare
				class="py-16"
			>
				{#snippet icon()}
					<Sparkles class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
				{/snippet}
			</EmptyState>

			<div class="px-3.5 sm:px-0">
				{@render nearbyBanner()}
			</div>
		{/if}

		{#snippet rail()}
			<!-- 桌面端右侧 Rail 辅助区：活跃邻居 + 社区热议话题 -->
			<div class="flex flex-col gap-4">
				<!-- 活跃邻居推荐 -->
				{#if activeNeighbors.length > 0}
					<RailSection title="活跃邻居">
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
									<button
										type="button"
										class="hover-btn shrink-0 rounded-full border border-border-default bg-surface-primary px-3 py-1 text-xxs font-medium text-text-primary hover:border-accent-default hover:text-accent-default hover:bg-accent-soft/30 focus-ring transition shadow-xs"
										onclick={() => void goto(ROUTES.user(neighbor.userId))}
									>
										查看
									</button>
								</div>
							{/each}
						</div>
					</RailSection>
				{/if}

				<!-- 社区热议话题 -->
				<RailSection title="社区热议">
					<div class="flex flex-col divide-y divide-border-subtle/30">
						{#each trendingTopics as topic (topic.tag)}
							<button
								type="button"
								class="hover-btn flex w-full items-center justify-between py-2 text-left transition hover:text-accent-default focus-ring first:pt-0 last:pb-0"
								onclick={() => void goto(`${ROUTES.search}?q=${encodeURIComponent('#' + topic.tag)}`)}
							>
								<div class="min-w-0 flex-1">
									<div class="flex items-center gap-1.5">
										<TrendingUp class="size-3 text-accent-default shrink-0" strokeWidth={1.75} />
										<span class="text-xs font-medium text-text-primary truncate hover:text-accent-default">#{topic.tag}</span>
									</div>
									<span class="text-xxs text-text-secondary truncate block mt-0.5 pl-4.5">{topic.desc}</span>
								</div>
								<span class="text-xxs text-text-secondary tabular-nums shrink-0 ml-2">{topic.count} 条</span>
							</button>
						{/each}
					</div>
				</RailSection>

				<!-- 快捷探索周边 -->
				<RailSection title="探索雷达">
					<button
						type="button"
						class="hover-btn group flex w-full items-center gap-3 text-left transition focus-ring"
						onclick={() => void goto(ROUTES.nearby)}
					>
						<span class="flex size-9 shrink-0 items-center justify-center rounded-full bg-accent-soft text-accent-default transition-transform group-hover:scale-105">
							<Compass class="size-4.5" strokeWidth={1.75} />
						</span>
						<div class="min-w-0 flex-1">
							<span class="block text-xs font-semibold text-text-primary group-hover:text-accent-default transition-colors">发现身边 5km</span>
							<span class="block truncate text-xxs text-text-secondary mt-0.5">人 · 群组 · 动态三合一雷达</span>
						</div>
						<ExternalLink class="size-3.5 text-text-secondary shrink-0 group-hover:text-accent-default" />
					</button>
				</RailSection>
			</div>
		{/snippet}
	</SplitPageLayout>
</div>

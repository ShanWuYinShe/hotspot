<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, Search, Compass, ArrowRight, Sparkles, Users, TrendingUp } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import type { MomentItem } from '@/api/types';
	import { searchMoments, fetchFollowingMoments, listFollowing } from '@/api';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ActivityRow from '@/components/home/ActivityRow.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import Button from '@/components/ui/button.svelte';
	import { resolveHomeSegment, resolveFollowingEmpty, type HomeFeedSegment } from '@/components/home/home-feed-mode';
	import { auth } from '@/stores/auth.svelte';
	import { ROUTES } from '@/lib/routes';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';

	// ═══ 首页 = 关系轴（关注）+ 算法轴（发现）（2026-10-19 信息架构调整 · Step 2）═══
	// 三条轴各占一个 Tab，内容不重复：
	//   首页「关注」= moments/fetch_following（我关注的人的最新动态，需登录）
	//   首页「发现」= moments/search 热门模式（spec api_spec_moment.md:64「发现页首屏」，匿名可用）
	//   附近「地理」= /nearby 枢纽页（人/群组/瞬间，5km）
	// 默认段与关注段空态由 components/home/home-feed-mode.ts 的纯函数决定（含冷启动矩阵，
	// 已用用例锁定）：未登录/零关系 → 发现；有关注 → 关注。

	const PAGE_SIZE = 20;
	const device = $derived(platform.platform);

	// 段选择：首屏判定一次，此后完全由用户切换控制（不被后续计数变化翻动）。
	// auth.user 是精简结构（user_id/nickname/avatar_url/user_type，见 stores/auth.svelte.ts），
	// **不含 following_count**，故关注数由 users/fetch_following 探测（limit:1 足够分辨零与非零），
	// 与关注流首屏并发拉取——判定在加载态内完成，用户不会看到「先关注后跳发现」的翻动。
	let segment = $state<HomeFeedSegment>('discover');
	let followingCount = $state<number | null>(null);
	let booting = $state(true);

	// 两个流共用一条分页管线（cursor 不透明，只透传；spec api_spec_base.md）
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

		/** 首屏已由外部并发取回时直接注入（避免同一请求发两次） */
		function prime(newItems: MomentItem[], newCursor: string | null) {
			items = newItems;
			nextCursor = newCursor;
			loaded = true;
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
			reset,
			prime
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

	// 首屏：未登录直接进「发现」；已登录则并发探测（关注流首屏 + 关注数）后判默认段
	$effect(() => {
		if (!booting) return;
		if (auth.status === 'loading') return;
		const uid = auth.user?.user_id;
		if (auth.status !== 'authenticated' || !uid) {
			segment = resolveHomeSegment(false, 0);
			booting = false;
			return;
		}
		let cancelled = false;
		void (async () => {
			try {
				const [feedRes, followRes] = await Promise.all([
					fetchFollowingMoments({ pagination: { limit: PAGE_SIZE } }),
					listFollowing({ user_id: uid, pagination: { limit: 1 } })
				]);
				if (cancelled) return;
				const page = paged(feedRes);
				following.prime(page.items, page.nextCursor);
				followingCount = followRes.code === 0 ? (followRes.data?.items?.length ?? 0) : 0;
				segment = resolveHomeSegment(true, followingCount);
			} catch {
				// 探测失败不阻塞首页：退回发现轴（关注段仍可手动进入）
				if (!cancelled) segment = 'discover';
			} finally {
				if (!cancelled) booting = false;
			}
		})();
		return () => {
			cancelled = true;
		};
	});

	// 懒加载：切到某段才拉该段数据（关注段首屏已由上面的并发探测注入）
	$effect(() => {
		if (segment === 'discover' && !discover.loaded && !discover.loading) void discover.load();
	});
	$effect(() => {
		if (segment !== 'following' || auth.status !== 'authenticated') return;
		if (!following.loaded && !following.loading) void following.load();
	});

	const followingEmpty = $derived(resolveFollowingEmpty(followingCount));
</script>
<svelte:head>
	<title>首页 · HotSpot</title>
</svelte:head>

{#snippet momentRows(pager: ReturnType<typeof createMomentPager>)}
	<ListContainer bordered>
		{#each pager.items as moment, i (moment.moment_id)}
			<!-- 非地理轴（关系流/全站热榜）不显示「距离」：全局内容配「300m」会误导为身边内容 -->
			<ActivityRow {moment} index={i} showDistance={false} onClick={() => void goto(ROUTES.moment(moment.moment_id))} />
		{/each}
	</ListContainer>
	{#if pager.hasMore}
		<LoadMoreButton loading={pager.loadingMore} onLoad={() => void pager.loadMore()} />
	{:else}
		<ListEnd />
	{/if}
{/snippet}

{#snippet nearbyEntry()}
	<button
		type="button"
		class="card-hover flex w-full items-center gap-3 rounded-lg border border-border-default bg-surface-primary px-4 py-3 text-left focus-ring"
		onclick={() => void goto(ROUTES.nearby)}
	>
		<span class="flex size-9 shrink-0 items-center justify-center rounded-full bg-accent-soft">
			<Compass class="size-4.5 text-accent-default" strokeWidth={1.75} />
		</span>
		<span class="min-w-0 flex-1">
			<span class="block text-sm font-medium text-text-primary">看看附近</span>
			<span class="block truncate text-xs text-text-secondary">附近的人 · 群组 · 瞬间，5km 内</span>
		</span>
		<ArrowRight class="size-4 shrink-0 text-text-secondary" strokeWidth={1.75} />
	</button>
{/snippet}

<StandardPageLayout contentClassName="animate-fade-in flex flex-col gap-5 pt-4 pb-16">
	{#snippet header()}
		<PageHeader title="首页" titleSize={device === 'desktop' ? 'xl' : '2xl'} subtitle="关注与发现" sticky>
			{#snippet below()}
				<!-- 系统级分段控制器（TabBar segmented 原语）：关系轴 / 算法轴切换 -->
				<div class="mt-3 pb-2">
					<div class="w-full max-w-72">
						<TabBar
							variant="segmented"
							items={[
								{ id: 'following', label: '关注', icon: Users },
								{ id: 'discover', label: '发现', icon: TrendingUp }
							]}
							value={segment}
							onChange={(id) => (segment = id as HomeFeedSegment)}
						/>
					</div>
				</div>
			{/snippet}
			{#snippet trailing()}
				<div class="flex shrink-0 items-center gap-2">
					{#if device !== 'desktop'}
						<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} variant="ghost" size={36} />
						<ActionIconButton icon={Bell} label="通知" onClick={() => void goto(ROUTES.notify)} variant="ghost" size={36} />
					{/if}
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if booting}
		<InlineLoading label="加载首页…" />
	{:else if segment === 'following'}
		{#if auth.status !== 'authenticated'}
			<NeedsAuthNotice title="登录后查看关注动态" description="关注的人发布的新动态会出现在这里" />
		{:else if following.loading}
			<InlineLoading label="加载关注动态…" />
		{:else if following.error}
			<LoadError message={following.error} onRetry={() => { following.reset(); }} />
		{:else if following.items.length === 0}
			<!-- 空态按 following_count 分流：零关系与「有关注但没人更新」都返回空数组，
			     文案不能混用（见 home-feed-mode.ts 的用例锁定） -->
			<EmptyState title={followingEmpty.title} description={followingEmpty.description}>
				{#snippet icon()}
					<Users class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
				{/snippet}
				{#snippet action()}
					<Button
						type="button"
						variant="primary"
						size="md"
						onclick={() => (followingEmpty.kind === 'no-relations' ? void goto(ROUTES.nearby) : (segment = 'discover'))}
					>
						{followingEmpty.actionLabel}
					</Button>
				{/snippet}
			</EmptyState>
		{:else}
			{@render momentRows(following)}
		{/if}
	{:else}
		{#if discover.loading}
			<InlineLoading label="加载热门内容…" />
		{:else if discover.error}
			<LoadError message={discover.error} onRetry={() => discover.reset()} />
		{:else}
			{@render nearbyEntry()}
			{#if discover.items.length === 0}
				<EmptyState title="暂时还没有热门内容" description="去附近看看，或者发布第一条动态。">
					{#snippet icon()}
						<Sparkles class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
					{/snippet}
				</EmptyState>
			{:else}
				<div>
					<SectionHead title="热门动态" subtitle="全站热度 · 实时更新" />
					{@render momentRows(discover)}
				</div>
			{/if}
		{/if}
	{/if}
</StandardPageLayout>

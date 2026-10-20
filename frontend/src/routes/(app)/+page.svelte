<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, Search, Compass, ArrowRight, Sparkles } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import type { MomentItem } from '@/api/types';
	import { searchMoments, fetchFollowingMoments, listFollowing } from '@/api';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ActivityRow from '@/components/home/ActivityRow.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { ROUTES } from '@/lib/routes';
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

	// 首屏：一条流 = 关系流（需登录）+ 发现流（匿名可用），两个 pager 并行加载。
	// 没有任何"默认段"判定——用户不需要知道自己在看哪个流（2026-10-20）。
	$effect(() => {
		if (!booting) return;
		if (auth.status === 'loading') return;
		const uid = auth.user?.user_id;
		if (auth.status !== 'authenticated' || !uid) {
			booting = false;
			return;
		}
		let cancelled = false;
		void (async () => {
			await following.load();
			if (!cancelled) booting = false;
		})();
		return () => {
			cancelled = true;
		};
	});

	// 发现流：登录与否都加载（热门内容匿名可读），与关注流并行
	$effect(() => {
		if (!discover.loaded && !discover.loading) void discover.load();
	});

	// 关注流（未登录不请求）
	$effect(() => {
		if (auth.status !== 'authenticated') return;
		if (!following.loaded && !following.loading) void following.load();
	});

	// 关注流已出现的 moment_id：发现流渲染时去重，避免同一条重复出现
	const followingIds = $derived(new Set(following.items.map((m) => m.moment_id)));
</script>
<svelte:head>
	<title>首页 · HotSpot</title>
</svelte:head>

{#snippet momentRows(pager: ReturnType<typeof createMomentPager>, exclude?: Set<string>)}
	{@const rows = exclude ? pager.items.filter((m) => !exclude.has(m.moment_id)) : pager.items}
	{#if rows.length > 0}
	<ListContainer bordered>
		{#each rows as moment, i (moment.moment_id)}
			<!-- 非地理轴（关系流/全站热榜）不显示「距离」：全局内容配「300m」会误导为身边内容 -->
			<ActivityRow {moment} index={i} showDistance={false} onClick={() => void goto(ROUTES.moment(moment.moment_id))} />
		{/each}
	</ListContainer>
	{/if}
	{#if pager.hasMore}
		<LoadMoreButton loading={pager.loadingMore} onLoad={() => void pager.loadMore()} />
	{:else if rows.length > 0}
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
		<!-- 无可见标题：只留右侧搜索/通知两个动作，内容直接开头的「沉浸流」（2026-10-20 用户审查：
		     不要标题块、不要切换器；用户只需看内容，不需要被告知"当前在看哪个流"）。 -->
		<div class="flex items-center justify-end pt-3 pb-1">
			<!-- sr-only 的 h1 是绝对定位（不占位），所以这里用它当首个 flex 子项时
			     justify-between 会把动作推到左边——直接右对齐 -->
			<h1 class="sr-only">首页</h1>
			<div class="flex items-center gap-2">
				{#if device !== 'desktop'}
					<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} variant="ghost" size={36} />
					<ActionIconButton icon={Bell} label="通知" onClick={() => void goto(ROUTES.notify)} variant="ghost" size={36} />
				{/if}
			</div>
		</div>
	{/snippet}

	{#if booting}
		<InlineLoading label="加载中…" />
	{:else}
		<!-- 关系流（需登录）在前：与你最相关的内容优先，不加任何分组标题 -->
		{#if auth.status === 'authenticated'}
			{#if following.error}
				<LoadError message={following.error} onRetry={() => following.reset()} />
			{:else if following.loading}
				<InlineLoading label="加载中…" />
			{:else if following.items.length > 0}
				{@render momentRows(following)}
			{/if}
		{/if}

		<!-- 发现流续后（按 moment_id 去重） -->
		{#if discover.error}
			<LoadError message={discover.error} onRetry={() => discover.reset()} />
		{:else if discover.loading}
			<InlineLoading label="加载内容…" />
		{:else if discover.items.length > 0}
			{@render momentRows(discover, followingIds)}
		{:else if !following.loading && following.items.length === 0}
			<EmptyState title="还没有内容" description="去附近看看，或者发布第一条动态。">
				{#snippet icon()}
					<Sparkles class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
				{/snippet}
			</EmptyState>
		{/if}

		{@render nearbyEntry()}
	{/if}
</StandardPageLayout>

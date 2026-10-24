<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, Search, Compass, ArrowRight, Sparkles, Users, MessageCircle } from '@lucide/svelte';
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
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
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

	// ═══ 真正的单一流合并（关系流在前，发现流按 moment_id 去重续后）═══
	const feedMoments = $derived.by(() => {
		const list: MomentItem[] = [];
		const seen = new Set<string>();
		if (auth.status === 'authenticated') {
			for (const m of following.items) {
				if (!seen.has(m.moment_id)) {
					seen.add(m.moment_id);
					list.push(m);
				}
			}
		}
		for (const m of discover.items) {
			if (!seen.has(m.moment_id)) {
				seen.add(m.moment_id);
				list.push(m);
			}
		}
		return list;
	});

	const feedError = $derived(following.error ?? discover.error);
	const feedLoading = $derived(booting || (feedMoments.length === 0 && (following.loading || discover.loading)));
	const hasMoreFeed = $derived(
		auth.status === 'authenticated' && following.hasMore ? true : discover.hasMore
	);
	const loadingMoreFeed = $derived(following.loadingMore || discover.loadingMore);

	async function loadMoreFeed() {
		if (auth.status === 'authenticated' && following.hasMore) {
			await following.loadMore();
		} else if (discover.hasMore) {
			await discover.loadMore();
		}
	}
</script>
<svelte:head>
	<title>首页 · HotSpot</title>
</svelte:head>

{#snippet nearbyEntry()}
	<button
		type="button"
		class="hover-btn card-hover flex w-full items-center gap-3 rounded-lg border border-border-default bg-surface-primary px-4 py-3 text-left focus-ring transition mt-4"
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

<!-- 桌面采用 SplitPageLayout（主内容区限制在 640px 黄金阅读宽度 + 右侧 Rail 辅助区）
     移动端铺上 bg-surface-primary，与消息页保持 100% 纯白通透质感 -->
<div class="flex flex-col flex-1 min-h-full bg-surface-primary">
	<SplitPageLayout
		contentMaxWidth="640px"
		contentClassName="animate-fade-in flex flex-col gap-4 pt-3 pb-8"
	>
		{#snippet header()}
			{#if device !== 'desktop'}
				<!-- 移动端顶栏：左侧品牌字标锚点，右侧为搜索/通知快捷动作 -->
				<div class="flex items-center justify-between px-[var(--cst-page-gutter)] pt-3 pb-2 bg-surface-primary border-b border-border-subtle/30">
					<div class="flex items-center gap-1.5">
						<span class="text-base font-bold tracking-tight text-text-primary">HotSpot</span>
					</div>
					<h1 class="sr-only">首页</h1>
					<div class="flex items-center gap-2">
						<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} variant="ghost" size={36} />
						<ActionIconButton icon={Bell} label="通知" onClick={() => void goto(ROUTES.notify)} variant="ghost" size={36} />
					</div>
				</div>
			{/if}
		{/snippet}

		{#if feedLoading}
			<InlineLoading label="加载动态中…" class="py-12" />
		{:else if feedError && feedMoments.length === 0}
			<LoadError message={feedError} onRetry={() => { following.reset(); discover.reset(); }} />
		{:else if feedMoments.length > 0}
			<!-- 单一连续平铺流：消除任何中间“没有更多了”和断裂卡片 -->
			<ListContainer class="divide-y divide-border-subtle/30">
				{#each feedMoments as moment, i (moment.moment_id)}
					<ActivityRow {moment} index={i} showDistance={false} onClick={() => void goto(ROUTES.moment(moment.moment_id))} />
				{/each}
			</ListContainer>

			{#if hasMoreFeed}
				<div class="mt-2 flex justify-center">
					<LoadMoreButton loading={loadingMoreFeed} onLoad={() => void loadMoreFeed()} />
				</div>
			{:else}
				<ListEnd class="py-3" />
			{/if}

			{@render nearbyEntry()}
		{:else}
			<EmptyState title="还没有内容" description="去附近看看，或者发布第一条动态。" bare class="py-16">
				{#snippet icon()}
					<Sparkles class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
				{/snippet}
			</EmptyState>

			{@render nearbyEntry()}
		{/if}

		{#snippet rail()}
			<!-- 桌面端右侧 Rail 辅助区：探索社区快捷入口，平衡主内容区视觉重心 -->
			<div class="flex flex-col gap-4">
				<RailSection title="探索社区">
					<div class="flex flex-col gap-2">
						<button
							type="button"
							class="hover-btn flex w-full items-center gap-3 rounded-md border border-border-default bg-surface-primary px-3.5 py-2.5 text-left transition focus-ring"
							onclick={() => void goto(ROUTES.nearbyPeople)}
						>
							<span class="flex size-8 shrink-0 items-center justify-center rounded-full bg-accent-soft">
								<Users class="size-4 text-accent-default" strokeWidth={1.75} />
							</span>
							<div class="min-w-0 flex-1">
								<span class="block text-sm font-medium text-text-primary">附近的人</span>
								<span class="block truncate text-xs text-text-secondary">发现身边的真实伙伴</span>
							</div>
							<ArrowRight class="size-3.5 text-text-secondary shrink-0" />
						</button>

						<button
							type="button"
							class="hover-btn flex w-full items-center gap-3 rounded-md border border-border-default bg-surface-primary px-3.5 py-2.5 text-left transition focus-ring"
							onclick={() => void goto(ROUTES.nearbyGroups)}
						>
							<span class="flex size-8 shrink-0 items-center justify-center rounded-full bg-accent-soft">
								<MessageCircle class="size-4 text-accent-default" strokeWidth={1.75} />
							</span>
							<div class="min-w-0 flex-1">
								<span class="block text-sm font-medium text-text-primary">附近的群组</span>
								<span class="block truncate text-xs text-text-secondary">加入同频兴趣圈子</span>
							</div>
							<ArrowRight class="size-3.5 text-text-secondary shrink-0" />
						</button>

						<button
							type="button"
							class="hover-btn flex w-full items-center gap-3 rounded-md border border-border-default bg-surface-primary px-3.5 py-2.5 text-left transition focus-ring"
							onclick={() => void goto(ROUTES.nearbyMoments)}
						>
							<span class="flex size-8 shrink-0 items-center justify-center rounded-full bg-accent-soft">
								<Compass class="size-4 text-accent-default" strokeWidth={1.75} />
							</span>
							<div class="min-w-0 flex-1">
								<span class="block text-sm font-medium text-text-primary">附近瞬间</span>
								<span class="block truncate text-xs text-text-secondary">看看身边正在发生什么</span>
							</div>
							<ArrowRight class="size-3.5 text-text-secondary shrink-0" />
						</button>
					</div>
				</RailSection>
			</div>
		{/snippet}
	</SplitPageLayout>
</div>

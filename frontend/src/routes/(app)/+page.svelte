<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, Search, Compass, ArrowRight, Sparkles } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import type { MomentItem } from '@/api/types';
	import { searchMoments } from '@/api';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ActivityRow from '@/components/home/ActivityRow.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import { ROUTES } from '@/lib/routes';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';

	// ═══ 首页「发现」（2026-10-19 信息架构调整）═══
	// 三个 Tab 各占一条轴，杜绝内容重复：
	//   首页 = 算法轴（无输入的热门发现）——spec api_spec_moment.md:64 明确
	//          「无输入的热门发现（发现页首屏）由 moments/search 的热门模式承担」，
	//          并声明该端点「不承担附近发现职责」；
	//   附近 = 地理轴（附近的人/群组/瞬间，5km）——见 routes/(app)/nearby/+page.svelte；
	//   关注流 = 关系轴（moments/fetch_following），后续独立接入（当前未上）。
	// 调整前首页的三个数据源全是 5km 地理查询（附近瞬间/用户/群组），事实上是「附近」
	// 的聚合页，而「附近」Tab 只落其中一段——重复由此而来。
	// 首页对零社交关系的新用户同样有内容：热门模式无输入条件、无需关系与位置，
	// 且经 pick(私有/公开) 走匿名版（spec api_spec_public.md:700）。

	const PAGE_SIZE = 20;

	let moments = $state<MomentItem[]>([]);
	let loading = $state(true);
	let loadingMore = $state(false);
	let error = $state<string | null>(null);
	let nextCursor = $state<string | null>(null);
	let reloadKey = $state(0);
	const device = $derived(platform.platform);
	const hasMore = $derived(Boolean(nextCursor));

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchFirstPage() {
			try {
				loading = true;
				error = null;
				const res = await searchMoments({ sort_by: 'popular', pagination: { limit: PAGE_SIZE } });
				if (cancelled) return;
				if (res.code !== 0) {
					error = res.message || '加载失败';
					return;
				}
				moments = res.data?.items ?? [];
				nextCursor = res.data?.next_cursor ?? null;
			} catch (e) {
				if (!cancelled) error = e instanceof Error ? e.message : '加载失败';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchFirstPage();
		return () => {
			cancelled = true;
		};
	});

	// 真分页：cursor 不透明，只透传（spec api_spec_base.md）
	const loadMore = async () => {
		if (loadingMore || !nextCursor) return;
		loadingMore = true;
		try {
			const res = await searchMoments({
				sort_by: 'popular',
				pagination: { limit: PAGE_SIZE, cursor: nextCursor }
			});
			if (res.code !== 0) {
				error = res.message || '加载失败';
				return;
			}
			const incoming = res.data?.items ?? [];
			const seen = new Set(moments.map((m) => m.moment_id));
			moments = [...moments, ...incoming.filter((m) => !seen.has(m.moment_id))];
			nextCursor = res.data?.next_cursor ?? null;
		} catch (e) {
			error = e instanceof Error ? e.message : '加载失败';
		} finally {
			loadingMore = false;
		}
	};
</script>
<svelte:head>
	<title>首页 · HotSpot</title>
</svelte:head>

<!-- ═══ 附近入口卡：地理轴内容的唯一入口（首页不再重复附近列表）═══ -->
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
		<!-- 箭头用既有语义类（text-text-tertiary 未注册，Tailwind v4 会静默失效）-->
		<ArrowRight class="size-4 shrink-0 text-text-secondary" strokeWidth={1.75} />
	</button>
{/snippet}

<StandardPageLayout contentClassName="animate-fade-in flex flex-col gap-5 pt-4 pb-16">
	{#snippet header()}
		<!-- 三端统一页头。副标题去掉「周边」——那属于「附近」轴（2026-10-19 IA 调整） -->
		<PageHeader title="首页" titleSize={device === 'desktop' ? 'xl' : '2xl'} subtitle="发现有趣的人和事" sticky>
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

	{#if loading}
		<InlineLoading label="加载热门内容…" />
	{:else if error}
		<LoadError message={error} onRetry={() => reloadKey++} />
	{:else}
		{@render nearbyEntry()}
		{#if moments.length === 0}
			<EmptyState title="暂时还没有热门内容" description="去附近看看，或者发布第一条动态。">
				{#snippet icon()}
					<Sparkles class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
				{/snippet}
			</EmptyState>
		{:else}
			<div>
				<SectionHead title="热门动态" subtitle="全站热度 · 实时更新" />
				<ListContainer bordered>
					{#each moments as moment, i (moment.moment_id)}
						<ActivityRow {moment} index={i} onClick={() => void goto(ROUTES.moment(moment.moment_id))} />
					{/each}
				</ListContainer>
				{#if hasMore}
					<LoadMoreButton loading={loadingMore} onLoad={() => void loadMore()} />
				{:else}
					<ListEnd />
				{/if}
			</div>
		{/if}
	{/if}
</StandardPageLayout>

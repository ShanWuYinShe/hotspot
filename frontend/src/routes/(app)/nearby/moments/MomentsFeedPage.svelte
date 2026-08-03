<script lang="ts">
	import { Sparkles } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platformStore } from '@/stores/platform.svelte';
	import { loadDefaultLocation } from '@/lib/default-location';
	import BspCardCanvas from '@/components/shared/BspCardCanvas.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import PostCard from '@/components/moments/PostCard.svelte';
	import { createFeedInteractions, type FeedItem } from '@/components/moments/feed-interactions.svelte';
	import { getFeedNearby } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import { nearbyContentMaxWidth } from '@/lib/nearby-layout';

	// 发现流（首页与 /nearby/moments 共用）：移动单列 / 平板双列 / 桌面
	// BSP 卡片墙。头部与附近的人对齐：标题+返回 / 搜索，下一行分类 Tab + tag 列表。
	// 迁移自旧版 MomentsFeedPage。

	const TABS = ['热门', '附近', '活动', '关注'];

	let apiFeedItems = $state<FeedItem[]>([]);
	let tags = $state<string[]>([]);
	let loading = $state(true);
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(null);

	$effect(() => {
		let cancelled = false;
		void loadDefaultLocation().then((loc) => {
			if (!cancelled) referencePoint = loc?.point ?? null;
		});
		return () => {
			cancelled = true;
		};
	});

	$effect(() => {
		// spec: feed/nearby reference_point is optional — server resolves via fallback
		// chain (explicit point → last known location → ClientConfig default).
		// 有 point 就传、没有就不传照常请求。
		let cancelled = false;
		async function fetchData() {
			try {
				loading = true;
				const feedRes = await getFeedNearby(referencePoint ? { reference_point: referencePoint } : {}).catch(() => ({ code: -1, data: { items: [] } }));
				if (cancelled) return;

				const feedData = (feedRes.code === 0 ? feedRes.data : null) as any;

				if (feedData) {
					const items: FeedItem[] = (feedData.items || []).map((m: any, i: number) => {
						const mine = (m.reactions || []).find((r: any) => r.reacted_by_me);
						return {
							id: m.moment_id ?? m.id ?? i + 1,
							user: m.user?.nickname ?? '',
							avatar: m.user?.avatar_url ?? '',
							location: m.location?.location_name ?? '',
							tag: m.tags?.[0] ?? '',
							content: m.description ?? m.content ?? '',
							image: m.images?.[0]?.url ?? '',
							likes: m.likes_count ?? 0,
							comments: m.comments_count ?? 0,
							time: m.created_at ?? '',
							liked: Boolean(mine),
							saved: Boolean(m.my_state?.is_favorited),
							likeEmoji: mine?.emoji ?? '👍'
						};
					});
					apiFeedItems = items;

					// Hot tags are aggregated from the feed itself — search/suggest is a
					// search-box autocomplete (needs a prefix) and must not be used to
					// browse tags. Dedupe tags across the returned moments, keep order.
					const seen = new Set<string>();
					const tagsList: string[] = [];
					for (const m of feedData.items || []) {
						for (const t of m.tags || []) {
							if (typeof t === 'string' && t && !seen.has(t)) {
								seen.add(t);
								tagsList.push(t);
							}
						}
					}
					tags = tagsList;
				}
			} catch {
				// empty data is normal
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchData();
		return () => {
			cancelled = true;
		};
	});

	// 三端共用的过滤逻辑（旧版每个布局内重复，此处收敛）
	let searchVal = $state('');
	let activeTab = $state('热门');
	let activeTag = $state<string | null>(null);
	let visibleCount = $state(12);

	// 传入 getter 而非数组快照：interactions.items 需跟随 apiFeedItems 更新
	const interactions = createFeedInteractions(() => apiFeedItems);

	const filtered = $derived.by(() => {
		const q = searchVal.trim().toLowerCase();
		return interactions.items.filter(
			(i) =>
				(!activeTag || i.tag === activeTag) &&
				(!q || i.user.toLowerCase().includes(q) || i.content.toLowerCase().includes(q) || i.location.toLowerCase().includes(q))
		);
	});

	const platform = $derived(platformStore.platform);
</script>

<!-- ═══ 三端统一页面 header（对齐 Figma DiscoverDesktop 结构：标题行 + 计数/精选 + 搜索 + Tab）═══ -->
{#snippet discoverHeader()}
	<PageHeader
		title="附近动态"
		sticky
		titleSize={platform === 'desktop' ? 20 : platform === 'tablet' ? 24 : 22}
		contentClassName="pt-4 pb-0"
		maxWidth={nearbyContentMaxWidth(platform)}
		onBack={() => history.back()}
	>
		{#snippet trailing()}
			{#if platform === 'tablet'}
				<div class="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-[var(--neu-accent-primary)]" style="box-shadow:0 3px 10px var(--neu-glow);">
					<Sparkles style="width:11px;height:11px;color:white;" strokeWidth={2} />
					<span class="text-white text-[11px]">精选</span>
				</div>
			{:else}
				<span class="text-[var(--neu-text-muted)] text-xs">{filtered.length} 条内容</span>
			{/if}
		{/snippet}
		{#snippet below()}
			<div class="mt-3 mb-2.5">
				<SearchInput value={searchVal} onChange={(v) => (searchVal = v)} placeholder="地点、人物、关键词…" class="px-4 py-2.5 rounded-relaxed" />
			</div>
			{@render tabBar()}
		{/snippet}
	</PageHeader>
{/snippet}

<!-- ═══ 分类 Tab 栏（下划线样式，对齐 Figma Make DiscoverHeader；随 header 吸顶）═══ -->
{#snippet tabBar()}
	<div class="no-scrollbar flex overflow-x-auto" style="border-bottom:1px solid var(--neu-border);">
		{#each TABS as tab}
			{@const isActive = activeTab === tab}
			<button
				type="button"
				onclick={() => (activeTab = tab)}
				aria-pressed={isActive}
				class="relative shrink-0 rounded-t-[12px] pb-3 pt-1 outline-none"
				style="padding-left:18px;padding-right:18px;color:{isActive ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};font-weight:{isActive ? 500 : 400};font-size:13px;transition:color 0.2s;"
			>
				{tab}
				{#if isActive}
					<span
						class="absolute"
						style="bottom:-1px;left:14%;right:14%;height:2px;border-radius:1px;background:var(--neu-accent-primary);"
					></span>
				{/if}
			</button>
		{/each}
	</div>
{/snippet}

<!-- ═══ 热门 tag 列表（scrollable / flex-wrap 由调用处 className 决定，对齐 Figma TagChips）═══ -->
{#snippet tagChips(className: string)}
	<div class="flex gap-1.5 no-scrollbar {className}">
		{#each tags as tag}
			<TagPill active={activeTag === tag} onClick={() => (activeTag = activeTag === tag ? null : tag)} size="md">{tag}</TagPill>
		{/each}
	</div>
{/snippet}

<!-- ═══ 加载中 / 空列表状态（与附近的人、附近群组一致）═══ -->
{#snippet feedState()}
	{#if loading}
		<div class="flex flex-col items-center gap-3 py-20">
			<div class="rounded-full border-2 border-[var(--neu-border)] border-t-[var(--neu-accent-primary)] animate-spin" style="width:28px;height:28px;"></div>
			<p class="text-[var(--neu-text-muted)] text-[13px]">加载附近动态…</p>
		</div>
	{:else}
		<EmptyState title="附近暂时没有瞬间">
			{#snippet icon()}
				<Sparkles size={24} color="var(--neu-text-muted)" opacity={0.4} />
			{/snippet}
		</EmptyState>
	{/if}
{/snippet}

{#if platform === 'desktop'}
	<!-- ═══ Desktop Discover — 统一 PageHeader + 3 列内容 ═══ -->
	<div class="content-mode-wide flex flex-col h-full">
		{@render discoverHeader()}

		<div class="flex flex-1 min-h-0 overflow-hidden">
			<!-- Main feed -->
			<div class="flex-1 overflow-y-auto no-scrollbar">
				<ContentColumn maxWidth={nearbyContentMaxWidth(platform)} class="pt-4 pb-6">
					<div class="mb-5">
						{@render tagChips('flex-wrap')}
					</div>

					{#if loading || !filtered.length}
						{@render feedState()}
					{:else}
						<BspCardCanvas
							items={filtered.slice(0, visibleCount)}
							gap={14}
							minRatio={0.68}
							maxRatio={1.48}
							minCardWidth={160}
							minCardHeight={260}
							minCanvasHeight={520}
							ariaLabel="发现内容的不规则满铺布局"
							hasMore={visibleCount < filtered.length}
							onLoadMore={() => (visibleCount = Math.min(visibleCount + 8, filtered.length))}
						>
							{#snippet renderItem(item: FeedItem, index: number)}
								<PostCard
									item={item}
									dense
									liked={item.liked}
									saved={item.saved}
									onLike={() => void interactions.toggleLike(item.id)}
									onSave={() => void interactions.toggleSave(item.id)}
									onClick={() => void goto(ROUTES.moment(item.id))}
								/>
							{/snippet}
						</BspCardCanvas>
					{/if}
				</ContentColumn>
			</div>
		</div>
	</div>
{:else if platform === 'tablet'}
	<!-- ═══ Tablet Discover — 统一 PageHeader + content ═══ -->
	{@render discoverHeader()}

	<ContentColumn maxWidth={nearbyContentMaxWidth(platform)} class="animate-fade-in pt-4 pb-6">
		<div class="min-w-0">
			<div class="mb-5">
				{@render tagChips('overflow-x-auto')}
			</div>

			{#if loading || !filtered.length}
				{@render feedState()}
			{:else}
				<!-- Tablet：BSP 瓦片墙（所有项进统一网格，无首项 hero；对齐 Figma DiscoverPage） -->
				<BspCardCanvas
					items={filtered.slice(0, visibleCount)}
					gap={14}
					minRatio={0.68}
					maxRatio={1.48}
					minCardWidth={160}
					minCardHeight={260}
					minCanvasHeight={520}
					ariaLabel="发现内容的不规则满铺布局"
					hasMore={visibleCount < filtered.length}
					onLoadMore={() => (visibleCount = Math.min(visibleCount + 8, filtered.length))}
				>
					{#snippet renderItem(item: FeedItem, index: number)}
						<PostCard
							item={item}
							dense
							liked={item.liked}
							saved={item.saved}
							onLike={() => void interactions.toggleLike(item.id)}
							onSave={() => void interactions.toggleSave(item.id)}
							onClick={() => void goto(ROUTES.moment(item.id))}
						/>
					{/snippet}
				</BspCardCanvas>
			{/if}
		</div>
	</ContentColumn>
{:else}
	<!-- ═══ Mobile Discover — 统一 PageHeader + 内容列（与附近的人/群组同宽契约）═══ -->
	{@render discoverHeader()}

	<ContentColumn maxWidth={nearbyContentMaxWidth(platform)} class="animate-fade-in pt-4 pb-6">
		<div class="mb-4">
			{@render tagChips('overflow-x-auto')}
		</div>

		{#if loading || !filtered.length}
			{@render feedState()}
		{:else}
			<!-- 2 列 BSP 瓦片墙（所有项进统一网格，无首项 hero；对齐 Figma DiscoverPage columns=2） -->
			<BspCardCanvas
				items={filtered.slice(0, visibleCount)}
				gap={14}
				minRatio={0.68}
				maxRatio={1.48}
				minCardWidth={150}
				minCardHeight={220}
				minCanvasHeight={440}
				ariaLabel="发现内容的不规则满铺布局"
				hasMore={visibleCount < filtered.length}
				onLoadMore={() => (visibleCount = Math.min(visibleCount + 8, filtered.length))}
			>
				{#snippet renderItem(item: FeedItem, index: number)}
					<PostCard
						item={item}
						dense
						liked={item.liked}
						saved={item.saved}
						onLike={() => void interactions.toggleLike(item.id)}
						onSave={() => void interactions.toggleSave(item.id)}
						onClick={() => void goto(ROUTES.moment(item.id))}
					/>
				{/snippet}
			</BspCardCanvas>
		{/if}
	</ContentColumn>
{/if}

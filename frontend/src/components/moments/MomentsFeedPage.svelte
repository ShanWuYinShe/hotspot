<script lang="ts">
	import { Sparkles } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platformStore } from '@/stores/platform.svelte';
	import { loadDefaultLocation } from '@/lib/default-location';
	import BspCardCanvas from '@/components/shared/BspCardCanvas.svelte';
	import PageHeader from '@/components/shared/PageHeader.svelte';
	import TagPill from '@/components/shared/TagPill.svelte';
	import PostCard from '@/components/moments/PostCard.svelte';
	import DiscoverHeader from '@/components/moments/DiscoverHeader.svelte';
	import { createFeedInteractions, type FeedItem } from '@/components/moments/feed-interactions.svelte';
	import { getFeedNearby } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 发现流（首页与 /nearby/moments 共用）：移动单列 / 平板双列 / 桌面
	// BSP 卡片墙。迁移自 React MomentsFeedPage.tsx。

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
		// spec: feed/nearby requires reference_point.
		// reference_point comes from ClientConfig.default_location (location denied fallback).
		const point = referencePoint;
		if (!point) return;
		let cancelled = false;
		async function fetchData() {
			try {
				loading = true;
				const feedRes = await getFeedNearby({ reference_point: point }).catch(() => ({ code: -1, data: { items: [] } }));
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

	// 三端共用的过滤逻辑（React 版每个布局内重复，此处收敛）
	let searchVal = $state('');
	let activeTab = $state('热门');
	let activeTag = $state<string | null>(null);
	let visibleCount = $state(12);

	const interactions = createFeedInteractions(apiFeedItems);

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

{#if platform === 'desktop'}
	<!-- ═══ Desktop Discover — 2-col BSP field ═══ -->
	<div class="content-mode-wide flex flex-col h-full">
		<div class="sticky top-0 z-20 flex-shrink-0 bg-[var(--neu-bg)]" style="border-bottom:1px solid var(--neu-border);">
			<PageHeader title="附近瞬间" subtitle="徐州 · 附近正在发生的事" onBack={() => history.back()} className="px-6 pt-3 pb-2">
				{#snippet actions()}
					<span class="text-[var(--neu-text-muted)] text-xs">{filtered.length} 条内容</span>
				{/snippet}
			</PageHeader>
			<div class="px-6">
				<DiscoverHeader searchVal={searchVal} setSearchVal={(v) => (searchVal = v)} activeTab={activeTab} setActiveTab={(t) => (activeTab = t)} />
			</div>
		</div>

		<div class="flex flex-1 min-h-0 overflow-hidden">
			<!-- Main feed -->
			<div class="flex-1 overflow-y-auto no-scrollbar">
				<div style="padding:20px 24px 28px;">
					<div class="mb-5">
						<div class="flex gap-2 flex-wrap">
							{#each tags as tag}
								<TagPill active={activeTag === tag} onClick={() => (activeTag = activeTag === tag ? null : tag)} size="md">{tag}</TagPill>
							{/each}
						</div>
					</div>

					<BspCardCanvas
						items={filtered.slice(0, visibleCount)}
						gap={24}
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
								edition={String(index + 1).padStart(2, '0')}
								liked={item.liked}
								saved={item.saved}
								onLike={() => void interactions.toggleLike(item.id)}
								onSave={() => void interactions.toggleSave(item.id)}
								onClick={() => void goto(ROUTES.moment(item.id))}
							/>
						{/snippet}
					</BspCardCanvas>
				</div>
			</div>
		</div>
	</div>
{:else if platform === 'tablet'}
	<!-- ═══ Tablet Discover — 2-column grid, max-width 800px centred ═══ -->
	<div class="sticky top-0 z-10 pt-4 pb-0 bg-[var(--neu-bg)]" style="border-bottom:1px solid var(--neu-border);">
		<div class="flex gap-6 mx-auto" style="width:100%;">
			<div class="mx-auto w-full lg:mx-0 px-6 min-w-0" style="width:100%;">
				<div class="flex items-center justify-between mb-2">
					<h1 class="text-[var(--neu-text)] text-2xl">发现</h1>
					<div class="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-[var(--neu-accent-primary)]" style="box-shadow:0 3px 10px var(--neu-glow);">
						<Sparkles style="width:11px;height:11px;color:white;" strokeWidth={2} />
						<span class="text-white text-[11px]">精选</span>
					</div>
				</div>
				<DiscoverHeader searchVal={searchVal} setSearchVal={(v) => (searchVal = v)} activeTab={activeTab} setActiveTab={(t) => (activeTab = t)} />
			</div>
			<div class="hidden lg:block flex-shrink-0" style="width:var(--context-rail-width);"></div>
		</div>
	</div>

	<div class="animate-fade-in flex gap-6 mx-auto" style="max-width:1124px;width:100%;padding:16px 0 28px;">
		<div class="mx-auto w-full lg:mx-0 px-5 min-w-0" style="width:100%;">
			<div class="mb-5">
				<div class="flex gap-2 overflow-x-auto no-scrollbar pb-1">
					{#each tags as tag}
						<TagPill active={activeTag === tag} onClick={() => (activeTag = activeTag === tag ? null : tag)} size="md">{tag}</TagPill>
					{/each}
				</div>
			</div>

			{#if filtered[0]}
				<div class="neu-ambient-glow mb-4">
					<PostCard
						item={filtered[0]}
						hero
						liked={filtered[0].liked}
						saved={filtered[0].saved}
						onLike={() => void interactions.toggleLike(filtered[0].id)}
						onSave={() => void interactions.toggleSave(filtered[0].id)}
						onClick={() => void goto(ROUTES.moment(filtered[0].id))}
					/>
				</div>
			{/if}

			<div style="display:grid;grid-template-columns:repeat(2, 1fr);gap:14px;">
				{#each filtered.slice(1) as item (item.id)}
					<PostCard
						item={item}
						dense
						liked={item.liked}
						saved={item.saved}
						onLike={() => void interactions.toggleLike(item.id)}
						onSave={() => void interactions.toggleSave(item.id)}
						onClick={() => void goto(ROUTES.moment(item.id))}
					/>
				{/each}
			</div>
		</div>
	</div>
{:else}
	<!-- ═══ Mobile Discover — single-column feed, max-width 560px centred ═══ -->
	<div class="sticky top-0 z-10 px-5 pt-3 pb-0 bg-[var(--neu-bg)]" style="border-bottom:1px solid var(--neu-border);">
		<div class="flex items-center justify-between mb-2">
			<h1 class="text-[var(--neu-text)] text-[22px]">发现</h1>
			<span class="text-[var(--neu-text-muted)] text-xs">{filtered.length} 条内容</span>
		</div>
		<DiscoverHeader searchVal={searchVal} setSearchVal={(v) => (searchVal = v)} activeTab={activeTab} setActiveTab={(t) => (activeTab = t)} />
	</div>

	<div class="animate-fade-in" style="max-width:560px;margin-left:auto;margin-right:auto;width:100%;padding:12px 16px 24px;">
		<div class="mb-4">
			<div class="flex gap-2 overflow-x-auto no-scrollbar pb-1">
				{#each tags as tag}
					<TagPill active={activeTag === tag} onClick={() => (activeTag = activeTag === tag ? null : tag)} size="md">{tag}</TagPill>
				{/each}
			</div>
		</div>

		<!-- Feed — first is hero with ambient, rest are flat -->
		<div class="flex flex-col gap-4">
			{#each filtered as item, i (item.id)}
				{#if i === 0}
					<div class="neu-ambient-glow">
						<PostCard
							item={item}
							hero
							liked={item.liked}
							saved={item.saved}
							onLike={() => void interactions.toggleLike(item.id)}
							onSave={() => void interactions.toggleSave(item.id)}
							onClick={() => void goto(ROUTES.moment(item.id))}
						/>
					</div>
				{:else}
					<PostCard
						item={item}
						liked={item.liked}
						saved={item.saved}
						onLike={() => void interactions.toggleLike(item.id)}
						onSave={() => void interactions.toggleSave(item.id)}
						onClick={() => void goto(ROUTES.moment(item.id))}
					/>
				{/if}
			{/each}
		</div>
	</div>
{/if}

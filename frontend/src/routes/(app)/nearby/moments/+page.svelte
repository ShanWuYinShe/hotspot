<script lang="ts">
	import { Map as MapIcon, MapPin, Search, Sparkles } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { platform } from '@/stores/platform.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import BspCardCanvas from '@/components/shared/BspCardCanvas.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import MomentCard from '@/components/moments/MomentCard.svelte';
	import { createFeedInteractions, type FeedItem } from '@/components/moments/moment-feed.svelte';
	import { fetchNearbyMoments , netError } from '@/api';
	import { formatDateTime } from '@/lib/utils';
	import { ROUTES } from '@/lib/routes';
	import { smartBack } from '@/lib/smart-back';
	import {
		getMomentsProgress,
		setMomentsProgress,
		clearMomentsProgress,
		feedProgressKey
	} from './shared/feed-progress.svelte';

	// 附近动态（/nearby/moments）：独立路由页。
	// BSP 卡片墙（桌面 3 列 / 平板 2 列 / 移动 2 列）；人/群组/动态三段各为
	// 独立页面，不再共用段位导航 Tabs；搜索/地图收敛为 header 右上角工具入口。

	// 「周边」= 按距离排序的默认流（nearby feed）；原「附近」与一级 Tab 同名嵌套
	// 易歧义，改用项目既有语汇「周边」。旧深链 ?tab=附近 经 normalizeTab 兼容映射。
	const TABS = ['热门', '周边', '关注'];
	const TAB_ALIASES: Record<string, string> = { 附近: '周边' };

	let apiFeedItems = $state<FeedItem[]>([]);
	let tags = $state<string[]>([]);
	let loading = $state(true);
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(null);
	let locationName = $state('');
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	// 真分页：服务端 cursor（moments/fetch_nearby 返回 next_cursor）翻页追加；
	// cursor 耗尽后由 visibleCount 在已加载集合内切片渲染。
	const PAGE_LIMIT = 20;
	let nextCursor = $state('');
	let loadingMore = $state(false);

	function mapMoment(m: import('@/api/types').MomentItem, i: number): FeedItem {
		const mine = (m.reactions || []).find((r) => r.reacted_by_me);
		return {
			// 缺 moment_id 的项仅在本地作列表 key；点击导航在下方 openMoment 统一校验
			id: m.moment_id ?? `local-${i + 1}`,
			user: m.user?.nickname ?? '',
			avatar: m.user?.avatar_url ?? '',
			location: m.location?.location_name ?? '',
			tag: m.tags?.[0] ?? '',
			content: m.description ?? '',
			image: m.images?.[0]?.url ?? '',
			likes: (m.reactions ?? []).reduce((sum, r) => sum + r.count, 0),
			comments: m.comments_count ?? 0,
			time: formatDateTime(m.created_at),
			liked: Boolean(mine),
			saved: Boolean(m.is_favorited),
			likeEmoji: mine?.emoji ?? '👍'
		};
	}

	$effect(() => startDefaultLocation((loc) => {
referencePoint = loc?.point ?? null;
locationName = loc?.location_name ?? '';
}));

	$effect(() => {
		// spec: moments/fetch_nearby reference_point 可选 — 服务端按回退链解析
		// （显式 point → 最近 update_location → ClientConfig 默认）。
		// 有 point 就传、没有就不传照常请求。
		// 会话进度保持：同一定位 key 下直接采用 store（返回不重拉）；定位变化/
		// 手动重试（先清 store）才走网络。
		void reloadKey;
		const progressKey = feedProgressKey(referencePoint);
		const saved = getMomentsProgress();
		if (saved && (progressKey === '' || saved.key === progressKey)) {
			apiFeedItems = saved.items;
			nextCursor = saved.nextCursor;
			visibleCount = saved.visibleCount;
			tags = saved.tags;
			loadError = null;
			loading = false;
			return;
		}
		let cancelled = false;
		async function fetchData() {
			try {
				loading = true;
				loadError = null;
				nextCursor = '';
				visibleCount = INITIAL_VISIBLE;
				const feedRes = await fetchNearbyMoments({
					...(referencePoint ? { reference_point: referencePoint } : {}),
					pagination: { limit: PAGE_LIMIT }
				}).catch(() => netError({ items: [] as import('@/api/types').MomentItem[], next_cursor: null }));
				if (cancelled) return;

				const feedData = feedRes.code === 0 ? feedRes.data : null;
				if (feedRes.code !== 0) {
					if (!cancelled) loadError = feedRes.message ?? '加载失败';
					return;
				}

				if (feedData) {
					apiFeedItems = (feedData.items || []).map(mapMoment);
					nextCursor = feedData.next_cursor ?? '';
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
					setMomentsProgress({ savedAt: Date.now(), key: progressKey, items: apiFeedItems, nextCursor, visibleCount, tags });
				}
			} catch {
				if (!cancelled) loadError = '加载失败，请稍后重试';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchData();
		return () => {
			cancelled = true;
		};
	});

	// 三端共用的过滤逻辑（收敛到单处实现）
	// tab 支持深链：?tab= 初始化（刷新/分享保留分类），切换写回 URL
	let activeTab = $state(initTabFromUrl());
	let activeTag = $state<string | null>(null);
	// 已加载集合内的渲染切片（cursor 耗尽后 loadMore 推进此值）
	const INITIAL_VISIBLE = 12;
	let visibleCount = $state(INITIAL_VISIBLE);

	// 哨兵触发：优先服务端 cursor 翻页追加（去重），cursor 耗尽后前端切片。
	// 失败不推进（loadingMore 复位后下次触发自然重试），避免静默丢页。
	async function handleLoadMore() {
		if (loadingMore) return;
		if (nextCursor) {
			loadingMore = true;
			try {
				const res = await fetchNearbyMoments({
					...(referencePoint ? { reference_point: referencePoint } : {}),
					pagination: { limit: PAGE_LIMIT, cursor: nextCursor }
				});
				if (res.code === 0 && res.data) {
					const known = new Set(apiFeedItems.map((i) => i.id));
					const fresh = (res.data.items || []).map(mapMoment).filter((i) => !known.has(i.id));
					apiFeedItems = [...apiFeedItems, ...fresh];
					nextCursor = res.data.next_cursor ?? '';
					visibleCount = Math.min(visibleCount + 8, apiFeedItems.length);
					const saved = getMomentsProgress();
					if (saved) setMomentsProgress({ ...saved, items: apiFeedItems, nextCursor, visibleCount });
				}
			} catch {
				// 忽略：下次哨兵触发重试
			} finally {
				loadingMore = false;
			}
			return;
		}
		visibleCount = Math.min(visibleCount + 8, apiFeedItems.length);
		const savedSlice = getMomentsProgress();
		if (savedSlice) setMomentsProgress({ ...savedSlice, visibleCount });
	}

	// 仅服务端 moment_id 可导航；本地合成 id（无 moment_id 的项）不可点进详情
	function openMoment(id: string, focusComments = false) {
		if (id.startsWith('local-')) return;
		void goto(focusComments ? `${ROUTES.moment(id)}?focus=comments` : ROUTES.moment(id));
	}

	function initTabFromUrl(): string {
		const t = page.url.searchParams.get('tab') ?? '';
		const normalized = TAB_ALIASES[t] ?? t;
		return TABS.includes(normalized) ? normalized : '热门';
	}

	// 传入 getter 而非数组快照：interactions.items 需跟随 apiFeedItems 更新
	const interactions = createFeedInteractions(() => apiFeedItems);

	const filtered = $derived.by(() => {
		let items = interactions.items.filter((i) => !activeTag || i.tag === activeTag);
		// 分类 Tab 过滤：热门=按热度排序；关注=关注流（mock 无数据源，恒空）
		if (activeTab === '热门') items = [...items].sort((a, b) => b.likes - a.likes);
		else if (activeTab === '关注') items = [];
		return items;
	});

	const device = $derived(platform.platform);
	const gridColumns = $derived(device === 'mobile' ? 2 : device === 'tablet' ? 2 : 3);
	const minCardWidth = $derived(device === 'mobile' ? 150 : 160);
	const minCardHeight = $derived(device === 'mobile' ? 220 : 260);
	const minCanvasHeight = $derived(device === 'mobile' ? 440 : 520);
</script>
<svelte:head>
	<title>附近动态 · HotSpot</title>
</svelte:head>


<!-- ═══ 三端统一页面 header（标题行 + trailing 工具入口 + below 内容源 Tab）═══ -->
{#snippet feedHeader()}
	<PageHeader
		title="附近"
		subtitle={locationName ? `${locationName} · 5km 内` : '5km 实时探索'}
		sticky
		titleSize={device === 'desktop' ? 'xl' : '2xl'}
		onBack={() => smartBack()}
	>
		{#snippet subtitleIcon()}
			<MapPin class="size-2.5 text-accent-default" strokeWidth={1.75} />
		{/snippet}
		{#snippet trailing()}
			<!-- 搜索/地图是页面级工具，收敛在标题右侧（对齐鸿蒙 NearbyPage trailing 地图入口） -->
			<div class="flex items-center gap-2">
				<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} />
				<ActionIconButton icon={MapIcon} label="地图浏览" onClick={() => void goto(ROUTES.map)} />
			</div>
		{/snippet}
		{#snippet below()}
			<div class="mt-3 pb-2">
				{@render tabBar()}
			</div>
		{/snippet}
	</PageHeader>
{/snippet}

<!-- ═══ 分类 Tab 栏（胶囊样式，随 header 吸顶）═══ -->
{#snippet tabBar()}
	<TabBar
		items={TABS.map((t) => ({ id: t, label: t }))}
		value={activeTab}
		onChange={(id) => {
			activeTab = id;
			void goto(`${ROUTES.nearbyMoments}?tab=${encodeURIComponent(id)}`);
		}}
	/>
{/snippet}

<!-- ═══ 热门 tag 列表（scrollable / flex-wrap 由调用处 className 决定，对齐 Figma TagChips）═══ -->
{#snippet tagChips(className: string)}
	<!-- 标签过滤统一走 TabBar（再次点击已激活项可取消筛选） -->
	<TabBar
		items={tags.map((t) => ({ id: t, label: t }))}
		value={activeTag ?? ''}
		onChange={(id) => (activeTag = activeTag === id ? null : id)}
		class={className}
	/>
{/snippet}

<!-- ═══ 加载中 / 空列表状态（与附近的人、附近群组一致）═══ -->
{#snippet feedState()}
	{#if loading}
		<InlineLoading label="加载附近动态…" />
	{:else if loadError}
		<LoadError message={loadError} onRetry={() => { clearMomentsProgress(); reloadKey++; }} />
	{:else if activeTab === '关注'}
		<!-- 关注流专属空态：与「附近无内容」区分，指引用户先去关注 -->
		<EmptyState title="关注的人还没有发布动态" description="去附近的人页关注感兴趣的用户，动态会出现在这里。">
			{#snippet icon()}
				<Sparkles size={28} color="var(--cst-text-secondary)" class="opacity-subtle" />
			{/snippet}
		</EmptyState>
	{:else}
		<EmptyState title="附近暂时没有瞬间">
			{#snippet icon()}
				<Sparkles size={28} color="var(--cst-text-secondary)" class="opacity-subtle" />
			{/snippet}
		</EmptyState>
	{/if}
{/snippet}

<!-- ═══ 样板 layout：StandardPageLayout（sticky header + ContentColumn 全宽 gutter）═══ -->
<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		{@render feedHeader()}
	{/snippet}

	<div class="mb-5">
		{@render tagChips(device === 'desktop' ? 'flex-wrap' : 'overflow-x-auto pr-1')}
	</div>

	{#if loading || !filtered.length}
		{@render feedState()}
	{:else}
		<!-- BSP 瓦片墙（所有项进统一网格，无首项 hero；列数随平台：桌面 3 / 平板 2 / 移动 2） -->
		<BspCardCanvas
			items={filtered.slice(0, visibleCount)}
			gap={14}
			minRatio={0.68}
			maxRatio={1.48}
			minCardWidth={minCardWidth}
			minCardHeight={minCardHeight}
			minCanvasHeight={minCanvasHeight}
			ariaLabel="附近动态的不规则满铺布局"
			hasMore={loadingMore || visibleCount < filtered.length || Boolean(nextCursor)}
			onLoadMore={() => void handleLoadMore()}
		>
		{#snippet renderItem(item: FeedItem, index: number)}
			<MomentCard
				item={item}
				liked={item.liked}
				saved={item.saved}
				onLike={() => void interactions.toggleLike(item.id)}
				onSave={() => void interactions.toggleSave(item.id)}
				onClick={() => openMoment(item.id)}
				onCommentClick={() => openMoment(item.id, true)}
			/>
		{/snippet}
		</BspCardCanvas>
		{#if !loadingMore && visibleCount >= filtered.length && !nextCursor}
			<ListEnd />
		{/if}
	{/if}
</StandardPageLayout>

<script lang="ts">
	import { Map as MapIcon, MapPin, Search, Sparkles } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { readLastLocation } from '@/lib/map-auto-locate';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import MomentCard from '@/components/moments/MomentCard.svelte';
	import { createFeedInteractions, type FeedItem } from '@/components/moments/moment-feed.svelte';
	import { fetchNearbyMoments, netError } from '@/api';
	import { formatCardTime, formatDistance } from '@/lib/utils';
	import { computeBspLayoutSequence } from '@/lib/semantic-bsp';
	import { ROUTES } from '@/lib/routes';
	import {
		getMomentsProgress,
		setMomentsProgress,
		clearMomentsProgress,
		feedProgressKey
	} from '@/lib/moments-feed-progress.svelte';

	// 附近瞬间（/nearby/moments）：**纯地理轴**页面（按距离排序的 nearby feed）。
	// BSP 卡片墙（桌面 3 列 / 平板 2 列 / 移动 2 列）；搜索/地图收敛为 header 右上角工具入口。
	//
	// 2026-10-19 信息架构调整：本页原有的「热门 / 周边 / 关注」三个分类 Tab 已移除——
	// 热门（算法轴）与关注（关系轴）迁到首页的两个段（首页「发现」/「关注」），
	// 一个表面只承担一条轴；地理叶子页再挂非地理流会让同一批内容出现在两处。
	// 标签筛选（tagChips）保留，仍是本页的筛选维度。

	let apiFeedItems = $state<FeedItem[]>([]);
	let tags = $state<string[]>([]);
	let loading = $state(true);
	// 本机浏览锚点（localStorage 同步可读）优先于服务端默认位置。
	const savedAnchor = readLastLocation();
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(
		savedAnchor ? { latitude: savedAnchor.lat, longitude: savedAnchor.lng } : null
	);
	let locationName = $state(savedAnchor?.name ?? '');
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	// 真分页：服务端 cursor（moments/fetch_nearby 返回 next_cursor）翻页追加；
	// cursor 耗尽后由 visibleCount 在已加载集合内切片渲染。
	const PAGE_LIMIT = 20;
	let nextCursor = $state('');
	let loadingMore = $state(false);

	function mapMoment(m: import('@/api/types').MomentItem, i: number): FeedItem {
		const mine = (m.reactions || []).find((r) => r.reacted_by_me);
		const firstImg = m.images?.[0];
		const w = firstImg?.width ?? null;
		const h = firstImg?.height ?? null;
		return {
			// 缺 moment_id 的项仅在本地作列表 key；点击导航在下方 openMoment 统一校验
			id: m.moment_id ?? `local-${i + 1}`,
			user: m.user?.nickname ?? '',
			userId: m.user?.user_id,
			avatar: m.user?.avatar_url ?? '',
			// 位置拼实时距离(distance 已下发未映射;Gemini 审查 r04:卡片补时空信息)
			location: [m.location?.location_name, formatDistance(m.location?.distance)].filter(Boolean).join(' · '),
			tag: m.tags?.[0] ?? '',
			tags: m.tags ?? [],
			content: m.description ?? '',
			image: firstImg?.url ?? '',
			imageWidth: w,
			imageHeight: h,
			aspectRatio: w && h && h > 0 ? w / h : null,
			likes: (m.reactions ?? []).reduce((sum, r) => sum + r.count, 0),
			comments: m.comments_count ?? 0,
			time: formatCardTime(m.created_at),
			authorRelation: m.author_relation?.is_following ?? false,
			liked: Boolean(mine),
			saved: Boolean(m.is_favorited),
			likeEmoji: mine?.emoji ?? '👍'
		};
	}

	// 浏览锚点优先（spec api_spec_types.md：reference_point = 当前在看哪里/距离基准）：
	// 地图页「设为浏览锚点」写入的 hotspot_last_location 才是本机生效值，
	// 无锚点才回退服务端默认位置。此前只读默认位置，用户在地图上设的锚点
	// 对附近列表毫无影响。
	$effect(() => startDefaultLocation((loc) => {
		if (savedAnchor) return;
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
		// 无定位（progressKey 为空）时不用缓存：空 key 会命中异地旧列表造成串台。
		if (saved && progressKey !== '' && saved.key === progressKey) {
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

	// 传入 getter 而非数组快照：点赞/收藏的乐观更新与回滚作用于当前展示的列表
	const interactions = createFeedInteractions(() => apiFeedItems);

	const filtered = $derived.by(() => {
		// 标签按全量匹配：首标签之外同样可命中（此前只比 tags[0]）。
		return interactions.items.filter((i) => !activeTag || i.tags.includes(activeTag));
	});

	// 列表到底收尾（ListEnd）与哨兵续载（hasMore）
	const showListEnd = $derived(!loadingMore && visibleCount >= filtered.length && !nextCursor);
	const feedHasMore = $derived(loadingMore || visibleCount < filtered.length || Boolean(nextCursor));

	const device = $derived(platform.platform);
	const bspLayouts = $derived(
		computeBspLayoutSequence(
			filtered.slice(0, visibleCount).map((item) => ({
				aspectRatio: item.aspectRatio,
				width: item.imageWidth,
				height: item.imageHeight,
				likes: item.likes,
				hasImage: Boolean(item.image)
			})),
			{ device }
		)
	);
	let sentinelEl = $state<HTMLElement | null>(null);

	$effect(() => {
		if (!sentinelEl) return;
		const observer = new IntersectionObserver(
			(entries) => {
				if (entries[0]?.isIntersecting && feedHasMore && !loadingMore) {
					void handleLoadMore();
				}
			},
			{ rootMargin: '300px' }
		);
		observer.observe(sentinelEl);
		return () => observer.disconnect();
	});
</script>
<svelte:head>
	<!-- 术语统一：全站用「瞬间」，「附近动态」是旧名；页头/导航/空态/加载态保持一致 -->
	<title>附近瞬间 · HotSpot</title>
</svelte:head>


<!-- ═══ 热门 tag 列表（scrollable / flex-wrap 由调用处 className 决定，对齐 Figma TagChips）═══ -->
{#snippet tagChips(className: string)}
	<!-- 标签过滤统一走 TabBar：行首「全部」为默认态（与附近的人/群组筛选语言一致），
	     再次点击已激活标签可取消回「全部」 -->
	<TabBar
		items={[{ id: '', label: '全部' }, ...tags.map((t) => ({ id: t, label: t }))]}
		value={activeTag ?? ''}
		onChange={(id) => (activeTag = id === '' || activeTag === id ? null : id)}
		class={className}
	/>
{/snippet}

<!-- ═══ 三端统一页面 header（标题行 + trailing 工具入口 + below 内容源 Tab）═══ -->
{#snippet feedHeader()}
	<PageHeader
		variant="integrated"
		title="附近瞬间"
		subtitle={locationName ? `${locationName} · 5km 内` : '5km 内 · 实时更新'}
	>
		{#snippet subtitleIcon()}
			<MapPin class="size-2.5 text-accent-default" strokeWidth={1.75} />
		{/snippet}
		{#snippet trailing()}
			<!-- 搜索/地图是页面级工具，收敛在标题右侧（对齐鸿蒙 NearbyPage trailing 地图入口） -->
			<div class="flex items-center gap-2">
				<ActionIconButton variant="ghost" icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} class={device === 'desktop' ? 'hidden' : ''} />
				<ActionIconButton variant="ghost" icon={MapIcon} label="地图浏览" onClick={() => void goto(ROUTES.map)} />
			</div>
		{/snippet}
		{#snippet below()}
			<div>
				{@render tagChips(device === 'desktop' ? 'flex-wrap' : 'pr-1')}
			</div>
		{/snippet}
	</PageHeader>
{/snippet}



<!-- ═══ 加载中 / 空列表状态 ═══ -->
{#snippet feedState()}
	{#if loading}
		<!-- 首屏骨架屏：与真实照片网格同构（移动 2 列、平板 3 列、桌面 4-6 列） -->
		<SkeletonFeed
			variant="card"
			count={device === 'mobile' ? 4 : 8}
			cardClass="aspect-[3/4] rounded-lg"
			class="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-4 lg:gap-4.5 xl:grid-cols-5 2xl:grid-cols-6"
			label="加载附近瞬间…"
		/>
	{:else if loadError}
		<LoadError message={loadError} onRetry={() => { clearMomentsProgress(); reloadKey++; }} />
	{:else}
		<!-- 页面级空态走新版 EmptyState（72px accent 软底图标 + spotlight） -->
		<EmptyState
			spotlight
			iconTone="accent"
			title="附近暂时没有瞬间"
			description="换个位置看看，或成为附近第一个发布瞬间的人。"
		>
			{#snippet icon()}
				<Sparkles class="size-7" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{/if}
{/snippet}

<!-- ═══ 样板 layout：StandardPageLayout（integrated header + ContentColumn 全宽 gutter）═══ -->
<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		{@render feedHeader()}
	{/snippet}

	{#if loading || loadError || !filtered.length}
		{@render feedState()}
	{:else}
		<div class="w-full">
			<!-- 语义化 BSP 照片流网格：移动 2 / 平板 3；lg 起按 300px 最小卡宽自动增列，
			     2560≈7 列、4K≈11 列，全宽利用超宽屏而不是被 max-w 限制在中间。
			     密集排布 (grid-flow-dense) + 自适应行高，感知图片长宽比与节奏防粘连 -->
			<div
				class="grid grid-flow-dense grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-[repeat(auto-fill,minmax(300px,1fr))]"
				style="grid-auto-rows: clamp(195px, 20vw, 275px);"
			>
				{#each filtered.slice(0, visibleCount) as item, index (item.id)}
					{@const layout = bspLayouts[index] ?? { tier: '1x1', cls: 'col-span-1 row-span-1' }}
					<div class="{layout.cls} h-full w-full">
						<MomentCard
							class="h-full w-full rounded-lg"
							{item}
							priority={index < 4}
							compact={device === 'mobile'}
							tier={layout.tier}
							liked={item.liked}
							onLike={() => void interactions.toggleLike(item.id)}
							onClick={() => openMoment(item.id)}
							onCommentClick={() => openMoment(item.id, true)}
						/>
					</div>
				{/each}
			</div>

			{#if feedHasMore}
				<div bind:this={sentinelEl} class="flex justify-center px-4 py-6" aria-hidden="true">
					<InlineLoading label="加载更多…" />
				</div>
			{:else if showListEnd}
				<ListEnd class="pb-6" />
			{/if}
		</div>
	{/if}
</StandardPageLayout>
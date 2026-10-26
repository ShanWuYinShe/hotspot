<script lang="ts">
	import { Map as MapIcon, MapPin, Search, Users } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { LOCATION_ANCHOR_UPDATED_EVENT, clearLastLocation, readLastLocation } from '@/lib/map-auto-locate';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import Button from '@/components/ui/button.svelte';
import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import UserCard, { type NearbyUser } from './UserCard.svelte';
	import { searchUsers, follow , netError } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import { formatDistance } from '@/lib/utils';
	import { showToast } from '@/stores/toast.svelte';
	import { auth, requireAuth } from '@/stores/auth.svelte';
	import { NEARBY_CATEGORIES } from '../shared/categories';
	import {
		getPeopleProgress,
		setPeopleProgress,
		clearPeopleProgress,
		peopleProgressKey
	} from './people-progress.svelte';

	// 附近的人：独立路由页（/nearby/people，也是 /nearby 落点）。
	// 人/群组/动态三段各为独立页面，不再共用段位导航 Tabs（段位导航已删除：
	// 同页多标签形态易把三份不同数据耦合到一起）；搜索/地图收敛为 header
	// 右上角工具入口，跨段切换走桌面侧栏、移动底部导航（附近→默认段）与首页三分流。

	// 「共同好友」排序依赖登录态的 mutual 计数，访客恒为 0 无意义，登录后才展示
	// （Gemini UI 审查：未登录顶栏「登录」与可用「共同好友」并存矛盾）。
	const sortOptions = $derived(auth.user ? ['距离', '在线', '共同好友'] : ['距离', '在线']);
	// 词库统一:原「运动/艺术」在标签池无数据选中必空(对齐 NEARBY_CATEGORIES;Gemini 审查 r03/r04)
	const FILTER_TAGS = NEARBY_CATEGORIES;

	let sort = $state('距离');
	// 登出后 sort 可能残留「共同好友」，此时 TabBar 无匹配项，回落到「距离」
	const effectiveSort = $derived(sortOptions.includes(sort) ? sort : '距离');
	let tagFilter = $state('全部');
	let followedIds = $state<Set<string>>(new Set());
	let visibleCount = $state(12);
	let sentinelEl = $state<HTMLElement | null>(null);
	let allUsers = $state<NearbyUser[]>([]);
	let loading = $state(true);
	// 本机浏览锚点（localStorage 同步可读）优先于服务端默认位置。锚点为响应式：
	// 地图/位置选择更新或清除锚点时经事件同步（见下方监听）。初始值取挂载时刻
	// 一次性快照（state_referenced_locally 警告口径：后续同步只经 applyAnchor）。
	const initialAnchor = readLastLocation();
	let anchor = $state(initialAnchor);
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(
		initialAnchor ? { latitude: initialAnchor.lat, longitude: initialAnchor.lng } : null
	);
	let locationName = $state(initialAnchor?.name ?? (initialAnchor ? '自定义位置' : ''));
	// 服务端默认位置名（空态「改用默认位置」按钮文案用；与基准选择解耦）。
	let defaultLocationName = $state('');
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	// 浏览锚点优先（spec api_spec_types.md：reference_point = 当前在看哪里/距离基准）：
	// 地图页「设为浏览锚点」写入的 hotspot_last_location 才是本机生效值，
	// 无锚点才回退服务端默认位置。此前只读默认位置，用户在地图上设的锚点
	// 对附近列表毫无影响。
	$effect(() => startDefaultLocation((loc) => {
		defaultLocationName = loc?.location_name ?? '';
		if (anchor) return;
		referencePoint = loc?.point ?? null;
		locationName = loc?.location_name ?? '';
	}));

	// 锚点更新/清除监听：地图「设为浏览锚点」与空态「改用默认位置」都会触发，
	// 此处同步基准并重拉列表（进度缓存按定位 key 隔离，手动清除防止串台）。
	function applyAnchor() {
		anchor = readLastLocation();
		clearPeopleProgress();
		if (anchor) {
			referencePoint = { latitude: anchor.lat, longitude: anchor.lng };
			locationName = anchor.name ?? '自定义位置';
		} else {
			referencePoint = null;
			locationName = '';
		}
	}
	$effect(() => {
		window.addEventListener(LOCATION_ANCHOR_UPDATED_EVENT, applyAnchor);
		return () => window.removeEventListener(LOCATION_ANCHOR_UPDATED_EVENT, applyAnchor);
	});

	$effect(() => {
		// spec: users/search radius 仅显式提供时按半径过滤（此处显式传 5km 承接
		// 附近的人语义）；sort_by='distance' 距离计算依赖 reference_point，
		// 可选 — 未传时服务端按回退链解析。
		// 会话进度保持：同一定位 key 下直接采用 store（返回不重拉）；定位变化/
		// 手动重试（先清 store）才走网络。
		void reloadKey;
		const progressKey = peopleProgressKey(referencePoint);
		const saved = getPeopleProgress();
		// 无定位（progressKey 为空）时不用缓存：空 key 会命中异地旧列表造成串台。
		if (saved && progressKey !== '' && saved.key === progressKey) {
			allUsers = saved.users;
			visibleCount = saved.visibleCount;
			// 关注态随列表缓存回填：返回本页时按钮仍反映真实关系。
			followedIds = new Set(saved.users.filter((u) => u.isFollowing).map((u) => u.id));
			loadError = null;
			loading = false;
			return;
		}
		let cancelled = false;
		async function fetchUsers() {
			try {
				loading = true;
				loadError = null;
				const req = {
					radius: 5000,
					sort_by: 'distance' as const,
					...(referencePoint ? { reference_point: referencePoint } : {})
				};
				const res = await searchUsers(req).catch(() => netError({ items: [] }));
				if (cancelled) return;
				const data = res.code === 0 ? res.data : null;
				if (res.code !== 0) {
					if (!cancelled) loadError = res.message ?? '加载失败';
					return;
				}
				if (data) {
					const items: NearbyUser[] = (data.items || []).map((u) => ({
						// spec: users/search → UserItem（user 嵌套 + presence_status + location）
						id: u.user?.user_id ?? '',
						name: u.user?.nickname ?? '',
						distance: formatDistance(u.location?.distance),
						avatar: u.user?.avatar_url ?? '',
						isOnline: u.presence_status === 'online',
						tag: u.tags?.[0] ?? '',
						tags: u.tags ?? [],
						mutual: u.social_context?.mutual_friends_count ?? 0,
						note: u.bio ?? '',
						// 查看者关注态回填：列表初始按钮即反映真实关系（而非一律「关注」）。
						isFollowing: u.relation?.is_following ?? false
					}));
					allUsers = items;
					followedIds = new Set(items.filter((u) => u.isFollowing).map((u) => u.id));
					setPeopleProgress({ key: progressKey, users: items, visibleCount });
				}
			} catch {
				if (!cancelled) loadError = '加载失败，请稍后重试';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchUsers();
		return () => {
			cancelled = true;
		};
	});

	// 过滤 + 排序（按昵称找人走全局搜索：header 右上角搜索入口）
	const list = $derived.by(() => {
		// 自己不在附近的人里露面（对齐鸿蒙客户端；后端不过滤请求者）。
		const selfId = auth.user?.user_id;
		const next = allUsers.filter(
			(user) =>
				(!selfId || user.id !== selfId) &&
				// 标签按全量匹配：首标签之外同样可命中（此前只比 tags[0]）。
				(tagFilter === '全部' || (user.tags?.includes(tagFilter) ?? false))
		);
		if (sort === '在线') return [...next].sort((a, b) => Number(b.isOnline) - Number(a.isOnline));
		if (sort === '共同好友') return [...next].sort((a, b) => (b.mutual ?? 0) - (a.mutual ?? 0));
		return next;
	});

	const isDesktop = $derived(platform.platform === 'desktop');
	const isTablet = $derived(platform.platform === 'tablet');
	const device = $derived(platform.platform);

	// in-flight 去重（防连点双翻转，响应式供按钮 pending 态）；业务失败（code≠0）显式判 code 回滚
	let followPending = $state(new Set<string>());

	// 无限滚动哨兵（内容型列表统一自动加载，对齐 nearby/moments；切片逻辑保留，
	// 每次 +8；过滤/排序变化导致 list 缩短时哨兵不再出现，visibleCount 由切片钳制）。
	// 切片推进同步回进度 store（返回保持已展开长度）。
	$effect(() => {
		if (visibleCount >= list.length) return;
		const el = sentinelEl;
		if (!el) return;
		const observer = new IntersectionObserver(
			(entries) => {
				if (entries.some((e) => e.isIntersecting)) {
					visibleCount = Math.min(visibleCount + 8, list.length);
					const saved = getPeopleProgress();
					if (saved) setPeopleProgress({ ...saved, visibleCount });
				}
			},
			{ root: null, rootMargin: '600px 0px 0px 0px' }
		);
		observer.observe(el);
		return () => observer.disconnect();
	});

	const toggleFollow = async (id: string) => {
		if (followPending.has(id)) return;
		if (!requireAuth({ action: '关注', targetId: id, run: () => void toggleFollow(id) })) return;
		const next = new Set(followedIds);
		const willFollow = !next.has(id);
		willFollow ? next.add(id) : next.delete(id);
		followedIds = next;
		followPending = new Set(followPending).add(id);
		try {
			// spec: users/set_follow 两态统一；乐观更新 + 失败回滚
			const res = await follow({ user_id: id, active: willFollow });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
		} catch (err) {
			const r = new Set(followedIds);
			willFollow ? r.delete(id) : r.add(id);
			followedIds = r;
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			const rest = new Set(followPending);
			rest.delete(id);
			followPending = rest;
		}
	};
</script>
<svelte:head>
	<title>附近的人 · HotSpot</title>
</svelte:head>


	<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
		{#snippet header()}
		<PageHeader
			title="附近的人"
			subtitle={locationName ? `${locationName} · 5km 内` : '5km 内 · 实时更新'}
			sticky
			titleSize={device === 'desktop' ? 'xl' : '2xl'}
		>
			{#snippet subtitleIcon()}
				<MapPin class="size-2.5 text-accent-default" strokeWidth={1.75} />
			{/snippet}
			{#snippet trailing()}
				<!-- 搜索/地图是页面级工具，收敛在标题右侧（对齐鸿蒙 NearbyPage trailing 地图入口） -->
				<div class="flex items-center gap-2">
					<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} class={device === 'desktop' ? 'hidden' : ''} />
					<ActionIconButton icon={MapIcon} label="地图浏览" onClick={() => void goto(ROUTES.map)} />
				</div>
			{/snippet}
			{#snippet below()}
				<!-- 排序：sticky header 内只保留排序；标签过滤放正文首行。
				     桌面端排序并入正文分类行右端合并单行（r02：双行筛选+分割线吃首屏），移动/平板维持 -->
				{#if device !== 'desktop'}
				<div class="mt-3 pb-2">
					<TabBar items={sortOptions.map((o) => ({ id: o, label: o }))} value={effectiveSort} onChange={(id) => (sort = id)} />
				</div>
				{/if}
			{/snippet}
		</PageHeader>
	{/snippet}

	<div class="mb-4 flex items-center justify-between gap-4">
		<TabBar
			items={FILTER_TAGS.map((t) => ({ id: t, label: t }))}
			value={tagFilter}
			onChange={(id) => (tagFilter = id)}
			class={device === 'desktop' ? 'flex-wrap' : 'overflow-x-auto pr-1'}
		/>
		{#if device === 'desktop'}
			<div class="shrink-0">
				<TabBar items={sortOptions.map((o) => ({ id: o, label: o }))} value={effectiveSort} onChange={(id) => (sort = id)} />
			</div>
		{/if}
	</div>

		<div class={isTablet ? 'grid gap-6' : ''}>
			<div class="min-w-0">
				{#if loadError}
					<LoadError message={loadError} onRetry={() => { clearPeopleProgress(); reloadKey++; }} />
				{:else if loading}
					<InlineLoading label="加载附近用户…" />
				{:else if !list.length}
					<!-- 空态自解释：锚点基准下为空必须让用户知道「在看哪里」，否则与
						 首页（同基准）之外的旧认知矛盾（线上实测困惑源）。 -->
					<EmptyState
						bare
						title={anchor ? (anchor.name ? `「${anchor.name}」附近暂时没有用户` : '当前查看的位置附近暂时没有用户') : '附近暂时没有匹配的用户'}
						description={anchor ? '正在按你设置的浏览位置筛选 5km 范围。' : undefined}
					>
						{#snippet icon()}
							<Users size={24} color="var(--cst-text-secondary)" class="opacity-subtle" />
						{/snippet}
						{#snippet action()}
							{#if anchor}
								<Button
									type="button"
									variant="secondary"
									size="sm"
									onclick={() => { clearLastLocation(); applyAnchor(); }}
								>
									改用默认位置{defaultLocationName ? `（${defaultLocationName}）` : ''}</Button>
							{/if}
						{/snippet}
					</EmptyState>
				{:else}
					<div class="grid gap-4 {isTablet ? 'grid-cols-3' : 'grid-cols-2'} lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6">
						{#each list.slice(0, visibleCount) as user (user.id)}
							<UserCard
								{user}
								followed={followedIds.has(user.id)}
								pending={followPending.has(user.id)}
								onFollow={() => toggleFollow(user.id)}
								onNavigate={() => void goto(ROUTES.user(user.id))}
							/>
						{/each}
					</div>

					{#if visibleCount < list.length}
						<div bind:this={sentinelEl} class="flex justify-center px-4 py-6" aria-hidden="true">
							<InlineLoading label="加载更多…" />
						</div>
					{:else}
						<ListEnd class="pb-6" />
					{/if}
				{/if}
			</div>
		</div>
</StandardPageLayout>

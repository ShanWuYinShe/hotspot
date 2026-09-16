<script lang="ts">
	import { Map as MapIcon, MapPin, Search, Users } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import UserCard, { type NearbyUser } from './UserCard.svelte';
	import { searchUsers, follow , netError } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import { formatDistance } from '@/lib/utils';
	import { showToast } from '@/stores/toast.svelte';
	import { requireAuth } from '@/stores/auth.svelte';
	import { smartBack } from '@/lib/smart-back';
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

	const SORT_OPTIONS = ['距离', '在线', '共同好友'];
	const FILTER_TAGS = ['全部', '摄影', '旅行', '音乐', '美食', '运动', '阅读', '艺术'];

	let sort = $state('距离');
	let tagFilter = $state('全部');
	let followedIds = $state<Set<string>>(new Set());
	let visibleCount = $state(12);
	let sentinelEl = $state<HTMLElement | null>(null);
	let allUsers = $state<NearbyUser[]>([]);
	let loading = $state(true);
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(null);
	let locationName = $state('');
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	$effect(() => startDefaultLocation((loc) => {
				referencePoint = loc?.point ?? null;
				locationName = loc?.location_name ?? '';
}));

	$effect(() => {
		// spec: users/search radius 仅显式提供时按半径过滤（此处显式传 5km 承接
		// 附近的人语义）；sort_by='distance' 距离计算依赖 reference_point，
		// 可选 — 未传时服务端按回退链解析。
		// 会话进度保持：同一定位 key 下直接采用 store（返回不重拉）；定位变化/
		// 手动重试（先清 store）才走网络。
		void reloadKey;
		const progressKey = peopleProgressKey(referencePoint);
		const saved = getPeopleProgress();
		if (saved && (progressKey === '' || saved.key === progressKey)) {
			allUsers = saved.users;
			visibleCount = saved.visibleCount;
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
						mutual: u.social_context?.mutual_friends_count ?? 0,
						note: u.bio ?? ''
					}));
					allUsers = items;
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
		const next = allUsers.filter((user) => tagFilter === '全部' || user.tag === tagFilter);
		if (sort === '在线') return [...next].sort((a, b) => Number(b.isOnline) - Number(a.isOnline));
		if (sort === '共同好友') return [...next].sort((a, b) => b.mutual - a.mutual);
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
			title="附近"
			subtitle={locationName ? `${locationName} · 5km 内` : '5km 内 · 实时更新'}
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
				<!-- 排序：sticky header 内只保留排序；标签过滤放正文首行 -->
				<div class="mt-3 pb-2">
					<TabBar items={SORT_OPTIONS.map((o) => ({ id: o, label: o }))} value={sort} onChange={(id) => (sort = id)} />
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	<div class="mb-4">
		<TabBar
			items={FILTER_TAGS.map((t) => ({ id: t, label: t }))}
			value={tagFilter}
			onChange={(id) => (tagFilter = id)}
			class={device === 'desktop' ? 'flex-wrap' : 'overflow-x-auto pr-1'}
		/>
	</div>

		<div class={isTablet ? 'grid gap-6' : ''}>
			<div class="min-w-0">
				{#if loadError}
					<LoadError message={loadError} onRetry={() => { clearPeopleProgress(); reloadKey++; }} />
				{:else if loading}
					<InlineLoading label="加载附近用户…" />
				{:else if !list.length}
					<EmptyState bare title="附近暂时没有匹配的用户">
						{#snippet icon()}
							<Users size={28} color="var(--cst-text-secondary)" class="opacity-subtle" />
						{/snippet}
					</EmptyState>
				{:else}
					<div class="grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
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

<script lang="ts">
	import { Users } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import BspCardCanvas from '@/components/shared/BspCardCanvas.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import UserCard, { type NearbyUser } from './UserCard.svelte';
	import { getNearbyUsers } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import { formatDistance } from '@/lib/utils';
	import { follow } from '@/api';
	import { showToast } from '@/stores/toast.svelte';

	// 附近的人：排序/标签过滤 + BSP 卡片墙（桌面）/ 网格（移动/平板）。

	const SORT_OPTIONS = ['距离', '在线', '共同好友'];
	const FILTER_TAGS = ['全部', '摄影', '旅行', '音乐', '美食', '运动', '阅读', '艺术'];

	let searchQuery = $state('');
	let sort = $state('距离');
	let tagFilter = $state('全部');
	let followedIds = $state<Set<string>>(new Set());
	let visibleCount = $state(12);
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
		// reference_point 可选:有显式锚点就传,否则由服务端回退链解析。
		void reloadKey;
		let cancelled = false;
		async function fetchUsers() {
			try {
				loading = true;
				loadError = null;
				const res = await getNearbyUsers(referencePoint ? { reference_point: referencePoint } : {}).catch(() => ({ code: -1, message: '网络异常', data: { items: [] } }));
				if (cancelled) return;
				const data = res.code === 0 ? res.data : null;
				if (res.code !== 0) {
					if (!cancelled) loadError = res.message ?? '加载失败';
					return;
				}
				if (data) {
					const items: NearbyUser[] = (data.items || []).map((u) => ({
						// spec: feed/nearby_users → UserItem（user 嵌套 + presence_status + location）
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

	// 过滤 + 排序
	const list = $derived.by(() => {
		const next = allUsers.filter(
			(user) =>
				(tagFilter === '全部' || user.tag === tagFilter) &&
				(!searchQuery || `${user.name}${user.tag}${user.note}`.toLowerCase().includes(searchQuery.toLowerCase()))
		);
		if (sort === '在线') return [...next].sort((a, b) => Number(b.isOnline) - Number(a.isOnline));
		if (sort === '共同好友') return [...next].sort((a, b) => b.mutual - a.mutual);
		return next;
	});

	const isDesktop = $derived(platform.platform === 'desktop');
	const isTablet = $derived(platform.platform === 'tablet');
	const device = $derived(platform.platform);
	const gridColumns = $derived(platform.platform === 'mobile' ? 2 : isDesktop ? 3 : 2);

	const toggleFollow = (id: string) => {
		const next = new Set(followedIds);
		const willFollow = !next.has(id);
		willFollow ? next.add(id) : next.delete(id);
		followedIds = next;
		// spec: users/set_follow 两态统一；乐观更新 + 失败回滚
		follow({ user_id: id, active: willFollow }).catch(() => {
			const r = new Set(followedIds);
			willFollow ? r.delete(id) : r.add(id);
			followedIds = r;
			showToast('操作失败，请重试', 'danger');
		});
	};
</script>
<svelte:head>
	<title>附近的人 · HotSpot</title>
</svelte:head>


	<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
		{#snippet header()}
		<PageHeader
			title="附近的人"
			subtitle={locationName ? `${locationName} · 5km 内` : '5km 内'}
			sticky
			titleSize={device === 'desktop' ? 20 : device === 'tablet' ? 24 : 22}
			contentClassName="pt-4 pb-0"
			onBack={() => history.back()}
		>
			{#snippet trailing()}
				<!-- 计数统一在标题右侧（对齐附近群组样板） -->
				<span class="text-text-secondary text-xs">{list.length} 人</span>
			{/snippet}
			{#snippet below()}
				<!-- 搜索框 + 排序：sticky header 内只保留搜索与排序；标签过滤放正文首行
				     （对齐附近动态样板，压缩 header 高度、把空间还给内容） -->
				<div class="mt-3">
					<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索昵称、兴趣或此刻状态" class="px-4 py-2.5 rounded-md" />
				</div>
				<div class="mt-3">
					<TabBar items={SORT_OPTIONS.map((o) => ({ id: o, label: o }))} value={sort} onChange={(id) => (sort = id)} />
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	<div class="mb-5">
		<TabBar
			items={FILTER_TAGS.map((t) => ({ id: t, label: t }))}
			value={tagFilter}
			onChange={(id) => (tagFilter = id)}
			class={device === 'desktop' ? 'flex-wrap' : 'overflow-x-auto pr-1'}
		/>
	</div>

		<div class={isTablet ? 'grid gap-6' : ''}>
			<div class="min-w-0">
				{#if list.length}
				{#if isDesktop || isTablet}
					<BspCardCanvas
						items={list.slice(0, visibleCount)}
						gap={14}
						minRatio={0.68}
						maxRatio={1.48}
						minCardWidth={160}
						minCardHeight={250}
						minCanvasHeight={560}
						ariaLabel="附近用户的不规则满铺布局"
						hasMore={visibleCount < list.length}
						onLoadMore={() => (visibleCount = Math.min(visibleCount + 8, list.length))}
					>
						{#snippet renderItem(user: NearbyUser, index: number)}
							<UserCard
								user={user}
								followed={followedIds.has(user.id)}
								onFollow={() => toggleFollow(user.id)}
								onNavigate={() => void goto(ROUTES.user(user.id))}
							/>
						{/snippet}
					</BspCardCanvas>
				{:else}
					<div class="grid gap-4" style="grid-template-columns:repeat({gridColumns}, minmax(0, 1fr));">
						{#each list as user, index (user.id)}
							<UserCard
								user={user}
								followed={followedIds.has(user.id)}
								onFollow={() => toggleFollow(user.id)}
								onNavigate={() => void goto(ROUTES.user(user.id))}
							/>
						{/each}
					</div>
				{/if}
			{:else if loadError}
				<LoadError message={loadError} onRetry={() => reloadKey++} />
			{:else}
				<EmptyState bare title="没有找到匹配的附近用户">
					{#snippet icon()}
						<Users size={40} color="var(--cst-text-secondary)" class="opacity-subtle" />
					{/snippet}
				</EmptyState>
			{/if}
		</div>
	</div>
</StandardPageLayout>

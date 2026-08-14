<script lang="ts">
	import { MapPin, Users } from '@lucide/svelte';
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
	import GroupCard from './shared/GroupCard.svelte';
	import { mapGroup, type Group } from './shared/group-data';
	import { getNearbyGroups, joinGroup, leaveGroup } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import { ROUTES } from '@/lib/routes';

	// 附近群组：分类过滤 + 精选卡 + 网格 / 桌面 BSP 卡片墙（对齐 Figma Make NearbyGroupsPage）。

	const CATEGORIES = ['全部', '摄影', '音乐', '户外', '阅读', '美食', '健身', '游戏'];

	let activeCategory = $state('全部');
	let searchQuery = $state('');
	let groups = $state<Group[]>([]);
	let visibleCount = $state(12);
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
		// spec: feed/nearby_groups — reference_point 可选,服务端按回退链解析
		// (显式传参 → 最近 update_location → ClientConfig.defaults.default_location)。
		void reloadKey;
		let cancelled = false;
		async function fetchGroups() {
			loading = true;
			loadError = null;
			const res = await getNearbyGroups(
				referencePoint ? { reference_point: referencePoint, pagination: { limit: 24 } } : { pagination: { limit: 24 } }
			).catch(() => ({ code: -1, message: '网络异常', data: { items: [] } }));
			if (cancelled) return;
			const items = res.code === 0 && res.data ? res.data.items : [];
			if (res.code !== 0) {
				if (!cancelled) loadError = res.message ?? '加载失败';
				return;
			}
			groups = items.map(mapGroup);
			loading = false;
		}
		void fetchGroups();
		return () => {
			cancelled = true;
		};
	});

	const isDesktop = $derived(platform.platform === 'desktop');
	const isTablet = $derived(platform.platform === 'tablet');
	const device = $derived(platform.platform);
	const filtered = $derived(
		groups.filter((group) => {
			const q = searchQuery.trim().toLowerCase();
			return (
				(activeCategory === '全部' || group.category === activeCategory) &&
				(!q || `${group.name}${group.desc}${group.category}`.toLowerCase().includes(q))
			);
		})
	);
	const visibleGroups = $derived(filtered.slice(0, visibleCount));
	const featured = $derived(filtered[0]);
	const rest = $derived(filtered.slice(1));
	const totalOnline = $derived(filtered.reduce((sum, group) => sum + group.online, 0));

	// 加入/退出：乐观更新 + 失败回滚（spec: groups/join、groups/leave）。
	// join 对需审核群组返回 pending_approval——UI 仍标记 joined 简化（真实后端
	// 以 is_joined 为准，审核通过前列表不会把它标为已加入）。
	const toggleJoin = (id: string) => {
		const group = groups.find((g) => g.id === id);
		if (!group) return;
		const next = !group.joined;
		groups = groups.map((g) => (g.id === id ? { ...g, joined: next } : g));
		const done = (res: { code: number }) => {
			if (res.code !== 0) {
				groups = groups.map((g) => (g.id === id ? { ...g, joined: !next } : g));
				showToast(next ? '加入失败，请重试' : '退出失败，请重试', 'danger');
			} else {
				showToast(next ? '已加入群组' : '已退出群组', 'success');
			}
		};
		if (next) {
			void joinGroup(id).then(done).catch(() => done({ code: -1 }));
		} else {
			void leaveGroup(id).then(done).catch(() => done({ code: -1 }));
		}
	};
</script>
<svelte:head>
	<title>附近群组 · HotSpot</title>
</svelte:head>



	<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
		{#snippet header()}
			<PageHeader
				title="附近群组"
				subtitle={locationName ? `${locationName} · 5km 内` : '5km 内'}
				sticky
				titleSize={device === 'desktop' ? 20 : device === 'tablet' ? 24 : 22}
				contentClassName="pt-4 pb-0"
				onBack={() => history.back()}
			>
				{#snippet subtitleIcon()}
					<MapPin class="size-2.5 text-accent-default"   strokeWidth={1.75} />
				{/snippet}
				{#snippet trailing()}
					<!-- 计数统一在标题右侧（对齐首页/附近动态样板） -->
					<span class="text-text-secondary text-xs">{filtered.length} 个活跃群组</span>
				{/snippet}
			{#snippet below()}
				<!-- 搜索框（全设备全宽，对齐 Figma） + 分类标签：同处 sticky header 内 -->
				<div class="mt-3">
					<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索群组、活动或兴趣" class="px-4 py-2.5 rounded-md" />
				</div>
				<!-- 分类过滤统一走 TabBar（胶囊） -->
				<div class="mt-3">
					<TabBar items={CATEGORIES.map((c) => ({ id: c, label: c }))} value={activeCategory} onChange={(id) => (activeCategory = id)} />
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	<div class="min-w-0">
		{#if loading}
			<div class="flex flex-col items-center gap-3 py-20">
				<div class="spinner size-7" ></div>
				<p class="text-text-secondary text-sm">加载附近群组…</p>
			</div>
		{:else if loadError}
			<LoadError message={loadError} onRetry={() => reloadKey++} />
		{:else if filtered.length}
			<div class="flex items-center justify-between mb-3">
				<span class="eyebrow">此刻附近群组</span>
				<span class="text-status-online text-xs">{totalOnline} 人正在群聊</span>
			</div>
			{#if isDesktop || isTablet}
				<!-- 精选卡用 GroupCard featured（对齐 Figma），其余用普通网格卡；保留 BSP 布局（决策） -->
				{#if featured}
					<div class="mb-5">
						<GroupCard featured group={featured} onToggle={() => toggleJoin(featured.id)} onOpen={() => void goto(ROUTES.group(featured.id))} />
					</div>
				{/if}
				<BspCardCanvas
					items={rest.slice(0, visibleCount - 1)}
					gap={14}
					minRatio={0.68}
					maxRatio={1.48}
					minCardWidth={160}
					minCardHeight={210}
					minCanvasHeight={420}
					ariaLabel="附近群组的不规则满铺布局"
					hasMore={visibleCount < filtered.length}
					onLoadMore={() => (visibleCount = Math.min(visibleCount + 8, filtered.length))}
				>
					{#snippet renderItem(group: Group, index: number)}
						<GroupCard group={group} onToggle={() => toggleJoin(group.id)} onOpen={() => void goto(ROUTES.group(group.id))} />
					{/snippet}
				</BspCardCanvas>
			{:else}
				{#if featured}
					<div class="mb-5">
						<GroupCard featured group={featured} onToggle={() => toggleJoin(featured.id)} onOpen={() => void goto(ROUTES.group(featured.id))} />
					</div>
				{/if}
				<div class="grid grid-cols-2 gap-4">
					{#each rest as group (group.id)}
						<GroupCard group={group} onToggle={() => toggleJoin(group.id)} onOpen={() => void goto(ROUTES.group(group.id))} />
					{/each}
				</div>
			{/if}
		{:else}
			<EmptyState bare title="没有找到相关群组">
				{#snippet icon()}
					<Users size={40} color="var(--cst-text-secondary)" class="opacity-subtle" />
				{/snippet}
			</EmptyState>
		{/if}
	</div>
</StandardPageLayout>

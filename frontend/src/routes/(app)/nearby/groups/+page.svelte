<script lang="ts">
	import { Map as MapIcon, MapPin, Search, Users, LogOut } from '@lucide/svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import BspCardCanvas from '@/components/shared/BspCardCanvas.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import GroupCard from './shared/GroupCard.svelte';
	import { mapGroup, type Group } from './shared/group-data';
	import { searchGroups, joinGroup, leaveGroup , netError } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import { ROUTES } from '@/lib/routes';
	import { smartBack } from '@/lib/smart-back';

	// 附近群组（/nearby/groups）：独立路由页。分类过滤 + 精选卡 + 网格 / 桌面 BSP 卡片墙；
	// 按名字找群走全局搜索（header 右上角搜索入口）。

	const CATEGORIES = ['全部', '摄影', '音乐', '户外', '阅读', '美食', '健身', '游戏'];

	let activeCategory = $state('全部');
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
		// spec: groups/search radius 仅显式提供时按半径过滤（此处显式传 5km 承接
		// 附近群组语义）；sort_by='distance' 距离计算依赖 reference_point，可选 —
		// 未传时服务端按回退链解析（显式传参 → 最近 update_location → ClientConfig 默认）。
		void reloadKey;
		let cancelled = false;
		async function fetchGroups() {
			loading = true;
			loadError = null;
			try {
				const res = await searchGroups({
					radius: 5000,
					sort_by: 'distance',
					...(referencePoint ? { reference_point: referencePoint } : {}),
					pagination: { limit: 24 }
				}).catch(() => netError({ items: [] }));
				if (cancelled) return;
				if (res.code !== 0) {
					loadError = res.message ?? '加载失败';
					return;
				}
				groups = (res.code === 0 && res.data ? res.data.items : []).map(mapGroup);
			} finally {
				if (!cancelled) loading = false;
			}
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
		groups.filter((group) => activeCategory === '全部' || group.category === activeCategory)
	);
	const visibleGroups = $derived(filtered.slice(0, visibleCount));
	const featured = $derived(filtered[0]);
	const rest = $derived(filtered.slice(1));
	const totalOnline = $derived(filtered.reduce((sum, group) => sum + group.online, 0));

	// 加入/退出：乐观更新 + 失败回滚（spec: groups/enter、groups/exit）。
	// join 对需审核群组返回 pending_approval——UI 仍标记 joined 简化（真实后端
	// 以 is_joined 为准，审核通过前列表不会把它标为已加入）。
	let joinPending = $state(new Set<string>());
	// 退群确认（对齐群详情页的危险操作确认范式）：误触即退群不可接受，
	// 加入仍即时生效（低风险、可随时退）。
	let leaveTarget = $state<Group | null>(null);

	const applyJoin = (id: string, next: boolean) => {
		joinPending = new Set(joinPending).add(id);
		groups = groups.map((g) => (g.id === id ? { ...g, joined: next } : g));
		const done = (res: { code: number }) => {
			const rest = new Set(joinPending);
			rest.delete(id);
			joinPending = rest;
			if (res.code !== 0) {
				groups = groups.map((g) => (g.id === id ? { ...g, joined: !next } : g));
				showToast(next ? '加入失败，请重试' : '退出失败，请重试', 'danger');
			} else {
				showToast(next ? '已加入群组' : '已退出群组', 'success');
			}
		};
		if (next) {
			void joinGroup(id).then(done).catch(() => done(netError(undefined)));
		} else {
			void leaveGroup(id).then(done).catch(() => done(netError(undefined)));
		}
	};

	const toggleJoin = (id: string) => {
		const group = groups.find((g) => g.id === id);
		if (!group || joinPending.has(id)) return;
		if (group.joined) {
			leaveTarget = group;
			return;
		}
		applyJoin(id, true);
	};
</script>
<svelte:head>
	<title>附近群组 · HotSpot</title>
</svelte:head>



	<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
		{#snippet header()}
			<PageHeader
				title="附近"
				subtitle={locationName ? `${locationName} · 5km 内` : '5km 实时探索'}
				sticky
				titleSize={device === 'desktop' ? 'xl' : '3xl'}
				onBack={() => smartBack()}
			>
				{#snippet subtitleIcon()}
					<MapPin class="size-2.5 text-accent-default"   strokeWidth={1.75} />
				{/snippet}
				{#snippet trailing()}
					<!-- 搜索/地图是页面级工具，收敛在标题右侧（对齐鸿蒙 NearbyPage trailing 地图入口） -->
					<div class="flex items-center gap-2">
						<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} />
						<ActionIconButton icon={MapIcon} label="地图浏览" onClick={() => void goto(ROUTES.map)} />
					</div>
				{/snippet}
				{#snippet below()}
					<!-- 分类过滤统一走 TabBar（胶囊），随 header 吸顶 -->
					<div class="mt-3 pb-2">
						<TabBar items={CATEGORIES.map((c) => ({ id: c, label: c }))} value={activeCategory} onChange={(id) => (activeCategory = id)} />
					</div>
				{/snippet}
			</PageHeader>
	{/snippet}

	<div class="min-w-0">
		{#if loading}
			<InlineLoading label="加载附近群组…" />
		{:else if loadError}
			<LoadError message={loadError} onRetry={() => reloadKey++} />
			{:else if filtered.length}
				<div class="flex items-center justify-between mb-3">
					<span class="eyebrow">此刻附近群组</span>
					<span class="text-status-online font-medium text-xs">{filtered.length} 个活跃 · {totalOnline} 人正在群聊</span>
				</div>
			{#if isDesktop || isTablet}
				<!-- 精选卡用 GroupCard featured（对齐 Figma），其余用普通网格卡；保留 BSP 布局（决策） -->
				{#if featured}
					<div class="mb-5">
						<GroupCard featured group={featured} pending={joinPending.has(featured.id)} onToggle={() => toggleJoin(featured.id)} onOpen={() => void goto(ROUTES.group(featured.id))} />
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
						<GroupCard group={group} pending={joinPending.has(group.id)} onToggle={() => toggleJoin(group.id)} onOpen={() => void goto(ROUTES.group(group.id))} />
					{/snippet}
				</BspCardCanvas>
			{:else}
				{#if featured}
					<div class="mb-5">
						<GroupCard featured group={featured} pending={joinPending.has(featured.id)} onToggle={() => toggleJoin(featured.id)} onOpen={() => void goto(ROUTES.group(featured.id))} />
					</div>
				{/if}
				<div class="grid grid-cols-2 gap-4">
					{#each rest as group (group.id)}
						<GroupCard group={group} pending={joinPending.has(group.id)} onToggle={() => toggleJoin(group.id)} onOpen={() => void goto(ROUTES.group(group.id))} />
					{/each}
				</div>
			{/if}
			{#if visibleCount >= filtered.length}
				<ListEnd />
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

<ConfirmDialog
	open={leaveTarget !== null}
	title="退出群组「{leaveTarget?.name ?? ''}」？"
	description="退出后将不再接收该群的聊天与公告。"
	icon={LogOut}
	confirmLabel="确认退出"
	onCancel={() => (leaveTarget = null)}
	onConfirm={() => {
		if (leaveTarget) applyJoin(leaveTarget.id, false);
		leaveTarget = null;
	}}
/>

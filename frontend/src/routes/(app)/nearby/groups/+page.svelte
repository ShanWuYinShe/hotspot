<script lang="ts">
	import { MapPin, Users } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import { loadDefaultLocation } from '@/lib/default-location';
	import BspCardCanvas from '@/components/shared/BspCardCanvas.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import FeaturedGroupCard from './FeaturedGroupCard.svelte';
	import GroupCard from './GroupCard.svelte';
	import { mapGroup, type Group } from './group-data';
	import { getNearbyGroups } from '@/api';
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

	$effect(() => {
		let cancelled = false;
		void loadDefaultLocation().then((loc) => {
			if (!cancelled) {
				referencePoint = loc?.point ?? null;
				locationName = loc?.location_name ?? '';
			}
		});
		return () => {
			cancelled = true;
		};
	});

	$effect(() => {
		// spec: feed/nearby_groups — reference_point 可选,服务端按回退链解析
		// (显式传参 → 最近 update_location → ClientConfig.defaults.default_location)。
		let cancelled = false;
		async function fetchGroups() {
			loading = true;
			const res = await getNearbyGroups(
				referencePoint ? { reference_point: referencePoint, pagination: { limit: 24 } } : { pagination: { limit: 24 } }
			).catch(() => ({ code: -1, data: { items: [] } }));
			if (cancelled) return;
			const items = res.code === 0 && res.data ? res.data.items : [];
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

	// TODO: 接入 groups/join 与 groups/leave（client 包装就绪后）；当前仅本地切换反映按钮意图。
	const toggleJoin = (id: string) => {
		groups = groups.map((group) => (group.id === id ? { ...group, joined: !group.joined } : group));
	};
</script>


	<StandardPageLayout contentClassName="py-5 sm:py-6">
		{#snippet header()}
			<PageHeader title="附近群组" subtitle={locationName ? `${locationName} · 5km 内` : '5km 内'} sticky glass contentClassName="pt-4 pb-0" onBack={() => history.back()}>
				{#snippet subtitleIcon()}
					<MapPin style="width:10px;height:10px;color:var(--accent-default);" strokeWidth={1.8} />
				{/snippet}
				{#snippet trailing()}
					<!-- 计数统一在标题右侧（对齐首页/附近动态样板） -->
					<span class="text-text-secondary text-xs">{filtered.length} 个活跃群组</span>
				{/snippet}
			{#snippet below()}
				<!-- 搜索框（全设备全宽，对齐 Figma） + 分类标签：同处 sticky header 内 -->
				<div class="mt-3">
					<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索群组、活动或兴趣" class="px-3.5 py-2" />
				</div>
				<div class="flex gap-1.5 overflow-x-auto no-scrollbar mt-3">
					{#each CATEGORIES as category}
						<TagPill active={activeCategory === category} onClick={() => (activeCategory = category)}>{category}</TagPill>
					{/each}
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	<div class="min-w-0">
		{#if loading}
			<div class="flex flex-col items-center gap-3 py-20">
				<div class="rounded-full border-2 border-border-default border-t-accent-default animate-spin" style="width:28px;height:28px;"></div>
				<p class="text-text-secondary text-base">加载附近群组…</p>
			</div>
		{:else if filtered.length}
			<div class="flex items-center justify-between mb-3">
				<span class="eyebrow">此刻附近群组</span>
				<span class="text-status-online text-xs">{totalOnline} 人正在群聊</span>
			</div>
			{#if isDesktop || isTablet}
				<!-- index 0 用 FeaturedGroupCard（对齐 Figma），其余用 GroupCard；保留 BSP 布局（决策） -->
				{#if featured}
					<div class="mb-5">
						<FeaturedGroupCard group={featured} onToggle={() => toggleJoin(featured.id)} onOpen={() => void goto(ROUTES.group(featured.id))} />
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
						<FeaturedGroupCard group={featured} onToggle={() => toggleJoin(featured.id)} onOpen={() => void goto(ROUTES.group(featured.id))} />
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
					<Users size={40} color="var(--text-secondary)" opacity={0.4} />
				{/snippet}
			</EmptyState>
		{/if}
	</div>
</StandardPageLayout>

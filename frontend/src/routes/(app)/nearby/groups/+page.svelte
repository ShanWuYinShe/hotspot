<script lang="ts">
	import { Plus, Users } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platformStore } from '@/stores/platform.svelte';
	import { loadDefaultLocation } from '@/lib/default-location';
	import * as Dialog from '@/components/ui/dialog';
	import CreateGroupSheet from '@/components/overlays/CreateGroupSheet.svelte';
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

	// 附近群组：分类过滤 + 精选卡 + 网格 / 桌面 BSP 卡片墙 + 创建群组底部 Sheet。
	// 迁移自旧版 NearbyGroupsPage。

	const CATEGORIES = ['全部', '摄影', '音乐', '户外', '阅读', '美食', '健身', '游戏'];

	let activeCategory = $state('全部');
	let searchQuery = $state('');
	let showCreate = $state(false);
	let groups = $state<Group[]>([]);
	let visibleCount = $state(12);
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
		// spec: feed/nearby_groups — reference_point 可选,服务端按回退链解析
		// (显式传参 → 最近 update_location → ClientConfig.defaults.default_location)。
		let cancelled = false;
		async function fetchGroups() {
			loading = true;
			const res = await getNearbyGroups(
				referencePoint ? { reference_point: referencePoint, pagination: { limit: 24 } } : { pagination: { limit: 24 } }
			).catch(() => ({ code: -1, data: { items: [] } }));
			if (cancelled) return;
			const items = (res.code === 0 && res.data ? (res.data as any).items : []) as any[];
			groups = items.map(mapGroup);
			loading = false;
		}
		void fetchGroups();
		return () => {
			cancelled = true;
		};
	});

	const isDesktop = $derived(platformStore.platform === 'desktop');
	const isTablet = $derived(platformStore.platform === 'tablet');
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

	// TODO: connect to groups/join & groups/leave once client wrappers exist;
	// for now toggle locally so the button reflects intent immediately.
	const toggleJoin = (id: string) => {
		groups = groups.map((group) => (group.id === id ? { ...group, joined: !group.joined } : group));
	};
</script>


<StandardPageLayout
	contentClassName="py-6"
	contentMaxWidth={isTablet ? '964px' : isDesktop ? undefined : '680px'}
>
	{#snippet header()}
		<PageHeader title="附近群组" subtitle={`徐州 · 5km 内 · ${filtered.length} 个活跃群组`} sticky onBack={() => history.back()}>
			{#snippet trailing()}
				<div class="flex items-center gap-2">
					<div class="hidden sm:flex flex-1 max-w-sm">
						<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索群组、活动或兴趣" compact />
					</div>
					<button
						type="button"
						class="neu-hover-btn flex items-center gap-1 px-3 py-1.5 rounded-full neu-focus-ring bg-[var(--neu-accent-primary)] text-white text-xs font-semibold"
						style="box-shadow:0 3px 10px var(--neu-glow);"
						onclick={() => (showCreate = true)}
					>
						<Plus size={14} />
						创建
					</button>
				</div>
			{/snippet}
			{#snippet below()}
				<!-- 分类标签（sticky header 内常驻，对齐 Figma Make NearbyGroupsPage） -->
				<div class="flex gap-1.5 overflow-x-auto no-scrollbar pt-3">
					{#each CATEGORIES as category}
						<TagPill active={activeCategory === category} onClick={() => (activeCategory = category)}>{category}</TagPill>
					{/each}
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if !isDesktop}
		<div class="sm:hidden -mt-2 pb-2">
			<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索群组、活动或兴趣" compact />
		</div>
	{/if}

	<div class="min-w-0">
		{#if loading}
			<div class="flex flex-col items-center gap-3 py-20">
				<div class="rounded-full border-2 border-[var(--neu-border)] border-t-[var(--neu-accent-primary)] animate-spin" style="width:28px;height:28px;"></div>
				<p class="text-[var(--neu-text-muted)] text-[13px]">加载附近群组…</p>
			</div>
		{:else if filtered.length}
			<div class="flex items-center justify-between mb-3">
				<span class="neu-eyebrow">正在附近的群组</span>
				<span class="text-[var(--semantic-online)] text-[11px]">{totalOnline} 人正在群聊</span>
			</div>
			{#if isDesktop || isTablet}
				<BspCardCanvas
					items={visibleGroups}
					gap={24}
					minRatio={0.68}
					maxRatio={1.48}
					minCardWidth={160}
					minCardHeight={210}
					minCanvasHeight={520}
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
				<div class="grid grid-cols-2 gap-3">
					{#each rest as group (group.id)}
						<GroupCard group={group} onToggle={() => toggleJoin(group.id)} onOpen={() => void goto(ROUTES.group(group.id))} />
					{/each}
				</div>
			{/if}
		{:else}
			<EmptyState title="没有找到相关群组">
				{#snippet icon()}
					<Users size={32} color="var(--neu-text-muted)" />
				{/snippet}
			</EmptyState>
		{/if}
	</div>
</StandardPageLayout>

<Dialog.Root open={showCreate} onOpenChange={(next) => (next ? undefined : (showCreate = false))}>
	<Dialog.Content side="bottom" class="h-[92%] overflow-hidden" aria-label="创建群组">
		<CreateGroupSheet onClose={() => (showCreate = false)} />
	</Dialog.Content>
</Dialog.Root>

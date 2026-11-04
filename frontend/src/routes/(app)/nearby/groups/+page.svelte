<script lang="ts">
import { formatCount } from '@/lib/utils';
	import { Map as MapIcon, MapPin, Search, Users, LogOut } from '@lucide/svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { readLastLocation } from '@/lib/map-auto-locate';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import InfiniteSentinel from '@/components/list/InfiniteSentinel.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import AdaptiveCardGrid from '@/components/layout/AdaptiveCardGrid.svelte';
	import SparseContentHint from '@/components/data-display/SparseContentHint.svelte';
import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import GroupCard from '@/components/data-display/GroupCard.svelte';
	import { mapGroup, type Group } from './shared/group-data';
	import { NEARBY_CATEGORIES } from '../shared/categories';
	import { requireAuth } from '@/stores/auth.svelte';
	import { computeBspLayoutSequence } from '@/lib/semantic-bsp';
	import { searchGroups, joinGroup, leaveGroup, cancelGroupApplication, listMyApplications, netError } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import { ROUTES } from '@/lib/routes';

	// 附近群组（/nearby/groups）：独立路由页。分类过滤 + 精选卡 + 网格 / 桌面 BSP 卡片墙；
	// 按名字找群走全局搜索（header 右上角搜索入口）。

	// 词库统一:与附近的人共用一套分类(原「游戏」无数据;Gemini 审查 r03/r04)
	const CATEGORIES = NEARBY_CATEGORIES;

	let activeCategory = $state('全部');
	let groups = $state<Group[]>([]);
	let visibleCount = $state(12);

	let loading = $state(true);
	// 本机浏览锚点（localStorage 同步可读）优先于服务端默认位置。
	const savedAnchor = readLastLocation();
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(
		savedAnchor ? { latitude: savedAnchor.lat, longitude: savedAnchor.lng } : null
	);
	let locationName = $state(savedAnchor?.name ?? '');
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

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
				// 待审态回填：viewer_state 仅成员非 null，申请 pending 时列表看不出——
				// 自查申请列表，让「审核中 · 点击取消」在刷新/外跳返回后依然诚实。
				void listMyApplications().then((mine) => {
					if (cancelled || mine.code !== 0) return;
					const approvals = new Map<string, string>();
					for (const a of mine.data?.items ?? []) {
						if (a.status === 'pending') approvals.set(a.group?.group_id ?? '', a.application_id);
					}
					pendingApprovals = approvals;
				});
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
		// 类目按全量标签匹配：首标签之外同样可命中（此前只比 tags[0]）。
		groups.filter((group) => activeCategory === '全部' || group.tags.includes(activeCategory))
	);
	const visibleGroups = $derived(filtered.slice(0, visibleCount));

	// 无限滚动续载（对齐 nearby/people、nearby/moments）：每次 +8，避免首屏一次渲染
	// 24 条；到底后由 ListEnd 收尾。此前 visibleCount 固定 12 且无加载入口，
	// 返回 13–24 条时被静默截断，用户无法判断「已到底」还是「加载卡住」。
	const hasMoreVisible = $derived(visibleCount < filtered.length);
	function revealMore() {
		if (visibleCount >= filtered.length) return;
		visibleCount = Math.min(visibleCount + 8, filtered.length);
	}

	const bspLayouts = $derived(
		computeBspLayoutSequence(
			visibleGroups.map((g) => ({
				members: g.members,
				isFeatured: g.members > 30 || g.joined,
				likes: g.online,
				// 无 banner 的群不放大到 2x2（否则巨型空画布只显示一个首字母）；
				// avatar 是方形小图，铺满大卡会拉伸，不能作为大卡的视觉依据。
				hasImage: Boolean(g.banner)
			})),
			{ device, kind: 'group' }
		)
	);
	const featured = $derived(filtered[0]);
	const rest = $derived(filtered.slice(1));
	const totalOnline = $derived(filtered.reduce((sum, group) => sum + group.online, 0));

	// 加入/退出：乐观更新 + 失败回滚（spec: groups/enter、groups/exit）。
	// join 对需审核群组返回 pending_approval：不再冒充「已加入」（3cbb2bec 同类），
	// 改入待审态——按钮「审核中 · 点击取消」，点击取消申请，可重新申请。
	// viewer_state 仅成员非 null，申请 pending 时列表看不出，故用
	// listMyApplications 回填（刷新/外跳返回后仍诚实）。
	let joinPending = $state(new Set<string>());
	/** 待审申请表：group_id → application_id */
	let pendingApprovals = $state(new Map<string, string>());
	// 退群确认（对齐群详情页的危险操作确认范式）：误触即退群不可接受，
	// 加入仍即时生效（低风险、可随时退）。
	let leaveTarget = $state<Group | null>(null);

	const applyJoin = (id: string, next: boolean) => {
		joinPending = new Set(joinPending).add(id);
		groups = groups.map((g) => (g.id === id ? { ...g, joined: next } : g));
		const done = (res: { code: number; message?: string; data?: unknown }) => {
			const rest = new Set(joinPending);
			rest.delete(id);
			joinPending = rest;
			if (res.code !== 0) {
				groups = groups.map((g) => (g.id === id ? { ...g, joined: !next } : g));
				showToast(res.message || (next ? '加入失败，请重试' : '退出失败，请重试'), 'danger');
			} else if (next) {
				const joinData = res.data as { status?: string; application_id?: string | null } | null | undefined;
				if (joinData?.status === 'pending_approval') {
					// 待审：纠正乐观的 joined，记录申请 id 供取消（对齐群详情页三态）。
					groups = groups.map((g) => (g.id === id ? { ...g, joined: false } : g));
					pendingApprovals = new Map(pendingApprovals).set(id, joinData.application_id ?? '');
					showToast('申请已提交，待群主审核', 'success');
				} else {
					showToast('已加入群组', 'success');
				}
			} else {
				showToast('已退出群组', 'success');
			}
		};
		if (next) {
			void joinGroup(id).then(done).catch(() => done(netError(undefined)));
		} else {
			void leaveGroup(id).then(done).catch(() => done(netError(undefined)));
		}
	};

	// 取消入群申请（待审态）：成功后回到未加入态，可重新申请（对齐群详情页）。
	const cancelApplication = async (id: string) => {
		const appId = pendingApprovals.get(id);
		if (!appId) return;
		joinPending = new Set(joinPending).add(id);
		const res = await cancelGroupApplication({ group_id: id, application_id: appId }).catch(() => netError(undefined));
		const rest = new Set(joinPending);
		rest.delete(id);
		joinPending = rest;
		if (res.code === 0) {
			const nextApprovals = new Map(pendingApprovals);
			nextApprovals.delete(id);
			pendingApprovals = nextApprovals;
			showToast('已取消申请', 'success');
		} else {
			showToast(res.message || '取消失败，请重试', 'danger');
		}
	};

	const toggleJoin = (id: string) => {
		const group = groups.find((g) => g.id === id);
		if (!group || joinPending.has(id)) return;
		if (pendingApprovals.has(id)) {
			void cancelApplication(id);
			return;
		}
		if (group.joined) {
			leaveTarget = group;
			return;
		}
		// 加入是写操作：访客先登录（登录成功后自动重放），避免直接拿到 10100 技术错误（s2-02）
		if (!requireAuth({ action: '加入群组', targetId: id, run: () => toggleJoin(id) })) return;
		applyJoin(id, true);
	};
</script>
<svelte:head>
	<title>附近群组 · HotSpot</title>
</svelte:head>



	<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
		{#snippet header()}
			<PageHeader
				variant="integrated"
				title="附近群组"
				subtitle={locationName
					? `${locationName} · 5km 内${!loading && !loadError && filtered.length ? ` · ${filtered.length} 个群组` : ''}`
					: `5km 内 · 实时更新${!loading && !loadError && filtered.length ? ` · ${filtered.length} 个群组` : ''}`}
			>
				{#snippet subtitleIcon()}
					<MapPin class="size-2.5 text-accent-default"   strokeWidth={1.75} />
				{/snippet}
				{#snippet trailing()}
					<!-- 在线数是本页状态（保留）；搜索/地图是页面级工具，只在移动/平板出现——
					     桌面顶栏已有搜索、左侧栏 LocationCard 常驻地图入口，再放一份就是重复入口。 -->
					<div class="flex items-center gap-2">
						{#if !loading && !loadError && filtered.length}
							<span class="flex items-center gap-1.5 text-xs tabular-nums text-text-secondary">
								<span class="size-1.5 rounded-full bg-status-online"></span>
								<span class="font-medium text-text-status-online-on-soft">{formatCount(totalOnline)} 人在线</span>
							</span>
						{/if}
						{#if device !== 'desktop'}
							<ActionIconButton variant="ghost" icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} />
							<ActionIconButton variant="ghost" icon={MapIcon} label="地图浏览" onClick={() => void goto(ROUTES.map)} />
						{/if}
					</div>
				{/snippet}
				{#snippet below()}
					<!-- 分类过滤统一走 TabBar（胶囊），随 header 吸顶 -->
					<div>
						<TabBar items={CATEGORIES.map((c) => ({ id: c, label: c }))} value={activeCategory} onChange={(id) => (activeCategory = id)} />
					</div>
				{/snippet}
			</PageHeader>
	{/snippet}

	<div class="min-w-0">
		{#if loading}
			<!-- 首屏骨架屏：图片优先群组卡栅格（移动/平板 2 列，桌面逐级加列） -->
			<SkeletonFeed
				variant="card"
				count={8}
				cardClass="h-full"
				class="grid grid-flow-dense grid-cols-2 gap-3 px-1 sm:grid-cols-3 sm:gap-3.5 lg:grid-cols-[repeat(auto-fill,minmax(18.75rem,1fr))] [grid-auto-rows:clamp(12.1875rem,20vw,17.1875rem)]"
				label="加载附近群组…"
			/>
		{:else if loadError}
			<LoadError message={loadError} onRetry={() => reloadKey++} />
			{:else if filtered.length}
			<div class="w-full">
				<!-- 语义化 BSP 群组网格：移动 2 列 / 平板 3 列；lg 起按 300px 最小卡宽自动增列，
				     2560≈7 列、4K≈11 列，避免固定 6 列在超宽屏把卡片拉成巨幅。
				     密集排布 (grid-flow-dense) + 自适应行高，全量卡片周期性穿插。 -->
				<AdaptiveCardGrid
					count={visibleGroups.length}
					minTrackRem={18.75}
					gapRem={[0.75, 0.875, 0.875]}
					centerSparse={bspLayouts.slice(0, visibleGroups.length).every((l) => l.tier === '1x1')}
					class="grid-flow-dense px-1"
					style="grid-auto-rows: clamp(12.1875rem, 20vw, 17.1875rem);"
				>
					{#each visibleGroups as group, index (group.id)}
						{@const layout = bspLayouts[index] ?? { tier: '1x1', cls: 'col-span-1 row-span-1', horizontal: false, featured: false }}
						<div class={layout.cls}>
							<GroupCard
								{group}
								priority={index < 4}
								featured={layout.featured}
								horizontal={layout.horizontal}
								tier={layout.tier}
								pending={joinPending.has(group.id)}
								pendingApproval={pendingApprovals.has(group.id)}
								onToggle={() => toggleJoin(group.id)}
								onOpen={() => void goto(ROUTES.group(group.id))}
								class="h-full w-full"
							/>
						</div>
					{/each}
				</AdaptiveCardGrid>
				{#if filtered.length > 0 && filtered.length < 3}
					<SparseContentHint
						title={`只找到 ${filtered.length} 个附近群组`}
						description="换个分类看看，或去附近发现更多圈子"
						actionLabel="去附近看看"
						onAction={() => void goto(ROUTES.nearby)}
					/>
				{/if}
			</div>
			{#if hasMoreVisible}
				<!-- 与附近的人/瞬间页统一：滚到底自动续载 + 可见 InlineLoading 作加载暗示 -->
				<div class="flex justify-center px-4 py-6">
					<InfiniteSentinel hasMore={hasMoreVisible} loading={false} onLoadMore={revealMore} />
					<InlineLoading label="加载更多…" />
				</div>
			{:else}
				<ListEnd />
			{/if}
		{:else}
			<EmptyState
				spotlight
				iconTone="accent"
				title="没有找到相关群组"
				description="换个分类看看，或去全景地图发现身边的圈子。"
			>
				{#snippet icon()}
					<Users class="size-7" strokeWidth={1.5} />
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

<script lang="ts">
import { formatCount } from '@/lib/utils';
	import { Map as MapIcon, MapPin, Search, Users, LogOut } from '@lucide/svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { readLastLocation } from '@/lib/map-auto-locate';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import BspCardCanvas from '@/components/shared/BspCardCanvas.svelte';
	import type { BspRect } from '@/lib/bsp-layout';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import GroupCard from './shared/GroupCard.svelte';
	import { mapGroup, type Group } from './shared/group-data';
	import { NEARBY_CATEGORIES } from '../shared/categories';
	import { requireAuth } from '@/stores/auth.svelte';
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
				title="附近群组"
				subtitle={locationName ? `${locationName} · 5km 内` : '5km 内 · 实时更新'}
				sticky
				titleSize={device === 'desktop' ? 'xl' : '2xl'}
			>
				{#snippet subtitleIcon()}
					<MapPin class="size-2.5 text-accent-default"   strokeWidth={1.75} />
				{/snippet}
				{#snippet trailing()}
					<!-- 搜索/地图是页面级工具，收敛在标题右侧（对齐鸿蒙 NearbyPage trailing 地图入口） -->
					<div class="flex items-center gap-2">
						<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} class={device === 'desktop' ? 'hidden' : ''} />
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
					<span class="text-status-online font-medium text-xs">{filtered.length} 个活跃 · {formatCount(totalOnline)} 人正在群聊</span>
				</div>
			{#if isDesktop && rest.length > 3}
			<!-- 精选卡用 GroupCard featured 横排（图片左 40%,r03:纵排大卡吃首屏）；
			     其余 BSP 满铺墙（高格子自动横排：图左 40% + 信息右面板），
			     信息沉底观感由 GroupCard autoLandscape 兜底（横格>420px 自动切横排） -->
			{#if featured}
				<div class="mb-6">
					<GroupCard featured horizontal group={featured} pending={joinPending.has(featured.id)} pendingApproval={pendingApprovals.has(featured.id)} onToggle={() => toggleJoin(featured.id)} onOpen={() => void goto(ROUTES.group(featured.id))} />
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
				{#snippet renderItem(group: Group, index: number, rect: BspRect)}
					<GroupCard group={group} cardWidth={rect.width} cardHeight={rect.height} pending={joinPending.has(group.id)} pendingApproval={pendingApprovals.has(group.id)} onToggle={() => toggleJoin(group.id)} onOpen={() => void goto(ROUTES.group(group.id))} />
				{/snippet}
			</BspCardCanvas>
			{:else}
				{#if featured}
					<div class="mb-6">
						<GroupCard featured horizontal={isDesktop || isTablet} group={featured} pending={joinPending.has(featured.id)} pendingApproval={pendingApprovals.has(featured.id)} onToggle={() => toggleJoin(featured.id)} onOpen={() => void goto(ROUTES.group(featured.id))} />
					</div>
				{/if}
				<!-- 手机/平板：规整 2 列（平板 BSP 会出 1 窄 + 2 宽 的尺寸落差与底部参差）；
				     桌面保持「精选卡 + BSP 满铺墙」 -->
				<div class="grid grid-cols-2 gap-4 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
					{#each rest as group (group.id)}
						<GroupCard group={group} pending={joinPending.has(group.id)} pendingApproval={pendingApprovals.has(group.id)} onToggle={() => toggleJoin(group.id)} onOpen={() => void goto(ROUTES.group(group.id))} />
					{/each}
				</div>
			{/if}
			{#if visibleCount >= filtered.length}
				<ListEnd />
			{/if}
		{:else}
			<EmptyState bare title="没有找到相关群组">
				{#snippet icon()}
					<Users size={24} color="var(--cst-text-secondary)" class="opacity-subtle" />
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

<script lang="ts">
	import { adminStatsGeo, adminStatsGeoDrilldown } from '@/api';
	import { netError } from '@/api';
	import type { AdminGeoStatsResponse, AdminGeoDrilldownItem } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { platform } from '@/stores/platform.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import MetricRow from '@/components/data-display/MetricRow.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import MapCanvas from '@/components/map/MapCanvas.svelte';
	import MapHeatDot from '@/components/map/MapHeatDot.svelte';
	import type { MapBounds } from '@/components/map/types';
	import { TABLE_WRAP, TABLE, TH, TR, TD } from '../shared/table';
	import { scrollFade } from '@/lib/scroll-fade';
	import { formatDateTime } from '@/lib/utils';

	// 地理统计：全站存量 + 每日新增趋势 + S2 网格区域排名（可下钻明细）+
	// 热门地点排名 + 区域落点地图（spec api_spec_admin.md「统计分析（stats）API」）。
	// 粒度/时间窗/趋势切换即重新拉取；后端为实时聚合快照（弱一致）。

	const canRead = $derived(hasAdminPermission('admin.stats.read'));
	const device = $derived(platform.platform);

	const LEVELS = [
		{ id: '4', label: '国家' },
		{ id: '6', label: '大区' },
		{ id: '8', label: '城市' },
		{ id: '10', label: '城区' },
		{ id: '12', label: '街区' }
	];
	const RANGES = [
		{ id: '', label: '不限时间' },
		{ id: '7d', label: '近 7 天' },
		{ id: '30d', label: '近 30 天' }
	];
	const TRENDS = [
		{ id: '0', label: '趋势关' },
		{ id: '7', label: '趋势 7 天' },
		{ id: '30', label: '趋势 30 天' }
	];
	const DRILL_TYPES = [
		{ id: 'user', label: '用户' },
		{ id: 'group', label: '群组' },
		{ id: 'moment', label: '动态' }
	];

	let level = $state(8);
	let range = $state('');
	let trendDays = $state(0);
	let data = $state<AdminGeoStatsResponse | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let reloadKey = $state(0);

	function rangeToTimeRange(key: string): { start_time: string | null; end_time: string | null } | null {
		if (!key) return null;
		const days = key === '7d' ? 7 : 30;
		const end = new Date();
		const start = new Date(end.getTime() - days * 24 * 60 * 60 * 1000);
		return { start_time: start.toISOString(), end_time: end.toISOString() };
	}

	const fmt = (n: number): string => n.toLocaleString('zh-CN');
	const coord = (v: number): string => v.toFixed(3);

	$effect(() => {
		void level;
		void range;
		void trendDays;
		void reloadKey;
		let cancelled = false;
		async function run() {
			loading = true;
			error = null;
			const res = await adminStatsGeo({
				level,
				time_range: rangeToTimeRange(range),
				limit: 50,
				trend_days: trendDays
			}).catch(() => netError<AdminGeoStatsResponse>({
				generated_at: '', totals: { user: 0, group: 0, moment: 0, total: 0 },
				regions: [], places: [], trends: []
			}));
			if (cancelled) return;
			if (res.code === 0 && res.data) {
				data = res.data;
				fitMapToRegions(res.data.regions);
				if (selectedToken) {
					const current = res.data.regions.find((x) => x.cell_token === selectedToken);
					if (!current || current.level !== selectedLevel) clearDrillSelection();
				}
			} else {
				data = null;
				error = res.message ?? '加载失败';
			}
			loading = false;
		}
		void run();
		return () => {
			cancelled = true;
		};
	});

	// === 地图视图 ===
	// center/zoom 由页面按数据拟合（初始 + 粒度/时间变化时重拟合），bounds 由
	// MapCanvas bind 回报真实视口——落点必须用画布回报的 bounds 定位：
	// 国家尺度上线性百分比仅对「画布真实视口」成立（MapCanvas 契约注释）。
	let mapCenter = $state<{ lat: number; lng: number }>({ lat: 35.0, lng: 105.0 });
	let mapZoom = $state(4);
	let mapBounds = $state<MapBounds>({ minLat: 15, maxLat: 55, minLng: 75, maxLng: 135 });

	function fitMapToRegions(regions: AdminGeoStatsResponse['regions']): void {
		if (regions.length === 0) {
			mapCenter = { lat: 35.0, lng: 105.0 };
			mapZoom = 4;
			return;
		}
		const lats = regions.map((r) => r.center.latitude);
		const lngs = regions.map((r) => r.center.longitude);
		const minLat = Math.min(...lats);
		const maxLat = Math.max(...lats);
		const minLng = Math.min(...lngs);
		const maxLng = Math.max(...lngs);
		mapCenter = { lat: (minLat + maxLat) / 2, lng: (minLng + maxLng) / 2 };
		// 视口宽度估算（桌面主列 ~900px / 移动 ~350px），Mercator 纬度方向再放宽一档。
		const viewW = device === 'mobile' ? 350 : 900;
		const lngSpan = Math.max(6, maxLng - minLng) * 1.5;
		const latSpan = Math.max(6, maxLat - minLat) * 1.5;
		const zoomLng = Math.log2((viewW * 360) / (lngSpan * 256));
		const zoomLat = Math.log2((viewW * 180) / (latSpan * 256));
		mapZoom = Math.max(3, Math.min(10, Math.floor(Math.min(zoomLng, zoomLat))));
	}

	// === 下钻 ===
	let selectedToken = $state<string | null>(null);
	let selectedLevel = $state(8);
	let drillType = $state<'user' | 'group' | 'moment'>('user');
	let drillItems = $state<AdminGeoDrilldownItem[]>([]);
	let drillCursor = $state<string | null>(null);
	let drillLoading = $state(false);
	let drillError = $state<string | null>(null);
	let drillSeq = 0;

	function clearDrillSelection(): void {
		selectedToken = null;
		drillItems = [];
		drillCursor = null;
		drillError = null;
	}

	function selectRegion(cellToken: string, lvl: number): void {
		if (selectedToken === cellToken) {
			clearDrillSelection();
			return;
		}
		selectedToken = cellToken;
		selectedLevel = lvl;
		drillType = 'user';
		drillItems = [];
		drillCursor = null;
		drillError = null;
	}

	async function loadDrill(append: boolean): Promise<void> {
		if (!selectedToken) return;
		const seq = ++drillSeq;
		drillLoading = true;
		if (!append) drillItems = [];
		try {
			const res = await adminStatsGeoDrilldown({
				cell_token: selectedToken,
				level: selectedLevel,
				entity_type: drillType,
				time_range: rangeToTimeRange(range),
				pagination: { limit: 20, cursor: append ? (drillCursor ?? undefined) : undefined }
			});
			if (seq !== drillSeq) return;
			if (res.code === 0 && res.data) {
				drillItems = append ? [...drillItems, ...(res.data.items ?? [])] : (res.data.items ?? []);
				drillCursor = res.data.next_cursor ?? null;
				drillError = null;
			} else {
				drillError = res.message ?? '加载失败';
			}
		} catch {
			if (seq !== drillSeq) return;
			drillError = '加载失败';
		} finally {
			if (seq === drillSeq) drillLoading = false;
		}
	}

	// 选中区域/类型/时间窗变化 → 重新拉第一页。
	$effect(() => {
		void selectedToken;
		void selectedLevel;
		void drillType;
		void range;
		if (!selectedToken) return;
		void loadDrill(false);
	});

	const trendMax = $derived(Math.max(1, ...(data?.trends ?? []).map((p) => p.user + p.group + p.moment)));
	const trendLabel = (d: string): string => d.slice(5).replace('-', '/');
	const selectedRegionLabel = $derived.by(() => {
		if (!selectedToken || !data) return '';
		const r = data.regions.find((x) => x.cell_token === selectedToken);
		if (!r) return '';
		return coord(r.center.latitude) + '°, ' + coord(r.center.longitude) + '°';
	});
</script>

<svelte:head>
	<title>地理统计 · HotSpot</title>
</svelte:head>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.stats.read 权限" bare />
{:else}
	<div class="mx-auto flex w-full max-w-5xl flex-col gap-4">
		<!-- 全站存量：指标统一走 MetricRow 单卡（原 4 张独立小卡 = 4 套边框/间距） -->
		<section aria-label="全站存量">
			{#if data}
				<MetricRow
					class="max-w-3xl"
					items={[
						{ label: '用户', value: fmt(data.totals.user) },
						{ label: '群组', value: fmt(data.totals.group) },
						{ label: '动态', value: fmt(data.totals.moment) },
						{ label: '合计', value: fmt(data.totals.total), tone: 'accent' }
					]}
				/>
			{:else if loading}
				<InlineLoading label="统计中…" />
			{/if}
		</section>

		<!-- 筛选：粒度 + 时间窗 + 趋势 -->
		<section class="flex flex-col gap-2" aria-label="筛选">
			<TabBar
				items={LEVELS}
				value={String(level)}
				onChange={(id) => { level = Number(id); }}
			/>
			<div class="flex flex-wrap gap-2">
				<TabBar
					items={RANGES}
					value={range}
					onChange={(id) => { range = id; }}
				/>
			</div>
			<div class="flex flex-wrap gap-2">
				<TabBar
					items={TRENDS}
					value={String(trendDays)}
					onChange={(id) => { trendDays = Number(id); }}
				/>
			</div>
		</section>

		{#if loading}
			<InlineLoading label="统计中…" />
		{:else if error}
			<LoadError message={error} onRetry={() => { reloadKey += 1; }} />
		{:else if data}
			<p class="text-xs text-text-secondary">统计生成于 {formatDateTime(data.generated_at)} · 弱一致快照</p>

			<!-- 每日新增趋势 -->
			{#if trendDays > 0 && data.trends.length > 0}
				<section aria-label="每日新增趋势" class="flex flex-col gap-2">
					<h2 class="text-md font-semibold text-text-primary">每日新增（近 {trendDays} 天 · UTC+8 日界）</h2>
					<div class="rounded-lg bg-surface-island p-4 shadow-xs">
						<div class="flex items-end gap-1 h-28" role="img" aria-label="每日新增柱状图">
							{#each data.trends as p (p.date)}
								{@const total = p.user + p.group + p.moment}
								{@const h = Math.round((total / trendMax) * 100)}
								<div class="flex-1 flex flex-col items-center gap-1 min-w-0">
									<div class="w-full flex items-end justify-center" style="height: 88px;">
										<div
											class="w-full max-w-8 rounded-t-sm bg-accent-default"
											style="height: {Math.max(total > 0 ? 4 : 1, h)}%;"
											title="{p.date}：用户 {p.user} · 群组 {p.group} · 动态 {p.moment}"
										></div>
									</div>
									<span class="text-xxs text-text-secondary">{trendLabel(p.date)}</span>
								</div>
							{/each}
						</div>
					</div>
				</section>
			{/if}

			<!-- 区域落点地图 -->
			<section aria-label="区域分布图" class="flex flex-col gap-2">
				<h2 class="text-md font-semibold text-text-primary">区域分布（网格中心落点 · 数字为聚合量）</h2>
				<div class="relative rounded-lg overflow-hidden shadow-xs" style="height: 320px;">
					<MapCanvas bind:center={mapCenter} bind:zoom={mapZoom} bind:bounds={mapBounds} />
					{#each data.regions as r (r.cell_token)}
						<MapHeatDot
							bucket={{ cell_token: r.cell_token, level: r.level, location: { latitude: r.center.latitude, longitude: r.center.longitude }, total_count: r.total_count, breakdown: { user: r.breakdown.user, group: r.breakdown.group, moment: r.breakdown.moment } }}
							bounds={mapBounds}
							onSelect={(bucket) => selectRegion(bucket.cell_token, r.level)}
						/>
					{/each}
				</div>
			</section>

			<!-- 区域排名 -->
			<section aria-label="区域排名" class="flex flex-col gap-2">
				<h2 class="text-md font-semibold text-text-primary">区域排名（S2 网格 · {LEVELS.find((l) => Number(l.id) === level)?.label ?? level}）<span class="ml-2 text-xs font-normal text-text-secondary">点击“明细”查看该网格</span></h2>
				{#if data.regions.length === 0}
					<EmptyState title="暂无地理数据" description="当前筛选条件下没有带位置的数据" bare />
				{:else}
					<div class={TABLE_WRAP} use:scrollFade>
						<table class={TABLE}>
							<thead>
								<tr>
									<th scope="col" class={TH}>#</th>
									<th scope="col" class={TH}>网格中心</th>
									<th scope="col" class={TH}>用户</th>
									<th scope="col" class={TH}>群组</th>
									<th scope="col" class={TH}>动态</th>
									<th scope="col" class={TH}>合计</th>
									<th scope="col" class={TH}><span class="sr-only">操作</span></th>
								</tr>
							</thead>
							<tbody>
								{#each data.regions as r, i (r.cell_token)}
									<tr class={TR} class:bg-interactive-hover={selectedToken === r.cell_token}>
										<td class={TD}>{i + 1}</td>
										<td class={TD} title={r.cell_token}>{coord(r.center.latitude)}°, {coord(r.center.longitude)}°</td>
										<td class={TD}>{fmt(r.breakdown.user)}</td>
										<td class={TD}>{fmt(r.breakdown.group)}</td>
										<td class={TD}>{fmt(r.breakdown.moment)}</td>
										<td class={TD + ' font-medium'}>{fmt(r.total_count)}</td>
										<td class={TD}>
											<button
												type="button"
												class="hover-btn tap-92 rounded-md bg-surface-subtle/60 px-2.5 py-1 text-xs font-medium text-text-primary focus-ring transition-colors hover:bg-surface-subtle active:scale-95"
												aria-pressed={selectedToken === r.cell_token}
												onclick={() => selectRegion(r.cell_token, r.level)}
											>
												{selectedToken === r.cell_token ? '已选' : '明细'}
											</button>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>

			<!-- 下钻明细 -->
			{#if selectedToken}
				<section aria-label="区域明细" class="flex flex-col gap-2 rounded-lg bg-surface-island p-4 shadow-xs">
					<div class="flex items-center justify-between gap-2">
						<h3 class="text-sm font-medium text-text-primary">网格明细 · {selectedRegionLabel}</h3>
						<button
							type="button"
							class="hover-btn tap-92 rounded-md bg-surface-subtle/60 px-2.5 py-1 text-xs font-medium text-text-primary focus-ring transition-colors hover:bg-surface-subtle active:scale-95"
							onclick={() => { selectedToken = null; }}
						>关闭</button>
					</div>
					<TabBar
						items={DRILL_TYPES}
						value={drillType}
						onChange={(id) => { drillType = id as 'user' | 'group' | 'moment'; }}
					/>
					{#if drillLoading && drillItems.length === 0}
						<InlineLoading label="加载明细…" />
					{:else if drillError}
						<LoadError message={drillError} onRetry={() => void loadDrill(false)} />
					{:else if drillItems.length === 0}
						<EmptyState title="该网格暂无此类数据" bare />
					{:else}
						<div class="flex flex-col">
							{#each drillItems as item (item.object_id + item.created_at)}
								<div class="flex items-center justify-between gap-3 py-2 border-b border-border-default last:border-b-0">
									<div class="min-w-0">
										<p class="text-sm font-medium text-text-primary truncate">{item.title}</p>
										<p class="mt-0.5 text-xs text-text-secondary truncate">
											{item.location_name ?? '未填地点'} · {formatDateTime(item.created_at)}
										</p>
									</div>
									<span class="text-xxs text-text-secondary font-mono shrink-0">{item.object_id}</span>
								</div>
							{/each}
						</div>
						{#if drillCursor}
							<LoadMoreButton
								loading={drillLoading}
								onLoad={() => void loadDrill(true)}
							/>
						{:else}
							<ListEnd />
						{/if}
					{/if}
				</section>
			{/if}

			<!-- 热门地点 -->
			<section aria-label="热门地点" class="flex flex-col gap-2">
				<h2 class="text-md font-semibold text-text-primary">热门地点（按地点名聚合）</h2>
				{#if data.places.length === 0}
					<EmptyState title="暂无地点数据" description="当前筛选条件下没有非空地点名" bare />
				{:else}
					<div class={TABLE_WRAP} use:scrollFade>
						<table class={TABLE}>
							<thead>
								<tr>
									<th scope="col" class={TH}>#</th>
									<th scope="col" class={TH}>地点</th>
									<th scope="col" class={TH}>用户</th>
									<th scope="col" class={TH}>群组</th>
									<th scope="col" class={TH}>动态</th>
									<th scope="col" class={TH}>合计</th>
								</tr>
							</thead>
							<tbody>
								{#each data.places as p, i (p.location_name)}
									<tr class={TR}>
										<td class={TD}>{i + 1}</td>
										<td class={TD + ' font-medium'}>{p.location_name}</td>
										<td class={TD}>{fmt(p.breakdown.user)}</td>
										<td class={TD}>{fmt(p.breakdown.group)}</td>
										<td class={TD}>{fmt(p.breakdown.moment)}</td>
										<td class={TD + ' font-medium'}>{fmt(p.total_count)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>
		{/if}
	</div>
{/if}
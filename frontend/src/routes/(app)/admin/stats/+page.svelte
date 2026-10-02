<script lang="ts">
	import { adminStatsGeo } from '@/api';
	import { netError } from '@/api';
	import type { AdminGeoStatsResponse } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { platform } from '@/stores/platform.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import { TABLE_WRAP, TABLE, TH, TR, TD } from '../shared/table';
	import { formatDateTime } from '@/lib/utils';

	// 地理统计：全站存量 + S2 网格区域排名 + 热门地点排名（spec api_spec_admin.md「统计分析（stats）API」）。
	// 粒度/时间窗切换即重新拉取；后端为实时聚合快照（弱一致）。

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

	let level = $state(8);
	let range = $state('');
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
		void reloadKey;
		let cancelled = false;
		async function run() {
			loading = true;
			error = null;
			const res = await adminStatsGeo({ level, time_range: rangeToTimeRange(range), limit: 50 })
				.catch(() => netError<AdminGeoStatsResponse>({ generated_at: '', totals: { user: 0, group: 0, moment: 0, total: 0 }, regions: [], places: [] }));
			if (cancelled) return;
			if (res.code === 0 && res.data) {
				data = res.data;
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
</script>

<svelte:head>
	<title>地理统计 · HotSpot</title>
</svelte:head>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.stats.read 权限" />
{:else}
	<div class="flex flex-col gap-4 max-w-5xl">
		<!-- 全站存量 -->
		<section aria-label="全站存量" class="grid grid-cols-2 sm:grid-cols-4 gap-3">
			{#if data}
				<div class="rounded-lg border border-border-default bg-surface-primary p-4 shadow-raised-sm">
					<p class="text-xs text-text-secondary">用户</p>
					<p class="mt-1 text-xl font-semibold text-text-primary leading-none">{fmt(data.totals.user)}</p>
				</div>
				<div class="rounded-lg border border-border-default bg-surface-primary p-4 shadow-raised-sm">
					<p class="text-xs text-text-secondary">群组</p>
					<p class="mt-1 text-xl font-semibold text-text-primary leading-none">{fmt(data.totals.group)}</p>
				</div>
				<div class="rounded-lg border border-border-default bg-surface-primary p-4 shadow-raised-sm">
					<p class="text-xs text-text-secondary">动态</p>
					<p class="mt-1 text-xl font-semibold text-text-primary leading-none">{fmt(data.totals.moment)}</p>
				</div>
				<div class="rounded-lg border border-border-default bg-surface-primary p-4 shadow-raised-sm">
					<p class="text-xs text-text-secondary">合计</p>
					<p class="mt-1 text-xl font-semibold text-accent-default leading-none">{fmt(data.totals.total)}</p>
				</div>
			{:else if loading}
				<div class="col-span-2 sm:col-span-4"><InlineLoading label="统计中…" /></div>
			{/if}
		</section>

		<!-- 筛选：粒度 + 时间窗 -->
		<section class="flex flex-col gap-2" aria-label="筛选">
			<TabBar
				items={LEVELS}
				value={String(level)}
				onChange={(id) => { level = Number(id); }}
			/>
			<TabBar
				items={RANGES}
				value={range}
				onChange={(id) => { range = id; }}
			/>
		</section>

		{#if loading}
			<InlineLoading label="统计中…" />
		{:else if error}
			<LoadError message={error} onRetry={() => { reloadKey += 1; }} />
		{:else if data}
			<p class="text-xs text-text-secondary">统计生成于 {formatDateTime(data.generated_at)} · 弱一致快照</p>

			<!-- 区域排名 -->
			<section aria-label="区域排名" class="flex flex-col gap-2">
				<h2 class="text-sm font-medium text-text-primary">区域排名（S2 网格 · {LEVELS.find((l) => Number(l.id) === level)?.label ?? level}）</h2>
				{#if data.regions.length === 0}
					<EmptyState title="暂无地理数据" description="当前筛选条件下没有带位置的公开数据" />
				{:else}
					<div class={TABLE_WRAP}>
						<table class={TABLE}>
							<thead>
								<tr>
									<th scope="col" class={TH}>#</th>
									<th scope="col" class={TH}>网格中心</th>
									<th scope="col" class={TH}>用户</th>
									<th scope="col" class={TH}>群组</th>
									<th scope="col" class={TH}>动态</th>
									<th scope="col" class={TH}>合计</th>
								</tr>
							</thead>
							<tbody>
								{#each data.regions as r, i (r.cell_token)}
									<tr class={TR}>
										<td class={TD}>{i + 1}</td>
										<td class={TD} title={r.cell_token}>{coord(r.center.latitude)}°, {coord(r.center.longitude)}°</td>
										<td class={TD}>{fmt(r.breakdown.user)}</td>
										<td class={TD}>{fmt(r.breakdown.group)}</td>
										<td class={TD}>{fmt(r.breakdown.moment)}</td>
										<td class={TD + ' font-medium'}>{fmt(r.total_count)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>

			<!-- 热门地点 -->
			<section aria-label="热门地点" class="flex flex-col gap-2">
				<h2 class="text-sm font-medium text-text-primary">热门地点（按地点名聚合）</h2>
				{#if data.places.length === 0}
					<EmptyState title="暂无地点数据" description="当前筛选条件下没有非空地点名" />
				{:else}
					<div class={TABLE_WRAP}>
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

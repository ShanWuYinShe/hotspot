<script lang="ts">
	import { goto } from '$app/navigation';
	import {
		Flag, KeyRound, LayoutGrid, MapPinned, Megaphone, ScrollText, Settings2, ShieldCheck, Tags, Users, UsersRound
	} from '@lucide/svelte';
	import { admin, hasAdminPermission } from '@/stores/admin.svelte';
	import { platform } from '@/stores/platform.svelte';
	import { adminListReports, adminStatsGeo } from '@/api';
	import MetricRow from '@/components/data-display/MetricRow.svelte';
	import type { MetricItem } from '@/components/data-display/MetricRow.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import Skeleton from '@/components/data-display/Skeleton.svelte';
	import { ROUTES } from '@/lib/routes';
	import { formatDateTime } from '@/lib/utils';
	import AdminModuleCard from './shared/AdminModuleCard.svelte';
	import AdminRecentAudit from './shared/AdminRecentAudit.svelte';

	// 平台管理概览（R13 真仪表盘）：管理员身份行 + 全站 KPI（用户/群组/动态/待处理举报）
	// + 9 个模块入口（图标砖 + 说明 + 待办角标 + 箭头）+ 最近审计活动。
	// 数据全部来自既有 admin API（stats/geo 一次拿全站存量、reports/fetch 拿 pending total、
	// audit/fetch 取最近 6 条），只读计数，不新增接口；无权限/失败时降级为「—」占位，不报错。
	// 模块入口按权限降级为禁用态（与后端权限键一一对应）。

	const MODULES = [
		{ href: `${ROUTES.admin}/stats`, label: '地理统计', desc: '全站分布与热门地点', icon: MapPinned, perm: 'admin.stats.read' },
		{ href: `${ROUTES.admin}/users`, label: '用户管理', desc: '搜索、封禁与解封用户', icon: Users, perm: 'admin.users.read' },
		{ href: `${ROUTES.admin}/groups`, label: '群组管理', desc: '搜索与解散群组', icon: UsersRound, perm: 'admin.groups.read' },
		{ href: `${ROUTES.admin}/reports`, label: '举报审核', desc: '裁决用户举报', icon: Flag, perm: 'admin.reports.read' },
		{ href: `${ROUTES.admin}/tags`, label: '标签管理', desc: '屏蔽与恢复标签', icon: Tags, perm: 'admin.tags.read' },
		{ href: `${ROUTES.admin}/announcements`, label: '系统公告', desc: '发布与管理公告', icon: Megaphone, perm: 'admin.announcements.read' },
		{ href: `${ROUTES.admin}/rbac`, label: '角色权限', desc: 'RBAC 角色与管理员', icon: KeyRound, perm: 'admin.rbac.manage' },
		{ href: `${ROUTES.admin}/audit`, label: '审计日志', desc: '平台操作留痕查询', icon: ScrollText, perm: 'admin.audit.read' },
		{ href: `${ROUTES.admin}/config`, label: '系统配置', desc: '应用配置与功能开关', icon: Settings2, perm: 'admin.app_config.read' }
	];

	const canStats = $derived(hasAdminPermission('admin.stats.read'));
	const canReports = $derived(hasAdminPermission('admin.reports.read'));
	const isSuper = $derived(admin.permissions.includes('*'));
	const isMobile = $derived(platform.platform === 'mobile');
	const roleLabel = $derived(
		isSuper ? '超级管理员' : (admin.roles.map((r) => r.name).join('、') || '平台管理员')
	);

	// ── KPI（全站存量 + 待处理举报；缺数据时占位，失败不报错）──
	let totals = $state<{ user: number; group: number; moment: number } | null>(null);
	let statsAt = $state('');
	// 举报待处理数：既是 KPI，也是「举报审核」模块卡的待办角标
	let pendingReports = $state<number | null>(null);
	let kpiLoading = $state(true);

	$effect(() => {
		const needStats = canStats;
		const needReports = canReports;
		if (!needStats && !needReports) {
			kpiLoading = false;
			return;
		}
		let cancelled = false;
		kpiLoading = true;
		const jobs: Promise<void>[] = [];
		if (needStats) {
			jobs.push(
				adminStatsGeo({ level: 8, limit: 1 })
					.then((res) => {
						if (cancelled) return;
						if (res.code === 0 && res.data) {
							totals = res.data.totals;
							statsAt = res.data.generated_at;
						}
					})
					.catch(() => {})
			);
		}
		if (needReports) {
			jobs.push(
				adminListReports({ status: 'pending', target_type: null, pagination: { limit: 1 } })
					.then((res) => {
						if (cancelled) return;
						if (res.code === 0 && res.data) pendingReports = res.data.total;
					})
					.catch(() => {})
			);
		}
		void Promise.all(jobs).then(() => {
			if (!cancelled) kpiLoading = false;
		});
		return () => {
			cancelled = true;
		};
	});

	const fmt = (n: number): string => n.toLocaleString('zh-CN');

	const kpiItems = $derived.by<MetricItem[]>(() => {
		const items: MetricItem[] = [];
		if (canStats) {
			items.push({ label: '用户', value: totals ? fmt(totals.user) : '—' });
			items.push({ label: '群组', value: totals ? fmt(totals.group) : '—' });
			items.push({ label: '动态', value: totals ? fmt(totals.moment) : '—' });
		}
		if (canReports) {
			// 移动端条宽紧张：「待处理举报」会截断，降为「待处理」，语义由下方举报模块承接
			items.push({
				label: isMobile ? '待处理' : '待处理举报',
				value: pendingReports === null ? '—' : fmt(pendingReports),
				tone: pendingReports ? 'warning' : 'default'
			});
		}
		return items;
	});
</script>

<svelte:head>
	<title>管理概览 · HotSpot</title>
</svelte:head>

<div class="flex w-full flex-col gap-6">
	<!-- 概览卡：管理员身份行（安静）→ 发丝分隔 → KPI 数字行（母题 A：数字与秩序） -->
	<section class="rounded-lg bg-surface-island p-5 shadow-xs" aria-label="平台概览">
		<div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
			<div class="flex min-w-0 items-center gap-2">
				<ShieldCheck class="size-4 shrink-0 text-accent-default" strokeWidth={1.75} aria-hidden="true" />
				<p class="truncate text-xs text-text-secondary">
					<span class="font-medium text-text-primary">{roleLabel}</span>
					<span class="mx-1.5" aria-hidden="true">·</span>
					{isSuper ? '拥有全部平台权限' : `持有 ${admin.permissions.length} 项权限`}
				</p>
			</div>
			{#if statsAt && !kpiLoading}
				<p class="max-sm:hidden text-xxs tabular-nums text-text-secondary">统计更新于 {formatDateTime(statsAt)}</p>
			{/if}
		</div>

		{#if kpiLoading}
			<div
				class="mt-4 flex items-stretch gap-6 border-t border-border-subtle pt-4"
				role="status"
				aria-busy="true"
				aria-label="加载平台指标…"
			>
				{#each Array(isMobile ? 4 : kpiItems.length || 4) as _, i (i)}
					<div class="flex min-w-0 flex-1 flex-col gap-2">
						<Skeleton class="rounded-sm" style="height: 24px; width: 44%;" />
						<Skeleton class="rounded-sm" style="height: 11px; width: 30%;" />
					</div>
				{/each}
			</div>
		{:else if kpiItems.length > 0}
			<div class="mt-4 border-t border-border-subtle pt-4">
				<MetricRow variant="strip" items={kpiItems} />
			</div>
		{/if}
	</section>

	<!-- 模块入口：桌面 3 列 / 平板 2 列 / 移动单列（断点与 platform 阈值同刻度 sm=600 / lg=1280） -->
	<section aria-label="管理模块" class="flex flex-col">
		<SectionHead title="管理模块" icon={LayoutGrid} />
		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
			{#each MODULES as m (m.href)}
				{@const isReports = m.href === `${ROUTES.admin}/reports`}
				<AdminModuleCard
					label={m.label}
					description={m.desc}
					icon={m.icon}
					allowed={hasAdminPermission(m.perm)}
					badge={isReports && pendingReports ? `待处理 ${pendingReports}` : undefined}
					onOpen={() => void goto(m.href)}
				/>
			{/each}
		</div>
	</section>

	<!-- 最近审计活动（自取 limit 6；无 admin.audit.read 时整块不渲染） -->
	<AdminRecentAudit />
</div>

<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { admin, ensureAdminLoaded, hasAdminPermission } from '@/stores/admin.svelte';
	import { adminListReports } from '@/api';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import { Shield } from '@lucide/svelte';
	import { ROUTES } from '@/lib/routes';
	import { smartBack } from '@/lib/smart-back';

	// 平台管理后台布局（/admin）：统一守卫（登录 → rbac/me 权限）+ 模块导航。
	// 非平台管理员（10200 not_admin）在此收敛为无权限空态，模块页不再各自判 admin。
	// 入口：桌面侧栏条件区块（对齐鸿蒙端 AdminOverviewPage「桌面侧边栏可进入」的预期）。

	let { children }: { children: Snippet } = $props();

	const device = $derived(platform.platform);

	$effect(() => {
		if (auth.status === 'authenticated') ensureAdminLoaded();
	});

	// 举报 Tab 计数（第二轮审查 r89：切换子页不丢待办感知）：与概览卡同源 reports/fetch total
	let pendingReports = $state<number | null>(null);
	$effect(() => {
		if (auth.status !== 'authenticated' || !admin.isAdmin || !hasAdminPermission('admin.reports.read')) return;
		void adminListReports({ status: 'pending', target_type: null, pagination: { limit: 1 } })
			.then((res) => {
				if (res.code === 0 && res.data) pendingReports = res.data.total;
			})
			.catch(() => {});
	});

	const MODULES = [
		{ id: 'overview', label: '概览', href: ROUTES.admin },
		{ id: 'stats', label: '地理统计', href: `${ROUTES.admin}/stats` },
		{ id: 'users', label: '用户', href: `${ROUTES.admin}/users` },
		{ id: 'groups', label: '群组', href: `${ROUTES.admin}/groups` },
		{ id: 'reports', label: '举报', href: `${ROUTES.admin}/reports` },
		{ id: 'tags', label: '标签', href: `${ROUTES.admin}/tags` },
		{ id: 'announcements', label: '公告', href: `${ROUTES.admin}/announcements` },
		{ id: 'rbac', label: '角色权限', href: `${ROUTES.admin}/rbac` },
		{ id: 'audit', label: '审计日志', href: `${ROUTES.admin}/audit` },
		{ id: 'config', label: '系统配置', href: `${ROUTES.admin}/config` }
	];

	const pathname = $derived(page.url.pathname);
	// 按 URL 第二段精确匹配模块，避免 /admin/users 被 /admin 前缀误命中为「概览」
	const activeId = $derived((pathname.split('/').filter(Boolean)[1] ?? 'overview') || 'overview');
</script>

<svelte:head>
	<title>管理后台 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		<PageHeader
			title="管理后台"
			subtitle="平台管理（仅平台管理员可见）"
			sticky
			titleSize={device === 'desktop' ? 'xl' : '2xl'}
			onBack={() => smartBack()}
		>
			{#snippet below()}
				{#if admin.isAdmin}
					<div class="mt-3 overflow-x-auto pr-1">
						<TabBar
							items={MODULES.map((m) => ({ id: m.id, label: m.label, count: m.id === 'reports' && pendingReports ? pendingReports : undefined }))}
							value={activeId}
							onChange={(id) => {
								const target = MODULES.find((m) => m.id === id);
								if (target && target.href !== pathname) void goto(target.href);
							}}
						/>
					</div>
				{/if}
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if auth.status === 'loading'}
		<InlineLoading label="加载登录状态…" />
	{:else if auth.status !== 'authenticated'}
		<NeedsAuthNotice title="登录后访问" description="平台管理后台仅对平台管理员开放" />
	{:else if !admin.loaded}
		<InlineLoading label="校验平台管理员权限…" />
	{:else if !admin.isAdmin}
		<EmptyState title="无访问权限" description="当前账号不是平台管理员，如需权限请联系超级管理员。" bare>
			{#snippet icon()}
				<Shield class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else}
		{@render children()}
	{/if}
</StandardPageLayout>

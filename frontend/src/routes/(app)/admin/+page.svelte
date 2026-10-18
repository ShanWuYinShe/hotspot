<script lang="ts">
	import { goto } from '$app/navigation';
	import { Users, UsersRound, Flag, Tags, Megaphone, ShieldCheck, KeyRound, ScrollText, Settings2, MapPinned, ChevronRight } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import { admin, hasAdminPermission } from '@/stores/admin.svelte';
	import { adminListReports } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 平台管理概览：当前管理员身份（角色 + 权限键）+ 模块入口卡片网格。
	// 无对应 read 权限的模块降级为禁用态（与后端权限键一一对应）。

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

	const isSuper = $derived(admin.permissions.includes('*'));
	// 移动端模块入口用行式列表（图标左 + 文案中 + chevron 右），一屏可见 8 个；
	// 平板/桌面保留卡片网格（信息密度足够，卡片是更好的视觉分组）。
	const isMobile = $derived(platform.platform === 'mobile');

	// 举报审核「待处理」角标（UI 复审 C1）：reports/fetch(status=pending, limit=1) 的
	// total（不受分页窗口影响）；失败或无权限时不渲染（诚实降级，不留死徽标）。
	let pendingReports = $state<number | null>(null);
	$effect(() => {
		if (!hasAdminPermission('admin.reports.read')) return;
		void adminListReports({ status: 'pending', target_type: null, pagination: { limit: 1 } })
			.then((res) => {
				if (res.code === 0 && res.data) pendingReports = res.data.total;
			})
			.catch(() => {});
	});
</script>

<svelte:head>
	<title>管理概览 · HotSpot</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-5xl flex-col gap-4">
	<!-- 身份卡：当前平台管理员 -->
	<section class="rounded-lg border border-border-default bg-surface-primary p-5 shadow-raised-sm" aria-label="管理员身份">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<div class="flex items-center gap-3 min-w-0">
				<div class="flex size-9 shrink-0 items-center justify-center rounded-full icon-badge">
					<ShieldCheck class="size-4.5 text-accent-default" strokeWidth={1.75} />
				</div>
				<div class="min-w-0">
					<p class="text-sm font-medium text-text-primary">
						{isSuper ? '超级管理员' : (admin.roles.map((r) => r.name).join('、') || '平台管理员')}
					</p>
					<p class="mt-0.5 text-xs text-text-secondary">
						{isSuper ? '拥有全部平台权限（PII 字段仍按显式授权展示）' : `持有 ${admin.permissions.length} 项权限`}
					</p>
				</div>
			</div>
		</div>
		{#if !isSuper && admin.permissions.length > 0}
			<div class="mt-3 flex flex-wrap gap-1.5">
				{#each admin.permissions as key (key)}
					<span class="badge bg-surface-subtle text-text-secondary font-mono text-xxs">{key}</span>
				{/each}
			</div>
		{/if}
	</section>

	<!-- 模块入口：移动端行式列表（同「我的」数据入口的密度与语言），平板/桌面卡片网格 -->
	{#if isMobile}
		<section aria-label="管理模块">
			<ListContainer bordered inset>
				{#each MODULES as m (m.href)}
					{@const allowed = hasAdminPermission(m.perm)}
					<ListItem
						onClick={allowed ? () => void goto(m.href) : undefined}
						ariaLabel={m.label}
						class="px-4 py-3 {allowed ? '' : 'opacity-faint pointer-events-none'}"
					>
						<span class="flex items-center gap-3">
							<span class="flex size-9 shrink-0 items-center justify-center rounded-full icon-badge">
								<m.icon class="size-4.5 text-accent-default" strokeWidth={1.75} />
							</span>
							<span class="min-w-0 flex-1">
								<span class="flex items-center gap-1.5">
									<span class="block truncate text-sm font-medium text-text-primary">{m.label}</span>
									{#if m.href === `${ROUTES.admin}/reports` && pendingReports}
										<span class="inline-flex h-5 items-center justify-center leading-none rounded-full bg-status-warning-bg text-status-warning px-2 text-xxs font-medium">待处理 {pendingReports}</span>
									{/if}
								</span>
								<span class="mt-0.5 block truncate text-xs text-text-secondary">{allowed ? m.desc : '无访问权限'}</span>
							</span>
							<ChevronRight class="size-4 shrink-0 text-text-secondary opacity-subtle" strokeWidth={1.5} />
						</span>
					</ListItem>
				{/each}
			</ListContainer>
		</section>
	{:else}
		<section class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4" aria-label="管理模块">
			{#each MODULES as m (m.href)}
				{@const allowed = hasAdminPermission(m.perm)}
				<button
					type="button"
					disabled={!allowed}
					onclick={() => void goto(m.href)}
					title={allowed ? m.label : '无对应权限'}
					class="hover-btn flex flex-col items-start gap-2 rounded-lg border border-border-default bg-surface-primary p-4 text-left shadow-raised-sm focus-ring transition-all disabled:opacity-faint disabled:pointer-events-none {allowed ? 'hover:border-accent-default' : ''}"
				>
					<span class="flex size-9 items-center justify-center rounded-full icon-badge shrink-0">
						<m.icon class="size-4.5 text-accent-default" strokeWidth={1.75} />
					</span>
					<span>
						<span class="flex items-center gap-1.5">
							<span class="block text-sm font-medium text-text-primary">{m.label}</span>
							{#if m.href === `${ROUTES.admin}/reports` && pendingReports}
								<span class="inline-flex h-5 items-center justify-center leading-none rounded-full bg-status-warning-bg text-status-warning px-2 text-xxs font-medium">待处理 {pendingReports}</span>
							{/if}
						</span>
						<span class="mt-0.5 block text-xs text-text-secondary">{allowed ? m.desc : '无访问权限'}</span>
					</span>
				</button>
			{/each}
		</section>
	{/if}
</div>

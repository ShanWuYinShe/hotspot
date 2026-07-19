<script lang="ts">
  import { onMount } from 'svelte';
  import { systemService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import {
    adminUsersService,
    adminGroupsService,
    adminReportsService,
    adminAuditService
  } from '$lib/api/services/admin';
  import AdminPageHeader from '$lib/components/common/admin-page-header.svelte';
  import { t } from '$lib/i18n';
  import { PAGE_SIZE } from '$lib/config/pagination';
  import Users from 'lucide-svelte/icons/users';
  import UsersRound from 'lucide-svelte/icons/users-round';
  import FileText from 'lucide-svelte/icons/file-text';
  import Flag from 'lucide-svelte/icons/flag';

  let userCount = $state('--');
  let groupCount = $state('--');
  let pendingReports = $state('--');
  let auditCount = $state('--');
  let healthStatus: { key: string; label: string; ok: boolean }[] = $state([]);

  async function loadStats() {
    const [usersRes, groupsRes, reportsRes, auditRes, healthRes] = await Promise.allSettled([
      adminUsersService.search({ pagination: { limit: PAGE_SIZE.admin } }),
      adminGroupsService.search({ pagination: { limit: PAGE_SIZE.admin } }),
      adminReportsService.search({ status: 'pending', pagination: { limit: PAGE_SIZE.admin } }),
      adminAuditService.getList({ pagination: { limit: PAGE_SIZE.admin } }),
      systemService.health()
    ]);

    if (usersRes.status === 'fulfilled' && isSuccess(usersRes.value)) {
      const items = usersRes.value.data?.items;
      userCount = items ? `${items.length}${items.length >= PAGE_SIZE.admin ? '+' : ''}` : '--';
    }
    if (groupsRes.status === 'fulfilled' && isSuccess(groupsRes.value)) {
      const items = groupsRes.value.data?.items;
      groupCount = items ? `${items.length}${items.length >= PAGE_SIZE.admin ? '+' : ''}` : '--';
    }
    if (reportsRes.status === 'fulfilled' && isSuccess(reportsRes.value)) {
      const items = reportsRes.value.data?.items;
      pendingReports = items ? `${items.length}${items.length >= PAGE_SIZE.admin ? '+' : ''}` : '--';
    }
    if (auditRes.status === 'fulfilled' && isSuccess(auditRes.value)) {
      const items = auditRes.value.data?.items;
      auditCount = items ? `${items.length}${items.length >= PAGE_SIZE.admin ? '+' : ''}` : '--';
    }
    if (healthRes.status === 'fulfilled' && isSuccess(healthRes.value)) {
      const h = healthRes.value.data;
      healthStatus = [
        { key: 'api', label: t('admin.serviceApi'), ok: h?.status === 'healthy' },
        { key: 'db', label: t('admin.serviceDb'), ok: h?.status === 'healthy' },
        { key: 'cache', label: t('admin.serviceCache'), ok: h?.status !== 'unhealthy' }
      ];
    }
  }

  const fallbackServices = $derived([
    t('admin.serviceApi'),
    t('admin.serviceDb'),
    t('admin.serviceCache')
  ]);

  const statCards = $derived([
    { icon: Users, label: t('admin.totalUsers'), value: userCount, sub: t('admin.totalUsersSub') },
    {
      icon: UsersRound,
      label: t('admin.totalGroups'),
      value: groupCount,
      sub: t('admin.totalGroupsSub')
    },
    {
      icon: Flag,
      label: t('admin.pendingReports'),
      value: pendingReports,
      sub: t('admin.pendingReportsSub')
    },
    { icon: FileText, label: t('admin.auditLogs'), value: auditCount, sub: t('admin.auditLogsSub') }
  ]);

  const quickActions = $derived([
    {
      href: '/admin/users',
      icon: Users,
      title: t('admin.userManagement'),
      desc: t('admin.userManagementDesc')
    },
    {
      href: '/admin/reports',
      icon: Flag,
      title: t('admin.reportReview'),
      desc: t('admin.reportReviewDesc')
    }
  ]);

  onMount(() => {
    loadStats();
  });
</script>

<svelte:head>
  <title>{t('admin.pageTitle')}</title>
</svelte:head>

<AdminPageHeader title={t('admin.dashboardTitle')} description={t('admin.dashboardDesc')} />

<div class="page-pad">
  <div class="page-shell page-shell--wide space-y-6">
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {#each statCards as stat (stat.label)}
        <div
          class="flex flex-col items-center justify-center rounded-[var(--radius-lg)] py-5"
          style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
        >
          <div
            class="mb-2 flex items-center justify-center rounded-full"
            style="width: 36px; height: 36px; border: 1px solid var(--border-theme);"
          >
            <stat.icon size={16} strokeWidth={1.75} style="color: var(--brand);" />
          </div>
          <span class="text-h1" style="color: var(--text-primary); font-weight: 700"
            >{stat.value}</span
          >
          <span
            class="text-caption"
            style="color: var(--text-primary); font-weight: 500; margin-top: 2px">{stat.label}</span
          >
          <span class="text-xs" style="color: var(--text-secondary); margin-top: 2px"
            >{stat.sub}</span
          >
        </div>
      {/each}
    </div>

    <div class="grid gap-6 lg:grid-cols-2">
      <!-- 快捷操作 -->
      <div
        class="rounded-[var(--radius-lg)] p-5"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <div class="text-title mb-1" style="font-weight: 600; color: var(--text-primary)">
          {t('admin.quickActions')}
        </div>
        <div class="text-meta mb-4" style="color: var(--text-secondary)">{t('admin.quickActionsDesc')}</div>
        <div style="height: 1px; background: var(--border-theme); margin-bottom: 16px;"></div>
        <div class="space-y-3">
          {#each quickActions as action (action.href)}
            <a
              href={action.href}
              class="flex items-center gap-3 rounded-[var(--radius-md)] p-4 transition-all duration-200"
              style="background: var(--surface-bg); border: 1px solid var(--border-active);"
            >
              <div
                class="flex items-center justify-center rounded-full"
                style="width: 36px; height: 36px; border: 1px solid var(--border-theme);"
              >
                <action.icon size={16} strokeWidth={1.75} style="color: var(--brand);" />
              </div>
              <div>
                <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
                  {action.title}
                </p>
                <p class="text-meta" style="color: var(--text-secondary)">{action.desc}</p>
              </div>
            </a>
          {/each}
        </div>
      </div>

      <!-- 系统状态 -->
      <div
        class="rounded-[var(--radius-lg)] p-5"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <div class="text-title mb-1" style="font-weight: 600; color: var(--text-primary)">
          {t('admin.systemStatus')}
        </div>
        <div class="text-meta mb-4" style="color: var(--text-secondary)">{t('admin.systemStatusDesc')}</div>
        <div style="height: 1px; background: var(--border-theme); margin-bottom: 16px;"></div>
        <div class="space-y-4">
          {#if healthStatus.length > 0}
            {#each healthStatus as svc (svc.key)}
              <div
                class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-3"
                style="background: var(--surface-bg); border: 1px solid var(--border-active);"
              >
                <span class="text-body-lg" style="color: var(--text-primary)">{svc.label}</span>
                <div class="flex items-center gap-2">
                  <span
                    style="width: 8px; height: 8px; border-radius: 50%; background: {svc.ok
                      ? 'var(--theme-success)'
                      : 'var(--theme-error)'};"
                  ></span>
                  <span
                    class="text-body"
                    style="color: {svc.ok ? 'var(--theme-success)' : 'var(--theme-error)'}"
                    >{svc.ok ? t('admin.statusNormal') : t('admin.statusAbnormal')}</span
                  >
                </div>
              </div>
            {/each}
          {:else}
            {#each fallbackServices as service (service)}
              <div
                class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-3"
                style="background: var(--surface-bg); border: 1px solid var(--border-active);"
              >
                <span class="text-body-lg" style="color: var(--text-primary)">{service}</span>
                <span class="text-body" style="color: var(--text-secondary)">{t('admin.statusChecking')}</span>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>

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
  import PageHeader from '$lib/components/layout/page-header.svelte';
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
      adminUsersService.search({ pagination: { limit: 50 } }),
      adminGroupsService.search({ pagination: { limit: 50 } }),
      adminReportsService.search({ status: 'pending', pagination: { limit: 50 } }),
      adminAuditService.getList({ pagination: { limit: 50 } }),
      systemService.health()
    ]);

    if (usersRes.status === 'fulfilled' && isSuccess(usersRes.value)) {
      const items = usersRes.value.data?.items;
      userCount = items ? `${items.length}${items.length >= 50 ? '+' : ''}` : '--';
    }
    if (groupsRes.status === 'fulfilled' && isSuccess(groupsRes.value)) {
      const items = groupsRes.value.data?.items;
      groupCount = items ? `${items.length}${items.length >= 50 ? '+' : ''}` : '--';
    }
    if (reportsRes.status === 'fulfilled' && isSuccess(reportsRes.value)) {
      const items = reportsRes.value.data?.items;
      pendingReports = items ? `${items.length}${items.length >= 50 ? '+' : ''}` : '--';
    }
    if (auditRes.status === 'fulfilled' && isSuccess(auditRes.value)) {
      const items = auditRes.value.data?.items;
      auditCount = items ? `${items.length}${items.length >= 50 ? '+' : ''}` : '--';
    }
    if (healthRes.status === 'fulfilled' && isSuccess(healthRes.value)) {
      const h = healthRes.value.data;
      healthStatus = [
        { key: 'api', label: 'API 服务', ok: h?.status === 'healthy' },
        { key: 'db', label: '数据库', ok: h?.status === 'healthy' },
        { key: 'cache', label: '缓存服务', ok: h?.status !== 'unhealthy' }
      ];
    }
  }

  onMount(() => {
    loadStats();
  });
</script>

<svelte:head>
  <title>管理后台 - HotSpot</title>
</svelte:head>

<div class="space-y-6">
  <PageHeader title="管理后台" description="系统管理概览" />

  <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
    {#each [{ icon: Users, label: '用户总数', value: userCount, sub: '注册用户数量' }, { icon: UsersRound, label: '群组总数', value: groupCount, sub: '创建的群组数量' }, { icon: Flag, label: '待处理举报', value: pendingReports, sub: '等待审核的举报' }, { icon: FileText, label: '审计日志', value: auditCount, sub: '今日操作记录' }] as stat}
      <div
        class="flex flex-col items-center justify-center rounded-[22px] py-5"
        style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
      >
        <div
          class="mb-2 flex items-center justify-center rounded-full"
          style="width: 36px; height: 36px; box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
        >
          <stat.icon size={16} strokeWidth={1.75} style="color: var(--brand);" />
        </div>
        <span style="font-size: 22px; color: var(--text-primary); font-weight: 700;"
          >{stat.value}</span
        >
        <span
          style="font-size: 11px; color: var(--text-primary); font-weight: 500; margin-top: 2px;"
          >{stat.label}</span
        >
        <span style="font-size: 10px; color: var(--text-secondary); margin-top: 2px;"
          >{stat.sub}</span
        >
      </div>
    {/each}
  </div>

  <div class="grid gap-6 lg:grid-cols-2">
    <!-- 快捷操作 -->
    <div
      class="rounded-[22px] p-5"
      style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
    >
      <div class="mb-1" style="font-size: 16px; font-weight: 600; color: var(--text-primary);">
        快捷操作
      </div>
      <div class="mb-4" style="font-size: 12px; color: var(--text-secondary);">
        常用管理功能入口
      </div>
      <div style="height: 1px; background: var(--border-theme); margin-bottom: 16px;"></div>
      <div class="space-y-3">
        {#each [{ href: '/admin/users', icon: Users, title: '用户管理', desc: '搜索、封禁、解封用户' }, { href: '/admin/reports', icon: Flag, title: '举报审核', desc: '处理用户举报' }] as action}
          <a
            href={action.href}
            class="flex items-center gap-3 rounded-[18px] p-4 transition-all duration-200"
            style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-active);"
          >
            <div
              class="flex items-center justify-center rounded-full"
              style="width: 36px; height: 36px; box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
            >
              <action.icon size={16} strokeWidth={1.75} style="color: var(--brand);" />
            </div>
            <div>
              <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">
                {action.title}
              </p>
              <p style="font-size: 12px; color: var(--text-secondary);">{action.desc}</p>
            </div>
          </a>
        {/each}
      </div>
    </div>

    <!-- 系统状态 -->
    <div
      class="rounded-[22px] p-5"
      style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
    >
      <div class="mb-1" style="font-size: 16px; font-weight: 600; color: var(--text-primary);">
        系统状态
      </div>
      <div class="mb-4" style="font-size: 12px; color: var(--text-secondary);">服务运行状态</div>
      <div style="height: 1px; background: var(--border-theme); margin-bottom: 16px;"></div>
      <div class="space-y-4">
        {#if healthStatus.length > 0}
          {#each healthStatus as svc (svc.key)}
            <div
              class="flex items-center justify-between rounded-[18px] px-4 py-3"
              style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-active);"
            >
              <span style="font-size: 14px; color: var(--text-primary);">{svc.label}</span>
              <div class="flex items-center gap-2">
                <span
                  style="width: 8px; height: 8px; border-radius: 50%; background: {svc.ok
                    ? 'var(--neu-success)'
                    : '#EF4444'};"
                ></span>
                <span style="font-size: 13px; color: {svc.ok ? 'var(--neu-success)' : '#EF4444'};"
                  >{svc.ok ? '正常' : '异常'}</span
                >
              </div>
            </div>
          {/each}
        {:else}
          {#each ['API 服务', '数据库', '缓存服务'] as service}
            <div
              class="flex items-center justify-between rounded-[18px] px-4 py-3"
              style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-active);"
            >
              <span style="font-size: 14px; color: var(--text-primary);">{service}</span>
              <span style="font-size: 13px; color: var(--text-secondary);">检测中...</span>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  </div>
</div>

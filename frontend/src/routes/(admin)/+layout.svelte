<script lang="ts">
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { authStore } from '$lib/stores/auth.svelte';
  import { adminPermissionsStore } from '$lib/stores/admin-permissions.svelte';
  import AppShell from '$lib/components/layout/app-shell.svelte';
  import AdminSidebar from '$lib/components/common/admin-sidebar.svelte';
  import AdminSubNav from '$lib/components/common/admin-subnav.svelte';
  import { Button } from '$lib/components/ui/button';
  import Shield from 'lucide-svelte/icons/shield';

  let { children } = $props();
</script>

{#if authStore.isAuthenticated && adminPermissionsStore.isAdmin}
  <AppShell>
    {#snippet sidebar()}
      <AdminSidebar />
    {/snippet}
    {#snippet subnav()}
      <AdminSubNav />
    {/snippet}
    {@render children()}
  </AppShell>
{:else if authStore.isAuthenticated && adminPermissionsStore.isLoading}
  <AppShell>
    <div class="flex flex-col items-center justify-center gap-3 py-20">
      <div
        class="h-8 w-8 animate-spin rounded-full"
        style="border: 4px solid var(--border-theme); border-top-color: var(--brand);"
      ></div>
      <p class="text-body" style="color: var(--text-secondary)">
        {t('admin.checkingPermissions')}
      </p>
    </div>
  </AppShell>
{:else if authStore.isAuthenticated && adminPermissionsStore.isForbidden}
  <AppShell>
    <div class="flex flex-col items-center justify-center py-20 text-center">
      <div
        class="mb-4 flex items-center justify-center rounded-full"
        style="width: 56px; height: 56px; background: var(--theme-error-subtle);"
      >
        <Shield size={24} strokeWidth={1.5} style="color: var(--theme-error);" />
      </div>
      <h2 class="text-h1" style="font-weight: 600; color: var(--text-primary)">
        {t('admin.accessDenied')}
      </h2>
      <p class="text-body" style="color: var(--text-secondary); margin-top: 8px">
        {t('admin.accessDeniedDescription')}
      </p>
      <Button variant="flat" class="mt-6" onclick={() => goto('/')}>
        {t('common.backHome')}
      </Button>
    </div>
  </AppShell>
{:else if authStore.isAuthenticated && adminPermissionsStore.isError}
  <AppShell>
    <div class="flex flex-col items-center justify-center py-20 text-center">
      <div
        class="mb-4 flex items-center justify-center rounded-full"
        style="width: 56px; height: 56px; border: 1px solid var(--border-theme);"
      >
        <Shield size={24} strokeWidth={1.5} style="color: var(--text-secondary);" />
      </div>
      <h2 class="text-h1" style="font-weight: 600; color: var(--text-primary)">
        {t('admin.permissionCheckFailed')}
      </h2>
      <p class="text-body" style="color: var(--text-secondary); margin-top: 8px">
        {t('admin.permissionCheckFailedDescription')}
      </p>
      <div class="mt-6 flex gap-3">
        <Button variant="flat" onclick={() => adminPermissionsStore.refresh()}>
          {t('common.retry')}
        </Button>
        <Button variant="flat" onclick={() => goto('/')}>
          {t('common.backHome')}
        </Button>
      </div>
    </div>
  </AppShell>
{:else if authStore.isRestoring}
  <AppShell>
    <div class="flex flex-col items-center justify-center gap-3 py-20">
      <div
        class="h-8 w-8 animate-spin rounded-full"
        style="border: 4px solid var(--border-theme); border-top-color: var(--brand);"
      ></div>
      <p class="text-body" style="color: var(--text-secondary)">{t('auth.restoringSession')}</p>
    </div>
  </AppShell>
{:else}
  <!-- 未登录：复用 AppShell 的统一整页登录占位（Lock 图标），与 (protected) 路由完全一致 -->
  <AppShell requireAuth>{''}</AppShell>
{/if}

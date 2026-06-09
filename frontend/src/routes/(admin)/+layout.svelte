<script lang="ts">
  import { browser } from '$app/environment';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { adminPermissionsStore } from '$lib/stores/admin-permissions.svelte';
  import { cn } from '$lib/utils';
  import Navbar from '$lib/components/common/navbar.svelte';
  import { Button } from '$lib/components/ui/button';
  import Users from 'lucide-svelte/icons/users';
  import UsersRound from 'lucide-svelte/icons/users-round';
  import Megaphone from 'lucide-svelte/icons/megaphone';
  import FileText from 'lucide-svelte/icons/file-text';
  import Flag from 'lucide-svelte/icons/flag';
  import Shield from 'lucide-svelte/icons/shield';
  import Settings from 'lucide-svelte/icons/settings';
  import LayoutDashboard from 'lucide-svelte/icons/layout-dashboard';

  let { children } = $props();

  let sidebarOpen = $state(false);

  function closeSidebar() {
    sidebarOpen = false;
  }

  const adminNavItems = $derived([
    { href: '/admin', label: t('admin.overview'), icon: LayoutDashboard },
    { href: '/admin/users', label: t('admin.users'), icon: Users },
    { href: '/admin/groups', label: t('admin.groups'), icon: UsersRound },
    { href: '/admin/announcements', label: t('admin.announcements'), icon: Megaphone },
    { href: '/admin/audit', label: t('admin.audit'), icon: FileText },
    { href: '/admin/reports', label: t('admin.reports'), icon: Flag },
    { href: '/admin/rbac', label: t('admin.rbac'), icon: Shield },
    { href: '/admin/config', label: t('admin.config'), icon: Settings }
  ]);

  function isActive(href: string): boolean {
    const pathname = page.url.pathname;
    if (href === '/admin') {
      return pathname === '/admin';
    }
    return pathname.startsWith(href);
  }

  $effect(() => {
    if (browser && authStore.initialized && !authStore.isAuthenticated && !authStore.isRestoring) {
      uiStore.saveRedirectUrl();
    }
  });
  function handleLoginClick() {
    uiStore.openLoginModalWithRedirect({});
  }
</script>

{#if authStore.isAuthenticated && adminPermissionsStore.isAdmin}
  <div class="min-h-screen" style="background: var(--page-bg);">
    <Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />

    <div class="app-shell-frame flex">
      <!-- Admin sidebar -->
      <aside
        class="hidden w-64 shrink-0 md:block"
        style="background: var(--surface-bg); border-right: 1px solid var(--border-theme);"
      >
        <nav class="space-y-1 p-4">
          <h2
            class="mb-4 px-3"
            style="font-size: 18px; font-weight: 600; color: var(--text-primary);"
          >
            {t('nav.adminPanel')}
          </h2>
          {#each adminNavItems as item (item.href)}
            <a
              href={item.href}
              class={cn(
                'flex items-center gap-3 rounded-[18px] px-3 py-2.5 text-sm font-medium transition-all duration-200',
                isActive(item.href) ? 'neu-pressed' : 'hover:opacity-70'
              )}
              style={isActive(item.href) ? 'color: var(--brand);' : 'color: var(--text-secondary);'}
            >
              <item.icon size={18} strokeWidth={1.75} />
              {item.label}
            </a>
          {/each}
        </nav>
      </aside>

      <!-- Mobile sidebar overlay -->
      {#if sidebarOpen}
        <button
          type="button"
          class="fixed inset-0 z-40 h-full w-full cursor-pointer border-none p-0 md:hidden"
          style="background: var(--theme-overlay-scene);"
          onclick={closeSidebar}
          onkeydown={(e) => e.key === 'Escape' && closeSidebar()}
          aria-label={t('nav.closeSidebar')}
        />
        <aside
          class="app-mobile-sidebar fixed left-0 z-50 w-64 md:hidden"
          style="background: var(--surface-bg); border-right: 1px solid var(--border-theme);"
        >
          <nav
            class="space-y-1 p-4"
            style="padding-bottom: calc(0.75rem + env(safe-area-inset-bottom, 0px));"
          >
            <h2
              class="mb-4 px-3"
              style="font-size: 18px; font-weight: 600; color: var(--text-primary);"
            >
              {t('nav.adminPanel')}
            </h2>
            {#each adminNavItems as item (item.href)}
              <a
                href={item.href}
                class={cn(
                  'flex items-center gap-3 rounded-[18px] px-3 py-2.5 text-sm font-medium transition-all duration-200',
                  isActive(item.href) ? 'neu-pressed' : 'hover:opacity-70'
                )}
                style={isActive(item.href)
                  ? 'color: var(--brand);'
                  : 'color: var(--text-secondary);'}
                onclick={closeSidebar}
              >
                <item.icon size={18} strokeWidth={1.75} />
                {item.label}
              </a>
            {/each}
          </nav>
        </aside>
      {/if}

      <!-- Main content -->
      <main class="mobile-app-main pb-safe-compact md:mx-4 md:rounded-[32px] md:p-6">
        <div class="mobile-page-shell md:mx-auto md:max-w-6xl">
          {@render children()}
        </div>
      </main>
    </div>
  </div>
{:else if authStore.isAuthenticated && adminPermissionsStore.isLoading}
  <!-- 管理员权限验证中 -->
  <div class="min-h-screen" style="background: var(--page-bg);">
    <Navbar onMenuClick={() => {}} />
    <div
      class="flex min-h-[calc(100dvh-var(--topbar-height-mobile))] items-center justify-center sm:min-h-[calc(100dvh-var(--topbar-height-desktop))]"
    >
      <div class="flex flex-col items-center gap-3">
        <div
          class="h-8 w-8 animate-spin rounded-full"
          style="border: 4px solid var(--border-theme); border-top-color: var(--brand);"
        ></div>
        <p style="font-size: 13px; color: var(--text-secondary);">
          {t('admin.checkingPermissions')}
        </p>
      </div>
    </div>
  </div>
{:else if authStore.isAuthenticated && adminPermissionsStore.isForbidden}
  <!-- 已登录但非管理员 -->
  <div class="min-h-screen" style="background: var(--page-bg);">
    <Navbar onMenuClick={() => {}} />
    <div class="app-shell-frame flex">
      <main class="mobile-app-main pb-safe-compact md:mx-4 md:rounded-[32px] md:p-6">
        <div class="mobile-page-shell md:mx-auto md:max-w-6xl">
          <div class="flex flex-col items-center justify-center py-20 text-center">
            <div
              class="mb-4 flex items-center justify-center rounded-full"
              style="width: 56px; height: 56px; background: var(--theme-error-subtle);"
            >
              <Shield size={24} strokeWidth={1.5} style="color: var(--theme-error);" />
            </div>
            <h2 style="font-size: 20px; font-weight: 600; color: var(--text-primary);">
              {t('admin.accessDenied')}
            </h2>
            <p style="font-size: 13px; color: var(--text-secondary); margin-top: 8px;">
              {t('admin.accessDeniedDescription')}
            </p>
            <Button variant="flat" class="mt-6" onclick={() => goto('/')}>
              {t('common.backHome')}
            </Button>
          </div>
        </div>
      </main>
    </div>
  </div>
{:else if authStore.isAuthenticated && adminPermissionsStore.isError}
  <div class="min-h-screen" style="background: var(--page-bg);">
    <Navbar onMenuClick={() => {}} />
    <div class="app-shell-frame flex">
      <main class="mobile-app-main pb-safe-compact md:mx-4 md:rounded-[32px] md:p-6">
        <div class="mobile-page-shell md:mx-auto md:max-w-6xl">
          <div class="flex flex-col items-center justify-center py-20 text-center">
            <div
              class="mb-4 flex items-center justify-center rounded-full"
              style="width: 56px; height: 56px; box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
            >
              <Shield size={24} strokeWidth={1.5} style="color: var(--text-secondary);" />
            </div>
            <h2 style="font-size: 20px; font-weight: 600; color: var(--text-primary);">
              {t('admin.permissionCheckFailed')}
            </h2>
            <p style="font-size: 13px; color: var(--text-secondary); margin-top: 8px;">
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
        </div>
      </main>
    </div>
  </div>
{:else if authStore.isRestoring}
  <!-- 登录态恢复中 -->
  <div class="min-h-screen" style="background: var(--page-bg);">
    <Navbar onMenuClick={() => {}} />
    <div
      class="flex min-h-[calc(100dvh-var(--topbar-height-mobile))] items-center justify-center sm:min-h-[calc(100dvh-var(--topbar-height-desktop))]"
    >
      <div class="flex flex-col items-center gap-3">
        <div
          class="h-8 w-8 animate-spin rounded-full"
          style="border: 4px solid var(--border-theme); border-top-color: var(--brand);"
        ></div>
        <p style="font-size: 13px; color: var(--text-secondary);">{t('auth.restoringSession')}</p>
      </div>
    </div>
  </div>
{:else}
  <!-- 未登录 -->
  <div class="min-h-screen" style="background: var(--page-bg);">
    <Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />
    <div class="app-shell-frame flex">
      <main class="mobile-app-main pb-safe-compact md:mx-4 md:rounded-[32px] md:p-6">
        <div class="mobile-page-shell md:mx-auto md:max-w-6xl">
          <div class="flex flex-col items-center justify-center py-20 text-center">
            <div
              class="mb-4 flex items-center justify-center rounded-full"
              style="width: 56px; height: 56px; box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
            >
              <Shield size={24} strokeWidth={1.5} style="color: var(--text-secondary);" />
            </div>
            <h2 style="font-size: 20px; font-weight: 600; color: var(--text-primary);">
              {t('auth.loginRequired')}
            </h2>
            <p style="font-size: 13px; color: var(--text-secondary); margin-top: 8px;">
              {t('admin.loginRequiredDescription')}
            </p>
            <Button variant="flat" class="mt-6" onclick={handleLoginClick}>
              {t('common.loginNow')}
            </Button>
          </div>
        </div>
      </main>
    </div>
  </div>
{/if}

<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { authStore } from '$lib/stores/auth.svelte';
  import { adminPermissionsStore } from '$lib/stores/admin-permissions.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import { messageStore } from '$lib/stores/message.svelte';
  import AnchorEntry from '$lib/components/common/anchor-entry.svelte';
  import {
    discoverNavItems,
    personalNavItems,
    isNavActive,
    type NavItem
  } from '$lib/config/navigation';
  import Shield from 'lucide-svelte/icons/shield';
  import LogIn from 'lucide-svelte/icons/log-in';

  const currentPathname = $derived(page.url.pathname);

  function isActive(item: NavItem): boolean {
    return isNavActive(item.href, item.exact, currentPathname);
  }

  function badgeCount(item: NavItem): number {
    if (!authStore.isAuthenticated) return 0;
    if (item.badge === 'notifications') return notificationStore.unreadCount;
    if (item.badge === 'messages') return messageStore.totalUnreadCount;
    return 0;
  }

  // 未登录时隐藏整个个人中心分区（全部 requiresAuth），
  // 改为展示单一「登录」入口，保证未登录用户有显眼的登录通道。
  const showPersonalSection = $derived(authStore.isAuthenticated);
  const showLoginEntry = $derived(!authStore.isAuthenticated);

  function handleNavClick(item: NavItem) {
    void goto(item.href);
  }

  function handleLoginClick() {
    uiStore.openLoginModalWithRedirect({});
  }

  const adminActive = $derived(
    currentPathname === '/admin' || currentPathname.startsWith('/admin/')
  );

  // ── 当前页面是否在侧边栏导航体系之外 ──
  const allNavItems = $derived([...discoverNavItems, ...personalNavItems]);
  const isOutsideSidebar = $derived(
    !allNavItems.some(item => isNavActive(item.href, item.exact, currentPathname))
  );

  const currentOutsideLabel = $derived.by(() => {
    if (!isOutsideSidebar) return null;
    if (currentPathname === '/profile' || currentPathname.startsWith('/profile/')) return t('nav.profile');
    if (currentPathname.startsWith('/settings')) return t('nav.settings');
    return null;
  });
</script>

<!-- Desktop sidebar — 192px fixed -->
<aside class="flex h-full min-h-0 flex-shrink-0 flex-col overflow-hidden" style="width: 192px;">
  <!-- Scrollable nav region -->
  <div class="flex min-h-0 flex-1 flex-col overflow-y-auto no-scrollbar">
    <!-- Section 1: Discovery nav -->
    <div class="flex flex-col" style="gap: 2px;">
      {#each discoverNavItems as item (item.href)}
        <button
          class="desktop-sidebar__nav-item {isActive(item) ? 'active' : ''}"
          onclick={() => handleNavClick(item)}
          aria-current={isActive(item) ? 'page' : undefined}
        >
          <item.icon size={17} strokeWidth={isActive(item) ? 2 : 1.6} />
          <span class="text-sm" class:font-medium={isActive(item)}>
            {t(item.labelKey)}
          </span>
        </button>
      {/each}
    </div>

    <!-- Divider -->
    <div class="my-3" style="height: 1px; background: var(--border-theme);"></div>

    {#if showPersonalSection}
      <!-- Section 2 label -->
      <p
        class="text-xs mb-2 px-3"
        style="font-weight: 600; letter-spacing: 0.12em; text-transform: uppercase; color: var(--text-secondary); opacity: 0.65"
      >
        {t('sidebar.personalCenter')}
      </p>

      <!-- Section 2: Personal nav -->
      <div class="flex flex-col" style="gap: 2px;">
        {#each personalNavItems as item (item.href)}
          <button
            class="desktop-sidebar__nav-item {isActive(item) ? 'active' : ''}"
            onclick={() => handleNavClick(item)}
            aria-current={isActive(item) ? 'page' : undefined}
          >
            <item.icon size={17} strokeWidth={isActive(item) ? 2 : 1.6} />
            <span class="text-sm" class:font-medium={isActive(item)}>
              {t(item.labelKey)}
            </span>
            {#if badgeCount(item) > 0}
              <span
                class="text-caption ml-auto flex items-center justify-center rounded-full text-on-accent"
                style="min-width: 20px; height: 20px; font-weight: 600; background: var(--brand)"
              >
                {badgeCount(item) > 99 ? '99+' : badgeCount(item)}
              </span>
            {/if}
          </button>
        {/each}
      </div>
    {:else if showLoginEntry}
      <!-- 未登录：个人中心分区折叠为单一「登录」入口 -->
      <div class="flex flex-col" style="gap: 2px;">
        <button
          class="desktop-sidebar__nav-item"
          onclick={handleLoginClick}
        >
          <LogIn size={17} strokeWidth={1.6} />
          <span class="text-sm">{t('nav.loginRegister')}</span>
        </button>
      </div>
    {/if}

    <!-- Admin section -->
    {#if authStore.isAuthenticated && adminPermissionsStore.isAdmin}
      <div class="mt-4">
        <button
          class="desktop-sidebar__nav-item {adminActive ? 'active' : ''}"
          onclick={() => void goto('/admin')}
          aria-current={adminActive ? 'page' : undefined}
        >
          <Shield size={17} strokeWidth={adminActive ? 2 : 1.6} />
          <span class="text-sm" class:font-medium={adminActive}>
            {t('sidebar.adminPanel')}
          </span>
        </button>
      </div>
    {/if}

    {#if currentOutsideLabel}
      <div class="sidebar-current-page">
        <span class="sidebar-current-page__dot"></span>
        <span>{currentOutsideLabel}</span>
      </div>
    {/if}
  </div>

  <!-- Location Anchor — pinned to the bottom-left corner of the sidebar -->
  <div class="mt-3 flex-shrink-0 pt-3" style="border-top: 1px solid var(--border-theme);">
    <AnchorEntry variant="card" />
  </div>
</aside>

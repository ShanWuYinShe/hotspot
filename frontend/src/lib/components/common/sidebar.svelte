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

  function handleNavClick(item: NavItem) {
    if (item.requiresAuth && !authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({ url: item.href });
      return;
    }
    void goto(item.href);
  }

  const adminActive = $derived(
    currentPathname === '/admin' || currentPathname.startsWith('/admin/')
  );
</script>

<!-- Figma Make DesktopLayout Sidebar — 192px fixed -->
<aside class="flex flex-shrink-0 flex-col" style="width: 192px;">
  <!-- Scrollable nav region -->
  <div class="flex min-h-0 flex-1 flex-col overflow-y-auto no-scrollbar">
    <!-- Section 1: Discovery nav -->
    <div class="flex flex-col" style="gap: 2px;">
      {#each discoverNavItems as item (item.href)}
        <button
          class="desktop-sidebar__nav-item {isActive(item) ? 'active' : ''}"
          onclick={() => handleNavClick(item)}
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

    <!-- Section 2 label -->
    <p
      class="mb-2 px-3"
      style="font-size: 10px; font-weight: 600; letter-spacing: 0.12em; text-transform: uppercase; color: var(--text-secondary); opacity: 0.65;"
    >
      {t('sidebar.personalCenter')}
    </p>

    <!-- Section 2: Personal nav -->
    <div class="flex flex-col" style="gap: 2px;">
      {#each personalNavItems as item (item.href)}
        <button
          class="desktop-sidebar__nav-item {isActive(item) ? 'active' : ''}"
          onclick={() => handleNavClick(item)}
        >
          <item.icon size={17} strokeWidth={isActive(item) ? 2 : 1.6} />
          <span class="text-sm" class:font-medium={isActive(item)}>
            {t(item.labelKey)}
          </span>
          {#if badgeCount(item) > 0}
            <span
              class="ml-auto flex items-center justify-center rounded-full text-on-accent"
              style="min-width: 20px; height: 20px; font-size: 11px; font-weight: 600; background: var(--brand);"
            >
              {badgeCount(item) > 99 ? '99+' : badgeCount(item)}
            </span>
          {/if}
        </button>
      {/each}
    </div>

    <!-- Admin section -->
    {#if authStore.isAuthenticated && adminPermissionsStore.isAdmin}
      <div class="mt-4">
        <button
          class="desktop-sidebar__nav-item {adminActive ? 'active' : ''}"
          onclick={() => void goto('/admin')}
        >
          <Shield size={17} strokeWidth={adminActive ? 2 : 1.6} />
          <span class="text-sm" class:font-medium={adminActive}>
            {t('sidebar.adminPanel')}
          </span>
        </button>
      </div>
    {/if}
  </div>

  <!-- Location Anchor — pinned to the bottom-left corner of the sidebar -->
  <div class="mt-3 flex-shrink-0 pt-3" style="border-top: 1px solid var(--border-theme);">
    <AnchorEntry variant="card" />
  </div>
</aside>

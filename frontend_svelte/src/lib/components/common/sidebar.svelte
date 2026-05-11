<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { authStore } from '$lib/stores/auth.svelte';
  import { adminPermissionsStore } from '$lib/stores/admin-permissions.svelte';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import { messageStore } from '$lib/stores/message.svelte';
  import LocationPickerModal from '$lib/components/common/location-picker-modal.svelte';
  import Home from 'lucide-svelte/icons/home';
  import Image from 'lucide-svelte/icons/image';
  import UsersRound from 'lucide-svelte/icons/users-round';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import Bell from 'lucide-svelte/icons/bell';
  import UserPlus from 'lucide-svelte/icons/user-plus';
  import Heart from 'lucide-svelte/icons/heart';
  import UserCheck from 'lucide-svelte/icons/user-check';
  import Navigation2 from 'lucide-svelte/icons/navigation-2';
  import Shield from 'lucide-svelte/icons/shield';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';

  interface NavItem {
    href: string;
    label: string;
    icon: typeof Home;
    requiresAuth?: boolean;
    exact?: boolean;
  }

  const mainNavItems: NavItem[] = [
    { href: '/', label: t('sidebar.home'), icon: Home },
    { href: '/users/nearby', label: t('sidebar.nearbyPeople'), icon: UsersRound },
    { href: '/groups/nearby', label: t('sidebar.nearbyGroups'), icon: UsersRound },
    { href: '/moments/nearby', label: t('sidebar.nearbyMoments'), icon: Image }
  ];

  const personalNavItems: NavItem[] = [
    { href: '/chat', label: t('sidebar.messages'), icon: MessageCircle, requiresAuth: true },
    { href: '/notifications', label: t('sidebar.notifications'), icon: Bell, requiresAuth: true },
    {
      href: '/moments',
      label: t('sidebar.myMoments'),
      icon: Image,
      requiresAuth: true,
      exact: true
    },
    {
      href: '/groups',
      label: t('sidebar.myGroups'),
      icon: UsersRound,
      requiresAuth: true,
      exact: true
    },
    { href: '/friends', label: t('sidebar.friends'), icon: Heart, requiresAuth: true },
    { href: '/follows', label: t('sidebar.following'), icon: UserCheck, requiresAuth: true },
    { href: '/followers', label: t('sidebar.followers'), icon: UserPlus, requiresAuth: true }
  ];

  const currentPathname = $derived(page.url.pathname);

  function isActive(item: NavItem): boolean {
    if (item.href === '/') return currentPathname === '/';
    if (item.exact) return currentPathname === item.href;
    return currentPathname === item.href || currentPathname.startsWith(`${item.href}/`);
  }

  function handleNavClick(item: NavItem) {
    if (item.requiresAuth && !authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({ url: item.href });
      return;
    }
    void goto(item.href);
  }

  let showLocationPicker = $state(false);
</script>

<!-- Figma Make DesktopLayout Sidebar — 240px fixed -->
<aside class="flex flex-shrink-0 flex-col" style="width: 240px;">
  <!-- Scrollable nav region -->
  <div class="flex flex-1 flex-col overflow-y-auto no-scrollbar min-h-0">
    <!-- Section 1: Discovery nav -->
    <div class="flex flex-col" style="gap: 2px;">
      {#each mainNavItems as item (item.href)}
        <button
          class="desktop-sidebar__nav-item {isActive(item) ? 'active' : ''}"
          onclick={() => handleNavClick(item)}
        >
          <item.icon
            size={18}
            strokeWidth={isActive(item) ? 2 : 1.5}
            color={isActive(item) ? 'var(--neu-active-text)' : 'var(--neu-text-muted)'}
          />
          <span class="text-sm" class:font-medium={isActive(item)}>
            {item.label}
          </span>
          {#if authStore.isAuthenticated && item.href === '/notifications' && notificationStore.unreadCount > 0}
            <span
              class="ml-auto flex items-center justify-center rounded-full neu-active-text"
              style="min-width: 20px; height: 20px; font-size: 11px; font-weight: 600; background: var(--neu-accent-primary);"
            >
              {notificationStore.unreadCount > 99 ? '99+' : notificationStore.unreadCount}
            </span>
          {/if}
          {#if authStore.isAuthenticated && item.href === '/chat' && messageStore.totalUnreadCount > 0}
            <span
              class="ml-auto flex items-center justify-center rounded-full neu-active-text"
              style="min-width: 20px; height: 20px; font-size: 11px; font-weight: 600; background: var(--neu-accent-primary);"
            >
              {messageStore.totalUnreadCount > 99 ? '99+' : messageStore.totalUnreadCount}
            </span>
          {/if}
          {#if !authStore.isAuthenticated && item.requiresAuth}
            <span
              class="ml-auto rounded-full px-2 py-0.5 text-xs"
              style="background: var(--neu-border); color: var(--neu-text-muted);"
            >
              {t('sidebar.login')}
            </span>
          {/if}
        </button>
      {/each}
    </div>

    <!-- Divider -->
    <div class="my-3" style="height: 1px; background: var(--neu-border);"></div>

    <!-- Section 2 label -->
    <p
      class="mb-2 px-3"
      style="font-size: 10px; font-weight: 600; letter-spacing: 0.12em; text-transform: uppercase; color: var(--neu-text-muted); opacity: 0.65;"
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
          <item.icon
            size={18}
            strokeWidth={isActive(item) ? 2 : 1.5}
            color={isActive(item) ? 'var(--neu-active-text)' : 'var(--neu-text-muted)'}
          />
          <span class="text-sm" class:font-medium={isActive(item)}>
            {item.label}
          </span>
          {#if authStore.isAuthenticated && item.href === '/notifications' && notificationStore.unreadCount > 0}
            <span
              class="ml-auto flex items-center justify-center rounded-full neu-active-text"
              style="min-width: 20px; height: 20px; font-size: 11px; font-weight: 600; background: var(--neu-accent-primary);"
            >
              {notificationStore.unreadCount > 99 ? '99+' : notificationStore.unreadCount}
            </span>
          {/if}
          {#if authStore.isAuthenticated && item.href === '/chat' && messageStore.totalUnreadCount > 0}
            <span
              class="ml-auto flex items-center justify-center rounded-full neu-active-text"
              style="min-width: 20px; height: 20px; font-size: 11px; font-weight: 600; background: var(--neu-accent-primary);"
            >
              {messageStore.totalUnreadCount > 99 ? '99+' : messageStore.totalUnreadCount}
            </span>
          {/if}
          {#if !authStore.isAuthenticated && item.requiresAuth}
            <span
              class="ml-auto rounded-full px-2 py-0.5 text-xs"
              style="background: var(--neu-border); color: var(--neu-text-muted);"
            >
              {t('sidebar.login')}
            </span>
          {/if}
        </button>
      {/each}
    </div>

    <!-- Admin section -->
    {#if authStore.isAuthenticated && adminPermissionsStore.isAdmin}
      <div class="mt-4">
        <button
          class="desktop-sidebar__nav-item {currentPathname.startsWith('/admin') ? 'active' : ''}"
          onclick={() => void goto('/admin')}
        >
          <Shield
            size={18}
            strokeWidth={currentPathname.startsWith('/admin') ? 2 : 1.5}
            color={currentPathname.startsWith('/admin')
              ? 'var(--neu-active-text)'
              : 'var(--neu-text-muted)'}
          />
          <span class="text-sm" class:font-medium={currentPathname.startsWith('/admin')}>
            {t('sidebar.adminPanel')}
          </span>
        </button>
      </div>
    {/if}
  </div>

  <!-- Location Anchor — pinned to bottom -->
  <div class="flex-shrink-0 pt-3">
    <button
      class="flex w-full items-center gap-3 rounded-[16px] text-left outline-none transition-shadow duration-200"
      style="padding: 10px 12px; background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: 5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light);"
      onclick={() => (showLocationPicker = true)}
    >
      <!-- Icon circle -->
      <div
        class="flex flex-shrink-0 items-center justify-center rounded-full"
        style="width: 34px; height: 34px; background: var(--neu-accent-primary); box-shadow: 0 3px 10px rgba(0,0,0,0.15);"
      >
        <Navigation2 size={14} strokeWidth={2} color="var(--neu-active-text)" />
      </div>
      <!-- Text -->
      <div class="min-w-0 flex-1">
        <p
          style="font-size: 10px; line-height: 1.4; color: var(--neu-text-muted); letter-spacing: 0.06em;"
        >
          {t('sidebar.browseAnchor')}
        </p>
        <p style="font-size: 13px; line-height: 1.4; color: var(--neu-text);">
          {mapAnchorStore.displayName}
        </p>
      </div>
      <ChevronRight
        size={14}
        strokeWidth={1.5}
        color="var(--neu-text-muted)"
        class="flex-shrink-0"
      />
    </button>
  </div>
</aside>

<LocationPickerModal bind:open={showLocationPicker} />

<script lang="ts">
  import { fade, fly } from 'svelte/transition';
  import { page } from '$app/state';
  import { goto, afterNavigate } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { mode, toggleMode } from 'mode-watcher';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { adminPermissionsStore } from '$lib/stores/admin-permissions.svelte';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import { messageStore } from '$lib/stores/message.svelte';
  import { THEME_FAMILY_OPTIONS, themeStore, type ThemeFamily } from '$lib/stores/theme.svelte';
  import Avatar from '$lib/components/common/avatar.svelte';
  import {
    discoverNavItems,
    personalNavItems,
    adminNavItems,
    isNavActive,
    type NavItem
  } from '$lib/config/navigation';
  import Settings from 'lucide-svelte/icons/settings';
  import LogIn from 'lucide-svelte/icons/log-in';
  import Sun from 'lucide-svelte/icons/sun';
  import Moon from 'lucide-svelte/icons/moon';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';

  const open = $derived(uiStore.showNavDrawer);
  const currentPathname = $derived(page.url.pathname);
  const currentMode = $derived(mode.current);
  const isAdmin = $derived(authStore.isAuthenticated && adminPermissionsStore.isAdmin);

  function isActive(item: NavItem): boolean {
    return isNavActive(item.href, item.exact, currentPathname);
  }

  function badgeCount(item: NavItem): number {
    if (!authStore.isAuthenticated) return 0;
    if (item.badge === 'notifications') return notificationStore.unreadCount;
    if (item.badge === 'messages') return messageStore.totalUnreadCount;
    return 0;
  }

  function close() {
    uiStore.closeNavDrawer();
  }

  function handleNavClick(item: NavItem) {
    if (item.requiresAuth && !authStore.isAuthenticated) {
      close();
      uiStore.openLoginModalWithRedirect({ url: item.href });
      return;
    }
    close();
    void goto(item.href);
  }

  function goProfile() {
    close();
    void goto('/profile');
  }

  function goSettings() {
    close();
    void goto('/settings');
  }

  function openLogin() {
    close();
    uiStore.openLoginModalWithRedirect({});
  }

  function selectThemeFamily(family: ThemeFamily) {
    themeStore.setThemeFamily(family);
  }

  // 路由变化后自动关闭抽屉（含浏览器返回）。
  afterNavigate(() => {
    if (uiStore.showNavDrawer) uiStore.closeNavDrawer();
  });

  // 切换到桌面时关闭抽屉（桌面以侧边栏为导航中枢）。
  $effect(() => {
    if (platformStore.isDesktop && uiStore.showNavDrawer) uiStore.closeNavDrawer();
  });
</script>

{#snippet navSection(label: string, items: NavItem[])}
  <p
    class="mb-2 px-3"
    style="font-size: 10px; font-weight: 600; letter-spacing: 0.12em; text-transform: uppercase; color: var(--text-secondary); opacity: 0.65;"
  >
    {label}
  </p>
  <div class="mb-4 flex flex-col" style="gap: 2px;">
    {#each items as item (item.href)}
      <button
        class="desktop-sidebar__nav-item {isActive(item) ? 'active' : ''}"
        onclick={() => handleNavClick(item)}
      >
        <item.icon size={18} strokeWidth={isActive(item) ? 2 : 1.6} />
        <span class="text-sm" class:font-medium={isActive(item)}>{t(item.labelKey)}</span>
        {#if badgeCount(item) > 0}
          <span
            class="ml-auto flex items-center justify-center rounded-full text-on-accent"
            style="min-width: 20px; height: 20px; font-size: 11px; font-weight: 600; background: var(--brand);"
          >
            {badgeCount(item) > 99 ? '99+' : badgeCount(item)}
          </span>
        {:else if !authStore.isAuthenticated && item.requiresAuth}
          <span
            class="ml-auto rounded-full px-2 py-0.5 text-xs"
            style="background: var(--border-theme); color: var(--text-secondary);"
          >
            {t('sidebar.login')}
          </span>
        {/if}
      </button>
    {/each}
  </div>
{/snippet}

{#if open}
  <!-- Overlay -->
  <button
    type="button"
    class="fixed inset-0 z-[60] h-full w-full cursor-default border-none p-0"
    style="background: var(--theme-overlay-scene);"
    aria-label={t('nav.closeSidebar')}
    transition:fade={{ duration: 180 }}
    onclick={close}
  ></button>

  <!-- Drawer panel -->
  <aside
    class="fixed top-0 left-0 z-[61] flex h-[100dvh] flex-col"
    style="width: min(86vw, 320px); background: var(--surface-bg); border-right: 1px solid var(--border-theme);"
    transition:fly={{ x: -340, duration: 240 }}
  >
    <!-- Header: profile / login -->
    <div
      class="flex-shrink-0 px-4 pt-4 pb-3"
      style="padding-top: calc(1rem + env(safe-area-inset-top, 0px));"
    >
      {#if authStore.isAuthenticated}
        <button
          type="button"
          class="flex w-full items-center gap-3 rounded-[16px] p-2 text-left outline-none transition-transform duration-150 active:scale-[0.98]"
          style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          onclick={goProfile}
        >
          <Avatar
            src={authStore.currentAccount?.avatar_url || ''}
            name={authStore.currentAccount?.nickname || 'Profile'}
            alt={authStore.currentAccount?.nickname || 'Profile'}
            size={44}
          />
          <span class="min-w-0 flex-1">
            <span
              class="block truncate"
              style="font-size: 14px; font-weight: 600; color: var(--text-primary);"
            >
              {authStore.currentAccount?.nickname || t('nav.profile')}
            </span>
            <span class="block truncate" style="font-size: 11px; color: var(--text-secondary);">
              {t('nav.profile')}
            </span>
          </span>
          <ChevronRight size={16} color="var(--text-secondary)" />
        </button>
      {:else}
        <button
          type="button"
          class="flex w-full items-center justify-center gap-2 rounded-[16px] px-4 py-3 outline-none transition-transform duration-150 active:scale-[0.98]"
          style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 3px 10px var(--glow);"
          onclick={openLogin}
        >
          <LogIn size={16} strokeWidth={2} />
          <span style="font-size: 14px; font-weight: 600;">{t('common.loginNow')}</span>
        </button>
      {/if}
    </div>

    <!-- Scrollable nav -->
    <nav class="min-h-0 flex-1 overflow-y-auto no-scrollbar px-3 pb-2">
      {@render navSection(t('nav.discover'), discoverNavItems)}
      {@render navSection(t('sidebar.personalCenter'), personalNavItems)}
      {#if isAdmin}
        {@render navSection(t('sidebar.adminPanel'), adminNavItems)}
      {/if}
    </nav>

    <!-- Footer: settings + theme -->
    <div
      class="flex-shrink-0 px-3 pt-3"
      style="border-top: 1px solid var(--border-theme); padding-bottom: calc(0.75rem + env(safe-area-inset-bottom, 0px));"
    >
      <button class="desktop-sidebar__nav-item" onclick={goSettings}>
        <Settings size={18} strokeWidth={1.6} />
        <span class="text-sm">{t('sidebar.settingsLabel')}</span>
      </button>
      <button class="desktop-sidebar__nav-item" onclick={toggleMode}>
        {#if currentMode === 'dark'}
          <Sun size={18} strokeWidth={1.6} />
          <span class="text-sm">{t('sidebar.light')}</span>
        {:else}
          <Moon size={18} strokeWidth={1.6} />
          <span class="text-sm">{t('sidebar.dark')}</span>
        {/if}
      </button>
      <div class="mt-2 flex items-center gap-1.5 px-3 pb-1">
        {#each THEME_FAMILY_OPTIONS as option (option.family)}
          <button
            type="button"
            class="flex-1 rounded-full px-2 py-1.5 text-center outline-none transition-colors"
            style="font-size: 11px; font-weight: {themeStore.themeFamily === option.family
              ? 600
              : 400}; background: {themeStore.themeFamily === option.family
              ? 'var(--theme-accent-soft)'
              : 'transparent'}; color: {themeStore.themeFamily === option.family
              ? 'var(--brand)'
              : 'var(--text-secondary)'}; border: 1px solid {themeStore.themeFamily ===
            option.family
              ? 'var(--theme-accent-soft-border)'
              : 'var(--border-theme)'};"
            onclick={() => selectThemeFamily(option.family)}
          >
            {t(option.labelKey)}
          </button>
        {/each}
      </div>
    </div>
  </aside>
{/if}
